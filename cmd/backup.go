package cmd

import (
	"errors"
	"flag"
	"log"
	"os"
	"time"

	"github.com/feederco/really-simple-db-backup/pkg"

	"github.com/digitalocean/godo"
	"github.com/minio/minio-go"
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
		pkg.ErrorLog.Printf("Usage:\n%s perform|perform-full|perform-incremental|restore|upload|test-alert [flags]\n\n", os.Args[0])
		os.Exit(1)
	}

	uploadFileFlag := flag.String("upload-file", "", "[upload] File to upload to bucket")
	existingVolumeIDFlag := flag.String("existing-volume-id", "", "Existing volume ID")
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

	err = prerequisites(configStruct.PersistentStorage)
	if err != nil {
		pkg.ErrorLog.Fatalln("Failed prerequisite tests", err)
	}

	digitalOceanClient := pkg.NewDigitalOceanClient(configStruct.DOKey)
	minioClient, err := minio.New(configStruct.DOSpaceEndpoint, configStruct.DOSpaceKey, configStruct.DOSpaceSecret, true)

	if err != nil {
		pkg.ErrorLog.Fatalln("Could not construct minio client.", err)
	}

	pkg.Log.Println("Backup started", time.Now().Format(time.RFC3339))
	defer pkg.Log.Println("Backup ended", time.Now().Format(time.RFC3339))

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
	case "test-alert":
		pkg.AlertError(configStruct.Alerting, "This is a test alert. Please ignore.", errors.New("Test error"))
	default:
		pkg.ErrorLog.Println("Unknown backup command:", args[0])
	}

	if err != nil {
		pkg.ErrorLog.Printf("Error running `%s`\n\n\t%v\n\n", args[0], err)
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
