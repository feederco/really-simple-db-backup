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
	pkg.Log = log.New(os.Stdout, "", log.LstdFlags)
	pkg.ErrorLog = log.New(os.Stderr, "", log.LstdFlags)

	args := cliArgs[1:]

	if len(args) == 0 {
		pkg.ErrorLog.Printf("Usage:\n%s perform|perform-full|perform-incremental|restore|upload|test-alert|list-backups|prune [flags]\n\n", os.Args[0])
		os.Exit(1)
	}

	uploadFileFlag := flag.String("upload-file", "", "[upload] File to upload to bucket")
	existingVolumeIDFlag := flag.String("existing-volume-id", "", "Existing volume ID")
	hostnameFlag := flag.String("hostname", "", "Hostname of backups to list")
	timestampFlag := flag.String("timestamp", "", "List backups since timestamp. Should be in format YYYYMMDDHHII")
	verboseFlag := flag.Bool("v", false, "Verbose logging")

	pkg.VerboseMode = *verboseFlag

	configStruct = loadConfig(args[1:])

	if configStruct.DOSpaceName == "" {
		pkg.ErrorLog.Fatalln("-do-space-name parameter required")
	}

	if configStruct.DOSpaceEndpoint == "" {
		pkg.ErrorLog.Fatalln("-do-space-endpoint parameter required")
	}

	if configStruct.DOSpaceKey == "" {
		pkg.ErrorLog.Fatalln("-do-space-key parameter required")
	}

	if configStruct.DOSpaceSecret == "" {
		pkg.ErrorLog.Fatalln("-do-space-secret-flag parameter required")
	}

	var err error

	digitalOceanClient := pkg.NewDigitalOceanClient(configStruct.DOKey)
	minioClient, err := minio.New(configStruct.DOSpaceEndpoint, configStruct.DOSpaceKey, configStruct.DOSpaceSecret, true)

	if err != nil {
		pkg.ErrorLog.Fatalln("Could not construct minio client.", err)
	}

	pkg.Log.Println("Backup started", time.Now().Format(time.RFC3339))
	defer pkg.Log.Println("Backup ended", time.Now().Format(time.RFC3339))

	hostname, _ := os.Hostname()
	if *hostnameFlag != "" {
		hostname = *hostnameFlag
	}

	switch args[0] {
	case "perform":
		err = backupMysqlPerform(backupTypeDecide, configStruct.DOSpaceName, configStruct.MysqlDataPath, *existingVolumeIDFlag, configStruct.PersistentStorage, digitalOceanClient, minioClient)
	case "perform-full":
		err = backupMysqlPerform(backupTypeFull, configStruct.DOSpaceName, configStruct.MysqlDataPath, *existingVolumeIDFlag, configStruct.PersistentStorage, digitalOceanClient, minioClient)
	case "perform-incremental":
		err = backupMysqlPerform(backupTypeIncremental, configStruct.DOSpaceName, configStruct.MysqlDataPath, *existingVolumeIDFlag, configStruct.PersistentStorage, digitalOceanClient, minioClient)
	case "restore":
		err = backupMysqlPerformRestore()
	case "upload":
		if *uploadFileFlag == "" {
			pkg.ErrorLog.Fatalln("-upload-file parameter required for `upload` command.")
		}

		err = backupMysqlUpload(*uploadFileFlag, configStruct.DOSpaceName, minioClient)
	case "prune":
		if configStruct.Retention == nil {
			pkg.Log.Println("No retention config. Nothing to do. Exiting")
			return
		}

		var allBackups []backupItem
		allBackups, err = listAllBackups(hostname, configStruct.DOSpaceName, minioClient)
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
				actuallyRemovedBackups, err = removeBackups(backupsToDelete, configStruct.DOSpaceName, minioClient)
				if err != nil {
					errString := ""
					if len(actuallyRemovedBackups) > 0 {
						errString = fmt.Sprintf("HOWEVER. %d %s deleted!", len(actuallyRemovedBackups), pluralize(len(actuallyRemovedBackups), "backup was", "backups were"))
					}
					pkg.ErrorLog.Fatalf("An error occurred when trying to delete backups. %s\n\nError: %s\n", errString, err)
				}

				log.Printf("Complete!\nDeleted %d %s", len(actuallyRemovedBackups), pluralize(len(actuallyRemovedBackups), "backup was", "backups were"))
			} else {
				log.Println("Everything left as-is.")
			}
		}
	case "test-alert":
		pkg.AlertError(configStruct.Alerting, "This is a test alert. Please ignore.", errors.New("Test error"))
	case "list-backups":
		pkg.Log.Printf("Loading backups for %s\n", hostname)

		var backups []backupItem
		backups, err = listAllBackups(hostname, configStruct.DOSpaceName, minioClient)

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
	} else {
		return plural
	}
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
