package cmd

import (
	"io/ioutil"
	"os"
	"testing"
)

const testLNSContents = `tool_version = 8.0.4
ibbackup_version = 8.0.4
server_version = 8.0.13
start_time = 2019-01-21 20:19:33
end_time = 2019-01-21 23:27:58
lock_time = 0
binlog_pos = filename 'binlog.000307', position '965530976'
innodb_from_lsn = 0
innodb_to_lsn = 422046960431
partial = N
incremental = N
format = xbstream
compressed = compressed
encrypted = N`

const testLNSContentsOther = `backup_type = full-backuped
from_lsn = 0
to_lsn = 422046960431
last_lsn = 422047039856`

const badTestLNSContents = `tool_version = 8.0.4
ibbackup_version = 8.0.4
server_version = 8.0.13
start_time = 2019-01-21 20:19:33
end_time = 2019-01-21 23:27:58
lock_time = 0
binlog_pos = filename 'binlog.000307', position '965530976'
innodb_from_lsn = 0
partial = N
incremental = N
format = xbstream
compressed = compressed
encrypted = N`

func TestGetLastLSNFromFile(t *testing.T) {
	testFileName := "./_testing_lsn_good"
	testFileNameBad := "./_testing_lsn_bad"
	testFileNameOther := "./_testing_lsn_other"

	ioutil.WriteFile(testFileName, []byte(testLNSContents), 0755)
	defer os.Remove(testFileName)

	ioutil.WriteFile(testFileNameBad, []byte(badTestLNSContents), 0755)
	defer os.Remove(testFileNameBad)

	ioutil.WriteFile(testFileNameOther, []byte(testLNSContentsOther), 0755)
	defer os.Remove(testFileNameOther)

	// Good example
	lsn, err := getLastLSNFromFile(testFileName)
	if err != nil {
		t.Error("No error expected", err)
	}

	if lsn != "422046960431" {
		t.Error("Incorrect LSN found from file", lsn)
	}

	// Other example
	lsn, err = getLastLSNFromFile(testFileName)
	if err != nil {
		t.Error("No error expected", err)
	}

	if lsn != "422046960431" {
		t.Error("Incorrect LSN found from file", lsn)
	}

	// Bad example
	lsn, err = getLastLSNFromFile(testFileNameBad)
	if err != nil {
		t.Error("No error expected", err)
	}

	if lsn != "" {
		t.Error("Didn't expect to find an LSN", lsn)
	}

	// No file example
	_, err = getLastLSNFromFile("no i dont exist.txt")
	if err == nil {
		t.Error("Error expected", err)
	}
}
