package cmd

import (
	"time"

	"github.com/minio/minio-go"
)

func backupPrune(retentionConfig *RetentionConfig, hostname string, bucketName string, minioClient *minio.Client) error {
	return nil
}

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

func findBackupsThatCanBeDeleted(retentionConfig *RetentionConfig, hostname string, bucketName string, minioClient *minio.Client) ([]backupItem, error) {
	if retentionConfig.RetentionInDays <= 0 {
		return nil, nil
	}

	allBackups, err := listAllBackups(hostname, bucketName, minioClient)
	if err != nil {
		return nil, err
	}

	lastTimestamp := time.Now().Add(-time.Duration(retentionConfig.RetentionInDays) * (time.Hour * 24))
	return findRelevantBackupsUpTo(lastTimestamp, allBackups), nil
}
