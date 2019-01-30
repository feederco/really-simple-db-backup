package cmd

import (
	"errors"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strconv"
	"time"

	"github.com/digitalocean/godo"
	"github.com/feederco/really-simple-db-backup/pkg"
	minio "github.com/minio/minio-go"
)

func backupMysqlPerformRestore(fromHostname string, restoreTimestamp string, backupBucket string, mysqlDataPath string, existingVolumeID string, existingBackupDirectory string, digitalOceanClient *pkg.DigitalOceanClient, minioClient *minio.Client) error {
	var err error

	err = prerequisites(configStruct.PersistentStorage)
	if err != nil {
		return err
	}

	err = backupPrerequisites()
	if err != nil {
		return err
	}

	sinceTimestamp := time.Now()
	if restoreTimestamp != "" {
		sinceTimestamp, err = parseBackupTimestamp(restoreTimestamp)
		if err != nil {
			return errors.New("Incorrect timestamp passed in: " + restoreTimestamp + " (error: " + err.Error() + ")")
		}
	}

	pkg.Log.Println("Listing backups since", sinceTimestamp.Format(time.RFC3339))

	// Game plan:

	// - List all backups we need
	allBackups, err := listAllBackups(fromHostname, backupBucket, minioClient)
	if err != nil {
		return err
	}

	backupFiles := findRelevantBackupsUpTo(sinceTimestamp, allBackups)
	if len(backupFiles) == 0 {
		return errors.New("No backup found to restore from")
	}

	pkg.Log.Printf("%d backup files found\n", len(backupFiles))

	totalSizeInBytes := int64(0)
	for _, backupFile := range backupFiles {
		totalSizeInBytes += backupFile.Size
	}

	// - Create & mount volume to house backup
	sizeInGigaBytes := bytesToGigaBytes(totalSizeInBytes)
	aDecentSizeInGigaBytes := sizeInGigaBytes * 10

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
		return nil
	}

	restoreDirectory := path.Join(mountDirectory, "really-simple-db-restore")
	err = os.MkdirAll(restoreDirectory, 0755)
	if err != nil {
		pkg.ErrorLog.Println("Could not create directory to house backup files.")
		return nil
	}

	pkg.Log.Println("Downloading backups")

	// - Download full backup and incremental pieces
	var localFiles []string
	localFiles, err = downloadBackups(backupFiles, restoreDirectory, backupBucket, minioClient)
	if err != nil {
		pkg.ErrorLog.Println("Could not download backups!")
		return err
	}

	pkg.Log.Println("Download completed backups")
	pkg.Log.Println("Decompressing backups")

	numberOfCPUs := runtime.NumCPU()

	for _, backupFile := range localFiles {
		err = decompressBackupFile(backupFile, restoreDirectory, numberOfCPUs)
		if err != nil {
			pkg.AlertError(configStruct.Alerting, "Could not decompress backup file.", err)
			return err
		}
	}

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
		return backupCleanup(volume, mountDirectory, digitalOceanClient)
	}

	pkg.Log.Println("Prepare completed! Putting files back")
	pkg.Log.Println("Warning: Removing everything in the MySQL data directory")

	// We try to run this command. If it fails, we just run xtrabackup --copy-back anyway.
	// It will error if the directory is not empty
	pkg.PerformCommand("mv", mysqlDataPath, "/tmp/")

	// - Move to MySQL data directory
	_, err = pkg.PerformCommand("xtrabackup", "--copy-back", "--target-dir", restoreDirectory)
	if err != nil {
		pkg.AlertError(configStruct.Alerting, "Could not copy back data files.", err)
		return err
	}

	pkg.Log.Println("Last step: Set correct permissions on backup files")

	// - Set correct permissions
	_, err = pkg.PerformCommand("chown", "-R", "mysql:mysql", mysqlDataPath)

	pkg.AlertMessage(configStruct.Alerting, "Backup restore complete. Now it is safe to start MySQL.")

	return backupCleanup(volume, mountDirectory, digitalOceanClient)
}

func downloadBackups(backups []backupItem, location string, bucketName string, minioClient *minio.Client) ([]string, error) {
	localFiles := make([]string, 0)
	for _, backup := range backups {
		fileName := path.Base(backup.Path)
		destinationFileName := path.Join(location, fileName)

		// Do not download files that are already downloaded
		stat, err := os.Stat(destinationFileName)
		if err == nil && stat.Size() > 0 {
			localFiles = append(localFiles, destinationFileName)
			continue
		}

		err = minioClient.FGetObject(
			bucketName,
			backup.Path,
			destinationFileName,
			minio.GetObjectOptions{},
		)

		if err != nil {
			return localFiles, err
		}

		localFiles = append(localFiles, destinationFileName)
	}

	return localFiles, nil
}

func decompressBackupFile(backupFile string, restoreDirectory string, numberOfCPUs int) error {
	inputFile, err := os.Open(backupFile)
	if err != nil {
		return err
	}

	execCmd := exec.Command("xbstream", "-x", "-C", restoreDirectory)
	execCmd.Stdin = inputFile

	err = execCmd.Start()
	if err != nil {
		return err
	}

	err = execCmd.Wait()
	if err != nil {
		return err
	}

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

	if err != nil {
		return err
	}

	return nil
}
