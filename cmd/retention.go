package cmd

import (
	"errors"
	"time"

	"github.com/minio/minio-go"
)

type backupItem struct {
	Path          string
	Size          int64
	IsIncremental string
	CreatedAt     time.Time
}

func getLastFullBackup() (time.Time, error) {
	return time.Now(), errors.New("WIP")
}

func listAllBackups(hostname string, doSpaceName string, minioClient *minio.Client) ([]backupItem, error) {
	backupKey := hostname

	items := minioClient.ListObjectsV2(
		doSpaceName,
		backupKey,
		true,
		nil,
	)

	backupItems := make([]backupItem, 0)

	for item := range items {
		if item.Err != nil {
			return nil, item.Err
		}

		backupItems = append(backupItems, newBackupItemFromMinioObject(item))
	}

	return backupItems, nil
}

func newBackupItemFromMinioObject(minioObject *minio.ObjectInfo) backupItem {
	return backupItem{
		Path:      minioObject.Key,
		CreatedAt: time.Now(),
		Size:      minioObject.Size,
	}
}

func parseBackupName(fileName string) (time.Time, bool, error) {
	return nil, false, nil
}
