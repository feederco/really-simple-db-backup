package cmd

import (
	"sort"
	"time"

	minio "github.com/minio/minio-go"
)

func removeBackups(backups []backupItem, bucketName string, minioClient *minio.Client) ([]backupItem, error) {
	removedBackups := make([]backupItem, 0)
	for _, backup := range backups {
		err := minioClient.RemoveObject(bucketName, backup.Path)
		if err != nil {
			return removedBackups, err
		}
		removedBackups = append(removedBackups, backup)
	}
	return removedBackups, nil
}

func findBackupsThatCanBeDeleted(allBackups []backupItem, nowTime time.Time, retentionConfig *RetentionConfig) []backupItem {
	if retentionConfig.RetentionInDays <= 0 && retentionConfig.RetentionInHours <= 0 {
		return nil
	}

	sort.Sort(byCreatedAt(allBackups))

	// Find cut-off timestamp
	addDuration := time.Duration(retentionConfig.RetentionInHours) * time.Hour
	if retentionConfig.RetentionInDays > 0 {
		addDuration = time.Duration(retentionConfig.RetentionInDays) * (time.Hour * 24)
	}

	lastTimestamp := nowTime.Add(-addDuration)

	// Build a map of lineages. A lineage is only deleted if all backups are outside of the range
	// If you delete at the end of the lineage all subsequent increment backups fail
	lineages := make(map[int64][]backupItem)
	for _, backupItem := range allBackups {
		lineages[backupItem.LineageID] = append(lineages[backupItem.LineageID], backupItem)
	}

	oldBackups := make([]backupItem, 0)

	for _, backupItems := range lineages {
		allStale := true
		for _, backup := range backupItems {
			if backup.CreatedAt.After(lastTimestamp) {
				allStale = false
			}
		}

		if allStale {
			oldBackups = append(oldBackups, backupItems...)
		}
	}

	return oldBackups
}
