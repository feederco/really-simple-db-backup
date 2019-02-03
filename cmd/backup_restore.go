package cmd

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strconv"
	"time"

	"github.com/cheggaaa/pb"
	"github.com/digitalocean/godo"
	"github.com/feederco/really-simple-db-backup/pkg"
	minio "github.com/minio/minio-go"
)

func backupMysqlDownloadAndPrepare(
	fromHostname string,
	restoreTimestamp string,
	backupBucket string,
	existingVolumeID string,
	existingBackupDirectory string,
	digitalOceanClient *pkg.DigitalOceanClient,
	minioClient *minio.Client,
) (string, string, *godo.Volume, error) {
	var err error

	err = prerequisites(configStruct.PersistentStorage)
	if err != nil {
		return "", "", nil, err
	}

	err = backupPrerequisites()
	if err != nil {
		return "", "", nil, err
	}

	sinceTimestamp := time.Now()
	if restoreTimestamp != "" {
		sinceTimestamp, err = parseBackupTimestamp(restoreTimestamp)
		if err != nil {
			return "", "", nil, errors.New("Incorrect timestamp passed in: " + restoreTimestamp + " (error: " + err.Error() + ")")
		}
	}

	pkg.Log.Println("Listing backups since", sinceTimestamp.Format(time.RFC3339))

	// Game plan:

	// - List all backups we need
	allBackups, err := listAllBackups(fromHostname, backupBucket, minioClient)
	if err != nil {
		return "", "", nil, err
	}

	backupFiles := findRelevantBackupsUpTo(sinceTimestamp, allBackups)
	if len(backupFiles) == 0 {
		return "", "", nil, errors.New("No backup found to restore from")
	}

	pkg.Log.Printf("%d backup files found\n", len(backupFiles))

	totalSizeInBytes := int64(0)
	for _, backupFile := range backupFiles {
		totalSizeInBytes += backupFile.Size
	}

	// - Create & mount volume to house backup
	sizeInGigaBytes := bytesToGigaBytes(totalSizeInBytes)
	aDecentSizeInGigaBytes := sizeInGigaBytes * 5

	var volume *godo.Volume
	var mountDirectory string

	volume, mountDirectory, err = createAndMountVolumeForUse(
		"mysql-restore-",
		aDecentSizeInGigaBytes,
		digitalOceanClient,
		existingVolumeID,
		existingBackupDirectory,
	)

	if err != nil {
		pkg.ErrorLog.Println("Could not create mount volume.", err)
		return "", mountDirectory, volume, nil
	}

	restoreDirectory := path.Join(mountDirectory, "really-simple-db-restore")
	err = os.MkdirAll(restoreDirectory, 0755)
	if err != nil {
		pkg.ErrorLog.Println("Could not create directory to house backup files.")
		return restoreDirectory, mountDirectory, volume, nil
	}

	pkg.Log.Println("Downloading and extracting backups")

	// - Download full backup and incremental pieces
	err = downloadBackups(backupFiles, restoreDirectory, backupBucket, minioClient)
	if err != nil {
		pkg.ErrorLog.Println("Could not download backups!")
		return restoreDirectory, mountDirectory, volume, err
	}

	pkg.Log.Println("Decompressing backups")

	numberOfCPUs := runtime.NumCPU()

	// - Decompress files with as many cores as possible
	_, err = pkg.PerformCommand(
		"xtrabackup",
		"--decompress",
		"--target-dir",
		restoreDirectory,
		"--parallel",
		strconv.FormatInt(int64(numberOfCPUs), 10),
		"--remove-original",
	)

	pkg.Log.Println("Preparing backups")

	// - Prepare backup
	_, err = pkg.PerformCommand(
		"xtrabackup",
		"--prepare",
		"--target-dir",
		restoreDirectory,
	)

	if err != nil {
		pkg.AlertError(configStruct.Alerting, "Could not create backup.", err)
		return restoreDirectory, mountDirectory, volume, backupCleanup(volume, mountDirectory, digitalOceanClient)
	}

	pkg.Log.Println("Prepare completed!")
	return restoreDirectory, mountDirectory, volume, nil
}

func backupMysqlFinalizeRestore(
	restoreDirectory string,
	mysqlDataPath string,
	mountDirectory string,
	volume *godo.Volume,
	digitalOceanClient *pkg.DigitalOceanClient,
	minioClient *minio.Client,
) error {
	var err error

	pkg.Log.Println("Starting to put everything back")
	pkg.Log.Println("Warning: Removing everything in the MySQL data directory")

	// We try to run this command. If it fails, we just run xtrabackup --copy-back anyway.
	// It will error if the directory is not empty
	pkg.PerformCommand("mv", mysqlDataPath, "/tmp/")

	copyCompletedChannel := make(chan bool)
	go pkg.ReportProgressOnCopy(restoreDirectory, mysqlDataPath, copyCompletedChannel)

	// - Move to MySQL data directory
	_, err = pkg.PerformCommand(
		"xtrabackup",
		"--copy-back",
		"--target-dir",
		restoreDirectory,
		"--datadir",
		mysqlDataPath,
	)
	copyCompletedChannel <- true

	if err != nil {
		pkg.AlertError(configStruct.Alerting, "Could not copy back data files.", err)
		return err
	}

	pkg.Log.Println("Last step: Set correct permissions on backup files")

	// - Set correct permissions
	_, err = pkg.PerformCommand("chown", "-R", "mysql:mysql", mysqlDataPath)
	if err != nil {
		pkg.AlertError(configStruct.Alerting, "Could not set correct permissions on MySQL data file", err)
	}

	pkg.AlertMessage(configStruct.Alerting, "Backup restore complete. Now it is safe to start MySQL.")

	return backupCleanup(volume, mountDirectory, digitalOceanClient)
}

func downloadBackups(backups []backupItem, restoreDirectory string, bucketName string, minioClient *minio.Client) error {
	numberOfCPUs := runtime.NumCPU()

	for _, backup := range backups {
		reader, size, err := getBackupReaderAndSize(minioClient, bucketName, backup.Path)

		if err != nil {
			return err
		}

		progressBar := pb.New(int(size)).SetUnits(pb.U_BYTES)
		progressReader := progressBar.NewProxyReader(reader)

		progressBar.Start()

		err = decompressBackupFile(progressReader, restoreDirectory, numberOfCPUs)

		progressBar.Finish()

		if err != nil {
			return err
		}
	}

	return nil
}

func decompressBackupFile(dataReader io.Reader, restoreDirectory string, numberOfCPUs int) error {
	execCmd := exec.Command("xbstream", "-x", "-C", restoreDirectory)
	execCmd.Stdin = dataReader

	err := execCmd.Start()
	if err != nil {
		return err
	}

	err = execCmd.Wait()
	return err
}

func getBackupReaderAndSize(client *minio.Client, bucketName, objectName string) (io.Reader, int64, error) {
	objectStat, err := client.StatObject(bucketName, objectName, minio.StatObjectOptions{})
	if err != nil {
		return nil, 0, err
	}

	// Seek to current position for incoming reader.
	objectReader, err := client.GetObject(bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, 0, err
	}

	return objectReader, objectStat.Size, nil
}
