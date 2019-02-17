package cmd

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/feederco/really-simple-db-backup/pkg"
)

const exampleJSONContents = `
{
	"digitalocean": {
	  "key": "do.key",
	  "space_endpoint": "do.space_endpoint",
	  "space_name": "do.space_name",
	  "space_key": "do.space_key",
	  "space_secret": "do.space_secret"
	},
	"mysql": {
  	"data_path": "mysql.data_path"
	},
  "persistent_storage": "hi: persistent_storage"
}
`

const exampleLegacyJSONContents = `
{
  "do_key": "hi: do_key",
  "do_space_endpoint": "hi: do_space_endpoint",
  "do_space_name": "hi: do_space_name",
  "do_space_key": "hi: do_space_key",
  "do_space_secret": "hi: do_space_secret",
  "mysql_data_path": "hi: mysql_data_path",
  "persistent_storage": "hi: persistent_storage"
}
`

func setupTest() {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	pkg.Log = log.New(os.Stdout, "", log.LstdFlags)
	pkg.ErrorLog = log.New(os.Stderr, "", log.LstdFlags)
}

func TestLoadConfigFromCommandLine(t *testing.T) {
	setupTest()

	configStruct := loadConfig([]string{
		"-do-key",
		"do_key",
		"-do-space-endpoint",
		"do_space_endpoint",
		"-do-space-name",
		"do_space_name",
		"-do-space-key",
		"do_space_key",
		"-do-space-secret",
		"do_space_secret",
		"-mysql-data-path",
		"mysql_data_path",
		"-persistent-storage",
		"persistent_storage",
	})

	if configStruct.DigitalOcean.Key != "do_key" {
		t.Errorf("Incorrect DOKey found: %s", configStruct.DigitalOcean.Key)
	}
	if configStruct.DigitalOcean.SpaceEndpoint != "do_space_endpoint" {
		t.Errorf("Incorrect DOSpaceEndpoint found: %s", configStruct.DigitalOcean.SpaceEndpoint)
	}
	if configStruct.DigitalOcean.SpaceName != "do_space_name" {
		t.Errorf("Incorrect DOSpaceName found: %s", configStruct.DigitalOcean.SpaceName)
	}
	if configStruct.DigitalOcean.SpaceKey != "do_space_key" {
		t.Errorf("Incorrect DOSpaceKey found: %s", configStruct.DigitalOcean.SpaceKey)
	}
	if configStruct.DigitalOcean.SpaceSecret != "do_space_secret" {
		t.Errorf("Incorrect DOSpaceSecret found: %s", configStruct.DigitalOcean.SpaceSecret)
	}
	if configStruct.Mysql.DataPath != "mysql_data_path" {
		t.Errorf("Incorrect MysqlDataPath found: %s", configStruct.Mysql.DataPath)
	}
	if configStruct.PersistentStorage != "persistent_storage" {
		t.Errorf("Incorrect PersistentStorage found: %s", configStruct.PersistentStorage)
	}
}

func TestLoadConfigFromConfigFile(t *testing.T) {
	setupTest()

	ioutil.WriteFile("_test_file.json", []byte(exampleJSONContents), 0755)
	defer os.Remove("_test_file.json")

	configStruct := loadConfig([]string{
		"-config",
		"_test_file.json",
	})

	if configStruct.DigitalOcean.Key != "do.key" {
		t.Errorf("Incorrect DOKey found: %s", configStruct.DigitalOcean.Key)
	}
	if configStruct.DigitalOcean.SpaceEndpoint != "do.space_endpoint" {
		t.Errorf("Incorrect DOSpaceEndpoint found: %s", configStruct.DigitalOcean.SpaceEndpoint)
	}
	if configStruct.DigitalOcean.SpaceName != "do.space_name" {
		t.Errorf("Incorrect DOSpaceName found: %s", configStruct.DigitalOcean.SpaceName)
	}
	if configStruct.DigitalOcean.SpaceKey != "do.space_key" {
		t.Errorf("Incorrect DOSpaceKey found: %s", configStruct.DigitalOcean.SpaceKey)
	}
	if configStruct.DigitalOcean.SpaceSecret != "do.space_secret" {
		t.Errorf("Incorrect DOSpaceSecret found: %s", configStruct.DigitalOcean.SpaceSecret)
	}
	if configStruct.Mysql.DataPath != "mysql.data_path" {
		t.Errorf("Incorrect MysqlDataPath found: %s", configStruct.Mysql.DataPath)
	}
	if configStruct.PersistentStorage != "hi: persistent_storage" {
		t.Errorf("Incorrect PersistentStorage found: %s", configStruct.PersistentStorage)
	}
}

