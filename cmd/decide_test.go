package cmd

import "testing"

func TestDeciding(t *testing.T) {
	sevenDayRetentionConfig := &RetentionConfig{
		HoursBetweenFullBackups: 24 * 7,
	}

	zeroDayRetentionConfig := &RetentionConfig{
		HoursBetweenFullBackups: 0,
	}

	allBackups := []backupItem{
		buildBackup("a/mysql-backup-201901011000.full.xbstream", 100),
		buildBackup("a/mysql-backup-201901021000.incremental.xbstream", 1),
	}

	// 3 days after last full
	testTime, _ := parseBackupTimestamp("201901041000")
	backupType := decideBackupType(allBackups, testTime, sevenDayRetentionConfig)

	if backupType != backupTypeIncremental {
		t.Errorf("Incorrect backupType: %s (expected %s)", backupType, backupTypeIncremental)
	}

	// 7 days after last full
	testTime, _ = parseBackupTimestamp("201901071000")
	backupType = decideBackupType(allBackups, testTime, sevenDayRetentionConfig)

	if backupType != backupTypeIncremental {
		t.Errorf("Incorrect backupType: %s (expected %s)", backupType, backupTypeIncremental)
	}

	// 8 days after last full
	testTime, _ = parseBackupTimestamp("201901081000")
	backupType = decideBackupType(allBackups, testTime, sevenDayRetentionConfig)

	if backupType != backupTypeFull {
		t.Errorf("Incorrect backupType: %s (expected %s)", backupType, backupTypeFull)
	}

	// No incremental backups
	testTime, _ = parseBackupTimestamp("201901031000")
	backupType = decideBackupType(allBackups, testTime, zeroDayRetentionConfig)

	if backupType != backupTypeFull {
		t.Errorf("Incorrect backupType: %s (expected %s)", backupType, backupTypeFull)
	}
}
