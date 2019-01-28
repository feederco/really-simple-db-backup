package cmd

import (
	"errors"
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
	aDecentSizeInGigaBytes := sizeInGigaBytes + 10

	var volume *godo.Volume
	var mountDirectory string

	volume, mountDirectory, err = createAndMountVolumeForUse(
		"mysql-backup-",
		aDecentSizeInGigaBytes,
		digitalOceanClient,
		existingVolumeID,
	)

	// - Mount volume to house backup
	// - Download full backup and incremental pieces
	// - Unpack into directory

	// - Prepare backup
	// _, err = pkg.PerformCommand("xtrabackup", "--prepare", "--target-dir", backupDirectory)
	// if err != nil {
	// 	pkg.AlertError("Could not create backup.", err)
	// 	return backupCleanup(volume, mountDirectory, digitalOceanClient)
	// }

	// - Move to MySQL data directory

	// - Set correct permissions
	_, err = pkg.PerformCommand("chown", "-R", "mysql:mysql", mysqlDataPath)

	return err
}
