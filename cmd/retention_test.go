package cmd

import (
	"errors"
	"testing"
	"time"
)

type retentionFilenameResult struct {
	Time       time.Time
	BackupType string
	Err        error
}

func TestParseFileName(t *testing.T) {
	tests := []string{
		"post-contents-db1/mysql-backup-201901232039.incremental.xbstream",
		"post-contents-db1/mysql-backup-201901232039.full.xbstream",
		"post-contents-db1/mysql-backup-201901232039.imploded.xbstream",
		"post-contents-db1/new-format-mysql-backup-201901232039.imploded.xbstream",
	}

	results := []retentionFilenameResult{
		retentionFilenameResult{time.Unix(1548275940, 0), "incremental", nil},
		retentionFilenameResult{time.Unix(1548275940, 0), "full", nil},
		retentionFilenameResult{time.Unix(1548275940, 0), "", errors.New("Incorrect backup type: imploded")},
		retentionFilenameResult{time.Time{}, "", errors.New("Incorrect prefix for filename: new-format-mysql-backup-201901232039.imploded.xbstream")},
	}

	for testIndex, test := range tests {
		createdAt, backupType, err := parseBackupName(test)
		result := results[testIndex]
		if result.Err != nil {
			if err == nil {
				t.Errorf("Failed test %d: Expected error but got none!", testIndex)
			} else if err.Error() != result.Err.Error() {
				t.Errorf("Failed test %d: Error not what expected: %s != %s", testIndex, err.Error(), result.Err.Error())
			}
		} else if err != nil {
			t.Errorf("Failed test %d: Expected no error but got one: %s", testIndex, err.Error())
		}
		if createdAt.Unix() != result.Time.Unix() {
			t.Errorf("Failed test %d: Timestamp not what expected: %d != %d", testIndex, createdAt.Unix(), result.Time.Unix())
		}
		if backupType != result.BackupType {
			t.Errorf("Failed test %d: BackupType not what expected: %s != %s", testIndex, backupType, result.BackupType)
		}
	}
}
