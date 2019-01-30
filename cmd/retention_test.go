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

func TestListingBackupsSince(t *testing.T) {
	allBackups := []backupItem{
		buildBackup(1, "a/mysql-backup-201812311000.incremental.xbstream", 0),
		buildBackup(2, "a/mysql-backup-201901011000.full.xbstream", 100),
		buildBackup(2, "a/mysql-backup-201901021000.incremental.xbstream", 1),
		buildBackup(2, "a/mysql-backup-201901031000.incremental.xbstream", 2),
		buildBackup(2, "a/mysql-backup-201901041000.incremental.xbstream", 3),
		buildBackup(3, "a/mysql-backup-201901051000.full.xbstream", 110),
		buildBackup(3, "a/mysql-backup-201901061000.incremental.xbstream", 4),
		buildBackup(3, "a/mysql-backup-201901071000.incremental.xbstream", 5),
		buildBackup(4, "a/mysql-backup-201901081000.full.xbstream", 120),
		buildBackup(4, "a/mysql-backup-201901091000.incremental.xbstream", 6),
	}

	// Test in the middle of the history
	sinceTimestamp, _ := parseBackupTimestamp("201901071000")
	backups := findRelevantBackupsUpTo(sinceTimestamp, allBackups)

	if len(backups) != 3 {
		t.Error("Found incorrect backups")
	}
	if backups[0].Size != 5 {
		t.Errorf("Wrong backup 0, found: %d", backups[0].Size)
	}
	if backups[1].Size != 4 {
		t.Errorf("Wrong backup 1, found: %d", backups[1].Size)
	}
	if backups[2].Size != 110 {
		t.Errorf("Wrong backup 2, found: %d", backups[2].Size)
	}

	// Test at the end of the history
	sinceTimestamp, _ = parseBackupTimestamp("201901101000")
	backups = findRelevantBackupsUpTo(sinceTimestamp, allBackups)

	if len(backups) != 2 {
		t.Error("Found incorrect backups")
	}
	if backups[0].Size != 6 {
		t.Errorf("Wrong backup 0, found: %d", backups[0].Size)
	}
	if backups[1].Size != 120 {
		t.Errorf("Wrong backup 1, found: %d", backups[1].Size)
	}

	// Test finding nothing
	sinceTimestamp, _ = parseBackupTimestamp("201801101000")
	backups = findRelevantBackupsUpTo(sinceTimestamp, allBackups)

	if len(backups) != 0 {
		t.Error("Found incorrect backups. Expected nothing")
	}

	// Test finding only an incremental backup should return 0
	sinceTimestamp, _ = parseBackupTimestamp("201812311000")
	backups = findRelevantBackupsUpTo(sinceTimestamp, allBackups)

	if len(backups) != 0 {
		t.Error("Found incorrect backups. Expected nothing")
	}
}

func TestComputingLineages(t *testing.T) {
	allBackups := []backupItem{
		buildBackup(0, "prod-test-db1/mysql-backup-201901281617.full.xbstream", 126),
		buildBackup(0, "prod-test-db1/mysql-backup-201901232039.incremental.xbstream", 4),
		buildBackup(0, "prod-test-db1/mysql-backup-201901231002.incremental.xbstream", 6),
		buildBackup(0, "prod-test-db1/mysql-backup-201901220907.incremental.xbstream", 5),
		buildBackup(0, "prod-test-db1/mysql-backup-201901211952.full.xbstream", 119),
	}

	results := []int64{
		1,
		2,
		2,
		2,
		2,
	}

	allBackups = addLineagesToBackups(allBackups)

	for index, backup := range allBackups {
		if backup.LineageID != results[index] {
			t.Errorf("Incorrect result for index #%d, expected %d got %d", index, results[index], backup.LineageID)
		}
	}
}

func TestFailingBackupScenario(t *testing.T) {
	allBackups := []backupItem{
		buildBackup(0, "prod-test-db1/mysql-backup-201901281617.full.xbstream", 126),
		buildBackup(0, "prod-test-db1/mysql-backup-201901232039.incremental.xbstream", 4),
		buildBackup(0, "prod-test-db1/mysql-backup-201901231002.incremental.xbstream", 6),
		buildBackup(0, "prod-test-db1/mysql-backup-201901220907.incremental.xbstream", 5),
		buildBackup(0, "prod-test-db1/mysql-backup-201901211952.full.xbstream", 119),
	}

	allBackups = addLineagesToBackups(allBackups)

	// Test in the middle of the history
	sinceTimestamp, _ := parseBackupTimestamp("201901281617")
	backups := findBackupsThatCanBeDeleted(allBackups, sinceTimestamp, &RetentionConfig{
		RetentionInDays: 4,
	})

	if len(backups) != 4 {
		t.Error("Found incorrect backups", len(backups))
	}
	if backups[0].Size != 4 {
		t.Errorf("Wrong backup 0, found: %d", backups[0].Size)
	}
	if backups[1].Size != 6 {
		t.Errorf("Wrong backup 1, found: %d", backups[1].Size)
	}
	if backups[2].Size != 5 {
		t.Errorf("Wrong backup 2, found: %d", backups[2].Size)
	}
	if backups[3].Size != 119 {
		t.Errorf("Wrong backup 3, found: %d", backups[3].Size)
	}
}

func buildBackup(lineageID int64, name string, size int64) backupItem {
	createdAt, backupType, _ := parseBackupName(name)
	return backupItem{
		Path:       name,
		Size:       size,
		BackupType: backupType,
		CreatedAt:  createdAt,

		LineageID: lineageID,
	}
}
