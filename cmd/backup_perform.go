package cmd

import (
	"errors"
	"fmt"
	"path"
	"time"

	"github.com/feederco/really-simple-db-backup/pkg"
	minio "github.com/minio/minio-go"

	"os"
	"os/exec"

	"github.com/digitalocean/godo"
)

func backupMysqlPerform(backupType string, backupsBucket string, mysqlDataPath string, existingVolumeID string, persistentStorageDirectory string, digitalOceanClient *pkg.DigitalOceanClient, minioClient *minio.Client) error {
	var err error

	pkg.Log.Println("Backup started", time.Now().Format(time.RFC3339))
	defer pkg.Log.Println("Backup ended", time.Now().Format(time.RFC3339))

	err = prerequisites(configStruct.PersistentStorage)
	if err != nil {
		pkg.ErrorLog.Fatalln("Failed prerequisite tests", err)
	}

	if backupType != backupTypeFull && backupType != backupTypeIncremental && backupType != backupTypeDecide {
		return errors.New("Invalid backupType: " + backupType)
	}

	checkpointFilePath := path.Join(persistentStorageDirectory, "xtrabackup_checkpoints")

	// # Game plan
	err = backupPrerequisites()
	if err != nil {
		return err
	}

	hostname, _ := os.Hostname()

	if backupType == backupTypeDecide {
		backupType, err = backupDecide(
			configStruct.Retention,
			checkpointFilePath,
			hostname,
			backupsBucket,
			minioClient,
		)
		if err != nil {
			pkg.AlertError(configStruct.Alerting, "Could not decide backup type", err)
		}

		pkg.Log.Printf("Decided on backup type: %s\n", backupType)
	}

	// - Get size of database
	sizeInBytes, err := pkg.DirSize(mysqlDataPath)
	if err != nil {
		return err
	}

	sizeInGigaBytes := bytesToGigaBytes(sizeInBytes)
	// aDecentSizeInGigaBytes := sizeInGigaBytes + (sizeInGigaBytes / 6)
	aDecentSizeInGigaBytes := sizeInGigaBytes

	var volume *godo.Volume
	var mountDirectory string

	volume, mountDirectory, err = createAndMountVolumeForUse(
		"mysql-backup-",
		aDecentSizeInGigaBytes,
		digitalOceanClient,
		existingVolumeID,
	)

	if err != nil {
		return backupCleanup(volume, mountDirectory, digitalOceanClient)
	}

	// !! From this point onward we have created things that need to be cleaned up

	pkg.Log.Println("Backups running.")

	backupName := volume.Name + "." + backupType

	backupDirectory := path.Join(mountDirectory, "mysql-backup-"+backupType)
	backupFileTemporary := path.Join(backupDirectory, backupName+".xbstream.incomplete")
	backupFile := path.Join(backupDirectory, backupName+".xbstream")

	// - Start Percona XtraBackup
	err = (func() error {
		err = os.MkdirAll(backupDirectory, 0755)
		if err != nil {
			pkg.AlertError(configStruct.Alerting, "Could not create backup directory.", err)
			return err
		}

		var outputFile *os.File
		outputFile, err = os.Create(backupFileTemporary)
		if err != nil {
			pkg.AlertError(configStruct.Alerting, "Could not create backup file.", err)
			return err
		}
		defer outputFile.Close()

		backupArgs := []string{
			"--backup",
			"--extra-lsndir",
			persistentStorageDirectory,
			"--target-dir",
			backupDirectory + "/",
			"--compress",
			"--stream=xbstream",
			"--slave-info",
		}

		// Add option to read LSN (log sequence number) if taking an incremental backup
		if backupType == backupTypeIncremental {
			lastLsn, lsnErr := getLastLSNFromFile(checkpointFilePath)
			if lsnErr != nil {
				pkg.AlertError(configStruct.Alerting, "Could not fetch LSN from checkpoint file while doing incremental backup.", lsnErr)
				return lsnErr
			}

			if lastLsn == "" {
				pkg.Log.Print("No last LSN found, doing full backup instead.")
			} else {
				backupArgs = append(backupArgs, "--incremental-lsn", lastLsn)
			}
		}

		backupCmd := exec.Command("xtrabackup", backupArgs...)
		backupCmd.Stdout = outputFile

		err = backupCmd.Start()
		if err != nil {
			pkg.AlertError(configStruct.Alerting, "Could not start xtrabackup command.", err)
			return err
		}

		err = backupCmd.Wait()
		if err != nil {
			pkg.AlertError(configStruct.Alerting, "Could not create backup.", err)
			return err
		}

		err = os.Rename(backupFileTemporary, backupFile)
		if err != nil {
			pkg.AlertError(configStruct.Alerting, "Backup was completed but couldnt rename the file to reflect this.", err)
			return err
		}

		return nil
	})()

	if err != nil {
		pkg.AlertError(configStruct.Alerting, "Could not create backup. Leaving it as is!", err)
		return err
	}

	// - On success: upload to a bucket
	err = backupMysqlUpload(backupFile, backupsBucket, minioClient)
	if err != nil {
		pkg.AlertError(configStruct.Alerting, "Could not upload backup to directory. Leaving it as is!", err)
		return err
	}

	// Success! Now we can consider removing old backups
	if backupType == backupTypeFull && configStruct.Retention != nil && configStruct.Retention.AutomaticallyRemoveOld {
		allBackups, backupErr := listAllBackups(hostname, backupsBucket, minioClient)
		if backupErr != nil {
			pkg.AlertError(configStruct.Alerting, "Backup completed, but could not perform pruning. Failed on listing backups.", err)
		} else {
			backupsToDelete := findBackupsThatCanBeDeleted(allBackups, time.Now(), configStruct.Retention)
			deletedBackups, backupErr := removeBackups(backupsToDelete, backupsBucket, minioClient)

			if backupErr != nil {
				pkg.AlertError(configStruct.Alerting, fmt.Sprintf("Backup completed, but could not delete backups pruning. Failed on deleting. Was able delete %d %s before failure.", len(deletedBackups), pluralize(len(deletedBackups), "backup", "backups")), err)
			}
		}
	}

	return backupCleanup(volume, mountDirectory, digitalOceanClient)
}

func bytesToGigaBytes(bytes int64) int64 {
	return bytes / (1 << (10 * 3))
}
