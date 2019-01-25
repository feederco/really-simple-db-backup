package cmd

import (
	"errors"
	"path"
	"strings"
	"time"

	"github.com/minio/minio-go"
)

type backupItem struct {
	Path       string
	Size       int64
	BackupType string
	CreatedAt  time.Time
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

		datum, err := newBackupItemFromMinioObject(item)
		if err != nil {
			continue
		}
		backupItems = append(backupItems, datum)
	}

	return backupItems, nil
}

func newBackupItemFromMinioObject(minioObject minio.ObjectInfo) (backupItem, error) {
	createdAt, backupType, err := parseBackupName(minioObject.Key)
	if err != nil {
		return backupItem{}, nil
	}

	return backupItem{
		Path:       minioObject.Key,
		CreatedAt:  createdAt,
		Size:       minioObject.Size,
		BackupType: backupType,
	}, nil
}

// Backup name is format mysql-backup-$TIMESTAMP.$BACKUP_TYPE.xbstream
func parseBackupName(backupPath string) (time.Time, string, error) {
	var err error
	var createdAt time.Time
	var backupType string

	fileName := path.Base(backupPath)
	pieces := strings.Split(fileName, ".")

	if len(pieces) != 3 {
		err = errors.New("Incorrect format for filename: " + fileName)
	} else if !strings.HasPrefix(pieces[0], "mysql-backup-") {
		err = errors.New("Incorrect prefix for filename: " + fileName)
	}

	if err == nil {
		timestampString := strings.Replace(pieces[0], "mysql-backup-", "", 1)
		createdAt, err = time.Parse("200601021504", timestampString)
	}

	if err == nil {
		backupTypePiece := pieces[1]
		if backupTypePiece != "full" && backupTypePiece != "incremental" {
			err = errors.New("Incorrect backup type: " + backupTypePiece)
		} else {
			backupType = backupTypePiece
		}
	}

	return createdAt, backupType, err
}
