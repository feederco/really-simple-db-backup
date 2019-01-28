package cmd

import (
	"time"

	"github.com/minio/minio-go"
)

func backupDecide(retentionConfig *RetentionConfig, checkpointFilePath string, hostname string, doSpaceName string, minioClient *minio.Client) (string, error) {
	lastLsn, lastLsnErr := getLastLSNFromFile(checkpointFilePath)
	if lastLsnErr != nil || len(lastLsn) == 0 {
		return backupTypeFull, nil
	}

	allBackups, err := listAllBackups(hostname, doSpaceName, minioClient)
	if err != nil {
		return "", err
	}

	return decideBackupType(allBackups, time.Now(), retentionConfig), nil
}

func decideBackupType(allBackups []backupItem, nowTime time.Time, retentionConfig *RetentionConfig) string {
	backupsSince := findRelevantBackupsUpTo(nowTime, allBackups)

	// No good backups found, we need a full backup
	if len(backupsSince) == 0 {
		return backupTypeFull
	}

	lastFullbackup := backupsSince[len(backupsSince)-1]

	cutoffDate := lastFullbackup.CreatedAt.Add(time.Duration(retentionConfig.HoursBetweenFullBackups) * time.Hour)

	if cutoffDate.After(nowTime) {
		return backupTypeIncremental
	}

	return backupTypeFull
}
