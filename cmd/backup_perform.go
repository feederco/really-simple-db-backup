package cmd

import (
	"errors"
	"path"

	"github.com/feederco/really-simple-db-backup/pkg"

	"os"
	"os/exec"

	"github.com/digitalocean/godo"
	"github.com/minio/minio-go"
)

func backupMysqlPerform(backupType string, backupsBucket string, mysqlDataPath string, existingVolumeID string, persistentStorageDirectory string, digitalOceanClient *pkg.DigitalOceanClient, minioClient *minio.Client) error {
	var err error

	if backupType != backupTypeFull && backupType != backupTypeIncremental && backupType != backupTypeDecide {
		return errors.New("Invalid backupType: " + backupType)
	}

	checkpointFilePath := path.Join(persistentStorageDirectory, "xtrabackup_checkpoints")

	// # Game plan
	err = backupPrerequisites()
	if err != nil {
		return err
	}

	if backupType == backupTypeDecide {
		lastLsn, lastLsnErr := getLastLSNFromFile(checkpointFilePath)
		if lastLsnErr == nil && len(lastLsn) > 0 {
			backupType = backupTypeIncremental
		} else {
			backupType = backupTypeFull
		}

		pkg.Log.Printf("Decided on backup type: %s\n", backupType)
	}

	// - Get size of database
	sizeInBytes, err := pkg.DirSize(mysqlDataPath)
	if err != nil {
		return err
	}

	sizeInGigaBytes := sizeInBytes / (1 << (10 * 3))
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

	pkg.Log.Println("Backups starting.")

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

	return backupCleanup(volume, mountDirectory, digitalOceanClient)
}
