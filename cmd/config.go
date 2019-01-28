package cmd

import (
	"encoding/json"
	"errors"
	"flag"
	"io/ioutil"
	"os"

	"github.com/feederco/really-simple-db-backup/pkg"
)

// ConfigStruct contains information that can be preloaded from a .json file
type ConfigStruct struct {
	DOKey             string              `json:"do_key"`
	DOSpaceEndpoint   string              `json:"do_space_endpoint"`
	DOSpaceName       string              `json:"do_space_name"`
	DOSpaceKey        string              `json:"do_space_key"`
	DOSpaceSecret     string              `json:"do_space_secret"`
	MysqlDataPath     string              `json:"mysql_data_path"`
	PersistentStorage string              `json:"persistent_storage"`
	Alerting          *pkg.AlertingConfig `json:"alerting"`
	Retention         *RetentionConfig    `json:"retention"`
}

// RetentionConfig contains options for scheduling: how often full backups are run, retention of old backups
type RetentionConfig struct {
	AutomaticallyRemoveOld  bool `json:"automatically_remove_old"`
	RetentionInDays         int  `json:"retention_in_days"`
	RetentionInHours        int  `json:"retention_in_hours"`
	HoursBetweenFullBackups int  `json:"hours_between_full_backups"`
}

func loadConfig(args []string) ConfigStruct {
	const defaultConfigPath = "/etc/really-simple-db-backup.json"

	configFlag := flag.String("config", "", "Path to a config file to load default configs from. (Default: "+defaultConfigPath+")")

	doKeyFlag := flag.String("do-key", "", "DigitalOcean OAuth2 key created in \"Applications & API\"")
	doSpaceEndpointFlag := flag.String("do-space-endpoint", "", "DigitalOcean Space endpoint to use when uploading backups")
	doSpaceNameFlag := flag.String("do-space-name", "", "DigitalOcean Space bucket name")
	doSpaceKeyFlag := flag.String("do-space-key", "", "DigitalOcean Space key")
	doSpaceSecretFlag := flag.String("do-space-secret", "", "DigitalOcean Space secret")

	mysqlDataPathFlag := flag.String("mysql-data-path", "", "Path to MySQL data directory to backup (Default: /var/lib/mysql)")
	persistentStorageDirectoryFlag := flag.String("persistent-storage", "", "Path to store persistent data about backups. (Default: /var/lib/backup-mysql)")

	err := pkg.ParseCommandLineFlags(args)
	if err != nil {
		flag.Usage()
		os.Exit(1)
	}

	configStruct := ConfigStruct{}

	if *configFlag != "" {
		var didExist bool
		configStruct, didExist, err = loadConfigAtPath(*configFlag)
		if !didExist {
			pkg.ErrorLog.Fatalln("Could not load file from -config flag:", err)
		}

		if err != nil {
			pkg.ErrorLog.Fatalln(err.Error())
		}
	} else {
		var didExist bool
		configStruct, didExist, err = loadConfigAtPath(defaultConfigPath)

		// If default file doesn't exist we don't error. But if it does and is broken we error.
		if didExist && err != nil {
			pkg.ErrorLog.Fatalln(err.Error())
		}
	}

	if configStruct.MysqlDataPath == "" {
		configStruct.MysqlDataPath = "/var/lib/mysql"
	}

	if configStruct.PersistentStorage == "" {
		configStruct.PersistentStorage = "/var/lib/backup-mysql"
	}

	if *doKeyFlag != "" {
		configStruct.DOKey = *doKeyFlag
	}

	if *doSpaceNameFlag != "" {
		configStruct.DOSpaceName = *doSpaceNameFlag
	}

	if *doSpaceEndpointFlag != "" {
		configStruct.DOSpaceEndpoint = *doSpaceEndpointFlag
	}

	if *doSpaceKeyFlag != "" {
		configStruct.DOSpaceKey = *doSpaceKeyFlag
	}

	if *doSpaceSecretFlag != "" {
		configStruct.DOSpaceSecret = *doSpaceSecretFlag
	}

	if *doKeyFlag != "" {
		configStruct.DOKey = *doKeyFlag
	}

	if *mysqlDataPathFlag != "" {
		configStruct.MysqlDataPath = *mysqlDataPathFlag
	}

	if *persistentStorageDirectoryFlag != "" {
		configStruct.PersistentStorage = *persistentStorageDirectoryFlag
	}

	return configStruct
}

func loadConfigAtPath(path string) (ConfigStruct, bool, error) {
	var configStruct ConfigStruct

	configFile, err := ioutil.ReadFile(path)
	if os.IsNotExist(err) {
		return configStruct, false, nil
	}

	if err = json.Unmarshal(configFile, &configStruct); err != nil {
		return configStruct, true, errors.New("Could not load config file. JSON decode failed: " + err.Error())
	}

	return configStruct, true, nil
}
