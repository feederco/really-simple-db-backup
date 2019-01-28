package cmd

import (
	"testing"
)

func TestFindingOldBackupsToRemove(t *testing.T) {
	sevenDayRetentionConfig := &RetentionConfig{
		RetentionInDays: 3,
	}

	allBackups := []backupItem{
		buildBackup(1, "a/mysql-backup-201901011000.full.xbstream", 100),
		buildBackup(1, "a/mysql-backup-201901021000.incremental.xbstream", 1),
		buildBackup(1, "a/mysql-backup-201901031000.incremental.xbstream", 2),
		buildBackup(1, "a/mysql-backup-201901041000.incremental.xbstream", 3),
		buildBackup(2, "a/mysql-backup-201901051000.full.xbstream", 120),
		buildBackup(3, "a/mysql-backup-201901061000.full.xbstream", 130),
		buildBackup(3, "a/mysql-backup-201901071000.incremental.xbstream", 4),
		buildBackup(3, "a/mysql-backup-201901081000.incremental.xbstream", 5),
	}

	nowTime, _ := parseBackupTimestamp("201901091000") // Should delete from 201901051000 and down
	backupsToDelete := findBackupsThatCanBeDeleted(allBackups, nowTime, sevenDayRetentionConfig)

	if len(backupsToDelete) != 5 {
		t.Error("Incorrect backups to delete. Expected 5, found", len(backupsToDelete))
	}

	if backupsToDelete[0].Size != 120 {
		t.Error("Incorrect last backup found. Expected 120, found", backupsToDelete[0].Size)
	}

	if backupsToDelete[4].Size != 100 {
		t.Error("Incorrect last backup found. Expected 100, found", backupsToDelete[4].Size)
	}

	// Should not include any because that would mean removing the full backup but keep incrementals
	nowTime, _ = parseBackupTimestamp("201901061000")
	backupsToDelete = findBackupsThatCanBeDeleted(allBackups, nowTime, sevenDayRetentionConfig)

	if len(backupsToDelete) != 0 {
		t.Error("Incorrect backups to delete. Expected 0, found", len(backupsToDelete))
	}
}
