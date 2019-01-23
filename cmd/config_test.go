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

	if configStruct.DOKey != "do_key" {
		t.Errorf("Incorrect DOKey found: %s", configStruct.DOKey)
	}
	if configStruct.DOSpaceEndpoint != "do_space_endpoint" {
		t.Errorf("Incorrect DOSpaceEndpoint found: %s", configStruct.DOSpaceEndpoint)
	}
	if configStruct.DOSpaceName != "do_space_name" {
		t.Errorf("Incorrect DOSpaceName found: %s", configStruct.DOSpaceName)
	}
	if configStruct.DOSpaceKey != "do_space_key" {
		t.Errorf("Incorrect DOSpaceKey found: %s", configStruct.DOSpaceKey)
	}
	if configStruct.DOSpaceSecret != "do_space_secret" {
		t.Errorf("Incorrect DOSpaceSecret found: %s", configStruct.DOSpaceSecret)
	}
	if configStruct.MysqlDataPath != "mysql_data_path" {
		t.Errorf("Incorrect MysqlDataPath found: %s", configStruct.MysqlDataPath)
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

	if configStruct.DOKey != "hi: do_key" {
		t.Errorf("Incorrect DOKey found: %s", configStruct.DOKey)
	}
	if configStruct.DOSpaceEndpoint != "hi: do_space_endpoint" {
		t.Errorf("Incorrect DOSpaceEndpoint found: %s", configStruct.DOSpaceEndpoint)
	}
	if configStruct.DOSpaceName != "hi: do_space_name" {
		t.Errorf("Incorrect DOSpaceName found: %s", configStruct.DOSpaceName)
	}
	if configStruct.DOSpaceKey != "hi: do_space_key" {
		t.Errorf("Incorrect DOSpaceKey found: %s", configStruct.DOSpaceKey)
	}
	if configStruct.DOSpaceSecret != "hi: do_space_secret" {
		t.Errorf("Incorrect DOSpaceSecret found: %s", configStruct.DOSpaceSecret)
	}
	if configStruct.MysqlDataPath != "hi: mysql_data_path" {
		t.Errorf("Incorrect MysqlDataPath found: %s", configStruct.MysqlDataPath)
	}
	if configStruct.PersistentStorage != "hi: persistent_storage" {
		t.Errorf("Incorrect PersistentStorage found: %s", configStruct.PersistentStorage)
	}
}

func TestLoadConfigOverrideFromCommandLine(t *testing.T) {
	setupTest()

	ioutil.WriteFile("_test_file.json", []byte(exampleJSONContents), 0755)
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

	if configStruct.DOKey != "do_key" {
		t.Errorf("Incorrect DOKey found: %s", configStruct.DOKey)
	}
	if configStruct.DOSpaceEndpoint != "do_space_endpoint" {
		t.Errorf("Incorrect DOSpaceEndpoint found: %s", configStruct.DOSpaceEndpoint)
	}
	if configStruct.DOSpaceName != "do_space_name" {
		t.Errorf("Incorrect DOSpaceName found: %s", configStruct.DOSpaceName)
	}
	if configStruct.DOSpaceKey != "do_space_key" {
		t.Errorf("Incorrect DOSpaceKey found: %s", configStruct.DOSpaceKey)
	}
	if configStruct.DOSpaceSecret != "do_space_secret" {
		t.Errorf("Incorrect DOSpaceSecret found: %s", configStruct.DOSpaceSecret)
	}
	if configStruct.MysqlDataPath != "mysql_data_path" {
		t.Errorf("Incorrect MysqlDataPath found: %s", configStruct.MysqlDataPath)
	}
	if configStruct.PersistentStorage != "persistent_storage" {
		t.Errorf("Incorrect PersistentStorage found: %s", configStruct.PersistentStorage)
	}
}