func TestLoadLegacyConfigFromConfigFile(t *testing.T) {
	setupTest()

	ioutil.WriteFile("_test_file.json", []byte(exampleLegacyJSONContents), 0755)
	defer os.Remove("_test_file.json")

	configStruct := loadConfig([]string{
		"-config",
		"_test_file.json",
	})

	if configStruct.DigitalOcean.Key != "hi: do_key" {
		t.Errorf("Incorrect DOKey found: %s", configStruct.DigitalOcean.Key)
	}
	if configStruct.DigitalOcean.SpaceEndpoint != "hi: do_space_endpoint" {
		t.Errorf("Incorrect DOSpaceEndpoint found: %s", configStruct.DigitalOcean.SpaceEndpoint)
	}
	if configStruct.DigitalOcean.SpaceName != "hi: do_space_name" {
		t.Errorf("Incorrect DOSpaceName found: %s", configStruct.DigitalOcean.SpaceName)
	}
	if configStruct.DigitalOcean.SpaceKey != "hi: do_space_key" {
		t.Errorf("Incorrect DOSpaceKey found: %s", configStruct.DigitalOcean.SpaceKey)
	}
	if configStruct.DigitalOcean.SpaceSecret != "hi: do_space_secret" {
		t.Errorf("Incorrect DOSpaceSecret found: %s", configStruct.DigitalOcean.SpaceSecret)
	}
	if configStruct.Mysql.DataPath != "hi: mysql_data_path" {
		t.Errorf("Incorrect MysqlDataPath found: %s", configStruct.Mysql.DataPath)
	}
	if configStruct.PersistentStorage != "hi: persistent_storage" {
		t.Errorf("Incorrect PersistentStorage found: %s", configStruct.PersistentStorage)
	}
}

func TestLoadConfigOverrideFromCommandLine(t *testing.T) {
	setupTest()

	ioutil.WriteFile("_test_file.json", []byte(exampleLegacyJSONContents), 0755)
	defer os.Remove("_test_file.json")

	configStruct := loadConfig([]string{
		"-config",
		"_test_file.json",

		"-do-key",
		"do_key",
		"-do-space-endpoint",
		"do_space_endpoint",
		"-do-space-name",
		"do_space_name",
		"-do-space-key",
		"do_space_key",
		"-do-space-secret",
		"do_space_secret",
		"-mysql-data-path",
		"mysql_data_path",
		"-persistent-storage",
		"persistent_storage",
	})

	if configStruct.DigitalOcean.Key != "do_key" {
		t.Errorf("Incorrect DOKey found: %s", configStruct.DigitalOcean.Key)
	}
	if configStruct.DigitalOcean.SpaceEndpoint != "do_space_endpoint" {
		t.Errorf("Incorrect DOSpaceEndpoint found: %s", configStruct.DigitalOcean.SpaceEndpoint)
	}
	if configStruct.DigitalOcean.SpaceName != "do_space_name" {
		t.Errorf("Incorrect DOSpaceName found: %s", configStruct.DigitalOcean.SpaceName)
	}
	if configStruct.DigitalOcean.SpaceKey != "do_space_key" {
		t.Errorf("Incorrect DOSpaceKey found: %s", configStruct.DigitalOcean.SpaceKey)
	}
	if configStruct.DigitalOcean.SpaceSecret != "do_space_secret" {
		t.Errorf("Incorrect DOSpaceSecret found: %s", configStruct.DigitalOcean.SpaceSecret)
	}
	if configStruct.Mysql.DataPath != "mysql_data_path" {
		t.Errorf("Incorrect MysqlDataPath found: %s", configStruct.Mysql.DataPath)
	}
	if configStruct.PersistentStorage != "persistent_storage" {
		t.Errorf("Incorrect PersistentStorage found: %s", configStruct.PersistentStorage)
	}
}
