package cmd

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/feederco/really-simple-db-backup/pkg"

	"github.com/digitalocean/godo"
	minio "github.com/minio/minio-go"
)

const fileSystemForVolume = "ext4"

const backupTypeIncremental = "incremental"
const backupTypeFull = "full"
const backupTypeDecide = "decide"

var configStruct ConfigStruct

// Begin begin!
func Begin(cliArgs []string) {
	var err error

	pkg.Log = log.New(os.Stdout, "", log.LstdFlags)
	pkg.ErrorLog = log.New(os.Stderr, "", log.LstdFlags)

	args := cliArgs[1:]

	if len(args) == 0 {
		pkg.ErrorLog.Printf("\nusage:\n%s perform|perform-full|perform-incremental|upload|restore|download|finalize-restore|test-alert|list-backups|prune [flags]\n\n", os.Args[0])
		os.Exit(1)
	}

	uploadFileFlag := flag.String("upload-file", "", "[upload] File to upload to bucket")
	existingVolumeIDFlag := flag.String("existing-volume-id", "", "Existing volume ID")
	existingBackupDirectoryFlag := flag.String("existing-backup-directory", "", "Existing backup directory")
	existingRestoreDirectoryFlag := flag.String("existing-restore-directory", "", "Existing restore directory")
	hostnameFlag := flag.String("hostname", "", "Hostname of backups to list")
	timestampFlag := flag.String("timestamp", "", "List backups since timestamp. Should be in format YYYYMMDDHHII")
	verboseFlag := flag.Bool("v", false, "Verbose logging")

	configStruct = loadConfig(args[1:])

	pkg.VerboseMode = *verboseFlag

	if configStruct.DigitalOcean.SpaceName == "" {
		pkg.ErrorLog.Fatalln("-do-space-name parameter required")
	}

	if configStruct.DigitalOcean.SpaceEndpoint == "" {
		pkg.ErrorLog.Fatalln("-do-space-endpoint parameter required")
	}

	if configStruct.DigitalOcean.SpaceKey == "" {
		pkg.ErrorLog.Fatalln("-do-space-key parameter required")
	}

	if configStruct.DigitalOcean.SpaceSecret == "" {
		pkg.ErrorLog.Fatalln("-do-space-secret-flag parameter required")
	}

	digitalOceanClient := pkg.NewDigitalOceanClient(configStruct.DigitalOcean.Key)
	minioClient, err := minio.New(configStruct.DigitalOcean.SpaceEndpoint, configStruct.DigitalOcean.SpaceKey, configStruct.DigitalOcean.SpaceSecret, true)

	if err != nil {
		pkg.ErrorLog.Fatalln("Could not construct minio client.", err)
	}

	hostname, _ := os.Hostname()
	if *hostnameFlag != "" {
		hostname = *hostnameFlag
	}

	switch args[0] {
	case "perform":
		err = backupMysqlPerform(
			backupTypeDecide,
			configStruct.DigitalOcean.SpaceName,
			configStruct.Mysql.DataPath,
			*existingVolumeIDFlag,
			*existingBackupDirectoryFlag,
			configStruct.PersistentStorage,
			digitalOceanClient,
			minioClient,
		)
	case "perform-full":
		err = backupMysqlPerform(
			backupTypeFull,
			configStruct.DigitalOcean.SpaceName,
			configStruct.Mysql.DataPath,
			*existingVolumeIDFlag,
			*existingBackupDirectoryFlag,
			configStruct.PersistentStorage,
			digitalOceanClient,
			minioClient,
		)
	case "perform-incremental":
		err = backupMysqlPerform(
			backupTypeIncremental,
			configStruct.DigitalOcean.SpaceName,
			configStruct.Mysql.DataPath,
			*existingVolumeIDFlag,
			*existingBackupDirectoryFlag,
			configStruct.PersistentStorage,
			digitalOceanClient,
			minioClient,
		)
	case "restore":
		fromHostname := hostname
		if *hostnameFlag != "" {
			fromHostname = *hostnameFlag
		}

		mysqlDataPath := configStruct.Mysql.DataPath

		// Make sure MySQL data path exists
		if _, fileErr := os.Stat(mysqlDataPath); fileErr != nil {
			// It did not exist, just to be sure we try to create it. If that fails this script can't continue
			if os.IsNotExist(fileErr) {
				err = os.MkdirAll(mysqlDataPath, 0700)
				if err != nil {
					pkg.ErrorLog.Fatalln("Could not access nor create the MySQL data path")
				}
			} else {
				pkg.ErrorLog.Fatalln("Could not access the MySQL data path")
			}
		}

		var restoreDirectory string
		var mountDirectory string
		var volume *godo.Volume
		restoreDirectory, mountDirectory, volume, err = backupMysqlDownloadAndPrepare(
			fromHostname,
			*timestampFlag,
			configStruct.DigitalOcean.SpaceName,
			*existingVolumeIDFlag,
			*existingBackupDirectoryFlag,
			digitalOceanClient,
			minioClient,
		)

		if err == nil {
			err = backupMysqlFinalizeRestore(
				restoreDirectory,
				configStruct.Mysql.DataPath,
				mountDirectory,
				volume,
				digitalOceanClient,
				minioClient,
			)
		}
	case "download":
		fromHostname := hostname
		if *hostnameFlag != "" {
			fromHostname = *hostnameFlag
		}

		var restoreDirectory string

		restoreDirectory, _, _, err = backupMysqlDownloadAndPrepare(
			fromHostname,
			*timestampFlag,
			configStruct.DigitalOcean.SpaceName,
			*existingVolumeIDFlag,
			*existingRestoreDirectoryFlag,
			digitalOceanClient,
			minioClient,
		)

		if err == nil {
			pkg.Log.Printf("Downloaded complete. Directory: %s\n", restoreDirectory)
		}
	case "finalize-restore":
		err = backupMysqlFinalizeRestore(
			*existingRestoreDirectoryFlag,
			configStruct.Mysql.DataPath,
			"",
			nil,
			digitalOceanClient,
			minioClient,
		)

		if err == nil {
			pkg.Log.Printf("Restore complete. Don't forget to cleanup manually!")
		}
	case "upload":
		if *uploadFileFlag == "" {
			pkg.ErrorLog.Fatalln("-upload-file parameter required for `upload` command.")
		}

		err = backupMysqlUpload(*uploadFileFlag, configStruct.DigitalOcean.SpaceName, minioClient)
	case "prune":
		if configStruct.Retention == nil {
			pkg.Log.Println("No retention config. Nothing to do. Exiting")
			return
		}

		var allBackups []backupItem
		allBackups, err = listAllBackups(hostname, configStruct.DigitalOcean.SpaceName, minioClient)
		if err != nil {
			pkg.ErrorLog.Fatalln("Could not list backups to remove:", err)
		}

		backupsToDelete := findBackupsThatCanBeDeleted(allBackups, time.Now(), configStruct.Retention)

		if len(backupsToDelete) > 0 {
			fmt.Println("")

			for index, backup := range backupsToDelete {
				fmt.Printf("#%d: %s (%.3f GB) (%.1f days old)\n", index+1, backup.Path, float64(backup.Size)/1000/1000/1000, time.Now().Sub(backup.CreatedAt).Truncate(time.Hour).Hours()/24)
			}

			fmt.Printf(
				"\nDelete %d %s backups forever: (yes or y to accept)\n",
				len(backupsToDelete),
				pluralize(len(backupsToDelete), "backup", "backups"),
			)

			reader := bufio.NewReader(os.Stdin)
			agreement, _ := reader.ReadString('\n')
			agreement = strings.ToLower(agreement)

			if agreement == "yes" || agreement == "y" {
				var actuallyRemovedBackups []backupItem
				actuallyRemovedBackups, err = removeBackups(backupsToDelete, configStruct.DigitalOcean.SpaceName, minioClient)
				if err != nil {
					errString := ""
					if len(actuallyRemovedBackups) > 0 {
						errString = fmt.Sprintf("HOWEVER. %d %s deleted!", len(actuallyRemovedBackups), pluralize(len(actuallyRemovedBackups), "backup was", "backups were"))
					}
					pkg.ErrorLog.Fatalf("An error occurred when trying to delete backups. %s\n\nError: %s\n", errString, err)
				}

				log.Println("Complete!")
				log.Printf("Deleted %d %s\n", len(actuallyRemovedBackups), pluralize(len(actuallyRemovedBackups), "backup was", "backups were"))
			} else {
				log.Println("Everything left as-is.")
			}
		}
	case "test-alert":
		pkg.AlertError(configStruct.Alerting, "This is a test alert. Please ignore.", errors.New("Test error"))
	case "list-backups":
		pkg.Log.Printf("Loading backups for %s\n", hostname)

		var backups []backupItem
		backups, err = listAllBackups(hostname, configStruct.DigitalOcean.SpaceName, minioClient)

		if err != nil {
			pkg.ErrorLog.Fatalln("Could not list backups:", err)
		}

		if *timestampFlag != "" {
			var sinceTimestamp time.Time
			sinceTimestamp, err = parseBackupTimestamp(*timestampFlag)
			if err != nil {
				pkg.ErrorLog.Fatalln("Incorrect timestamp past in:", err)
			}

			pkg.Log.Printf("Listing backups since %s\n", sinceTimestamp.Format(time.RFC3339))
			backups = findRelevantBackupsUpTo(sinceTimestamp, backups)
		}

		for index, backup := range backups {
			pkg.Log.Printf("%d:\t%s (created at %s)", index, backup.Path, backup.CreatedAt)
		}
	default:
		pkg.ErrorLog.Println("Unknown backup command:", args[0])
	}

	if err != nil {
		pkg.ErrorLog.Printf("Error running `%s`\n\n\t%v\n\n", args[0], err)
	}
}

func pluralize(count int, singular string, plural string) string {
	if count == 1 {
		return singular
	}

	return plural
}

func backupCleanup(volume *godo.Volume, mountDirectory string, digitalOceanClient *pkg.DigitalOceanClient) error {
	if mountDirectory != "" {
		err := pkg.UnmountVolume(mountDirectory, volume.ID, volume.DropletIDs[0], digitalOceanClient)
		if err != nil {
			pkg.AlertError(configStruct.Alerting, "Could not unmount volume.", err)
			return err
		}
	}

	if volume != nil {
		err := pkg.DestroyVolume(volume.ID, digitalOceanClient)
		if err != nil {
			pkg.AlertError(configStruct.Alerting, "Could not destroy volume: "+volume.ID, err)
			return err
		}
	}

	return nil
}
