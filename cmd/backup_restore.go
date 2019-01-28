package cmd

import (
	"errors"
	"os"
	"path"
	"runtime"
	"strconv"
	"time"

	"github.com/digitalocean/godo"
	"github.com/feederco/really-simple-db-backup/pkg"
	minio "github.com/minio/minio-go"
)

func backupMysqlPerformRestore(fromHostname string, restoreTimestamp string, backupBucket string, mysqlDataPath string, existingVolumeID string, digitalOceanClient *pkg.DigitalOceanClient, minioClient *minio.Client) error {
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
	)

	restoreDirectory := path.Join(mountDirectory, "really-simple-db-restore")
	err = os.Mkdir(restoreDirectory, 0755)
	if err != nil {
		pkg.ErrorLog.Println("Could not create directory to house backup files.")
		return nil
	}

	// - Download full backup and incremental pieces
	err = downloadBackups(backupFiles, restoreDirectory, backupBucket, minioClient)
	if err != nil {
		pkg.ErrorLog.Println("Could not download backups!")
		return err
	}

	numberOfCPUs := runtime.NumCPU()

	// - Decompress files with as many cores as possible
	_, err = pkg.PerformCommand(
		"xtrabackup",
		"--decompress",
		"--target-dir",
		restoreDirectory,
		"--paralell",
		strconv.FormatInt(int64(numberOfCPUs), 10),
	)

	if err != nil {
		pkg.AlertError(configStruct.Alerting, "Could not create decompress backup.", err)
		return err
	}

	// - Prepare backup
	_, err = pkg.PerformCommand("xtrabackup", "--prepare", "--target-dir", restoreDirectory)
	if err != nil {
		pkg.AlertError(configStruct.Alerting, "Could not create backup.", err)
		return backupCleanup(volume, mountDirectory, digitalOceanClient)
	}

	// - Move to MySQL data directory
	_, err = pkg.PerformCommand("xtrabackup", "--copy-back", "--target-dir", restoreDirectory)
	if err != nil {
		pkg.AlertError(configStruct.Alerting, "Could not copy back data files.", err)
		return err
	}

	// - Set correct permissions
	_, err = pkg.PerformCommand("chown", "-R", "mysql:mysql", mysqlDataPath)

	pkg.AlertMessage(configStruct.Alerting, "Backup restore complete. Now it is safe to start MySQL.")

	return backupCleanup(volume, mountDirectory, digitalOceanClient)
}

func downloadBackups(backups []backupItem, location string, bucketName string, minioClient *minio.Client) error {
	for _, backup := range backups {
		fileName := path.Base(backup.Path)

		err := minioClient.FGetObject(
			bucketName,
			backup.Path,
			path.Join(location, fileName),
			minio.GetObjectOptions{},
		)

		if err != nil {
			return err
		}
	}

	return nil
}
