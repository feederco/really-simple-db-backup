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
	LegacyDOKey           string `json:"do_key"`
	LegacyDOSpaceEndpoint string `json:"do_space_endpoint"`
	LegacyDOSpaceName     string `json:"do_space_name"`
	LegacyDOSpaceKey      string `json:"do_space_key"`
	LegacyDOSpaceSecret   string `json:"do_space_secret"`
	LegacyMysqlDataPath   string `json:"mysql_data_path"`

	DigitalOcean      DigitalOceanConfigStruct `json:"digitalocean"`
	Mysql             MysqlConfigStruct        `json:"mysql"`
	PersistentStorage string                   `json:"persistent_storage"`
	Alerting          *pkg.AlertingConfig      `json:"alerting"`
	Retention         *RetentionConfig         `json:"retention"`
}

// DigitalOceanConfigStruct contains information related to DigitalOcean
type DigitalOceanConfigStruct struct {
	Key           string `json:"key"`
	SpaceEndpoint string `json:"space_endpoint"`
	SpaceName     string `json:"space_name"`
	SpaceKey      string `json:"space_key"`
	SpaceSecret   string `json:"space_secret"`
}

// MysqlConfigStruct contains information related to MySQL
type MysqlConfigStruct struct {
	DataPath string `json:"data_path"`
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

	newConfigStruct := ConfigStruct{}

	if *configFlag != "" {
		var didExist bool
		newConfigStruct, didExist, err = loadConfigAtPath(*configFlag)
		if !didExist {
			pkg.ErrorLog.Fatalln("Could not load file from -config flag:", err)
		}

		if err != nil {
			pkg.ErrorLog.Fatalln(err.Error())
		}
	} else {
		var didExist bool
		newConfigStruct, didExist, err = loadConfigAtPath(defaultConfigPath)

		// If default file doesn't exist we don't error. But if it does and is broken we error.
		if didExist && err != nil {
			pkg.ErrorLog.Fatalln(err.Error())
		}
	}

	// Setting legacy properties

	if newConfigStruct.LegacyMysqlDataPath != "" && newConfigStruct.Mysql.DataPath == "" {
		newConfigStruct.Mysql.DataPath = newConfigStruct.LegacyMysqlDataPath
	}

	if newConfigStruct.LegacyDOKey != "" && newConfigStruct.DigitalOcean.Key == "" {
		newConfigStruct.DigitalOcean.Key = newConfigStruct.LegacyDOKey
	}
	if newConfigStruct.LegacyDOSpaceEndpoint != "" && newConfigStruct.DigitalOcean.SpaceEndpoint == "" {
		newConfigStruct.DigitalOcean.SpaceEndpoint = newConfigStruct.LegacyDOSpaceEndpoint
	}
	if newConfigStruct.LegacyDOSpaceName != "" && newConfigStruct.DigitalOcean.SpaceName == "" {
		newConfigStruct.DigitalOcean.SpaceName = newConfigStruct.LegacyDOSpaceName
	}
	if newConfigStruct.LegacyDOSpaceKey != "" && newConfigStruct.DigitalOcean.SpaceKey == "" {
		newConfigStruct.DigitalOcean.SpaceKey = newConfigStruct.LegacyDOSpaceKey
	}
	if newConfigStruct.LegacyDOSpaceSecret != "" && newConfigStruct.DigitalOcean.SpaceSecret == "" {
		newConfigStruct.DigitalOcean.SpaceSecret = newConfigStruct.LegacyDOSpaceSecret
	}

	if *doKeyFlag != "" {
		newConfigStruct.DigitalOcean.Key = *doKeyFlag
	}

	if *doSpaceNameFlag != "" {
		newConfigStruct.DigitalOcean.SpaceName = *doSpaceNameFlag
	}

	if *doSpaceEndpointFlag != "" {
		newConfigStruct.DigitalOcean.SpaceEndpoint = *doSpaceEndpointFlag
	}

	if *doSpaceKeyFlag != "" {
		newConfigStruct.DigitalOcean.SpaceKey = *doSpaceKeyFlag
	}

	if *doSpaceSecretFlag != "" {
		newConfigStruct.DigitalOcean.SpaceSecret = *doSpaceSecretFlag
	}

	if *doKeyFlag != "" {
		newConfigStruct.DigitalOcean.Key = *doKeyFlag
	}

	if *mysqlDataPathFlag != "" {
		newConfigStruct.Mysql.DataPath = *mysqlDataPathFlag
	}

	if *persistentStorageDirectoryFlag != "" {
		newConfigStruct.PersistentStorage = *persistentStorageDirectoryFlag
	}

	// Setting defaults

	if newConfigStruct.Mysql.DataPath == "" {
		newConfigStruct.Mysql.DataPath = "/var/lib/mysql"
	}

	if newConfigStruct.PersistentStorage == "" {
		newConfigStruct.PersistentStorage = "/var/lib/backup-mysql"
	}

	return newConfigStruct
}

func loadConfigAtPath(path string) (ConfigStruct, bool, error) {
	var newConfigStruct ConfigStruct

	configFile, err := ioutil.ReadFile(path)
	if os.IsNotExist(err) {
		return newConfigStruct, false, nil
	}

	if err = json.Unmarshal(configFile, &newConfigStruct); err != nil {
		return newConfigStruct, true, errors.New("Could not load config file. JSON decode failed: " + err.Error())
	}

	return newConfigStruct, true, nil
}
