package cmd

import (
	"errors"
	"path"
	"sort"
	"strings"
	"time"

	minio "github.com/minio/minio-go"
)

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

		backupItem, err := newBackupItemFromMinioObject(item)
		if err != nil {
			continue
		}
		backupItems = append(backupItems, backupItem)
	}

	sort.Sort(byCreatedAt(backupItems))

	lineageID := int64(1)
	for _, backupItem := range backupItems {
		if backupItem.BackupType == backupTypeFull {
			backupItem.LineageID = lineageID
			lineageID++
		} else {
			backupItem.LineageID = lineageID
		}

	}

	return backupItems, nil
}

func findRelevantBackupsUpTo(sinceTimestamp time.Time, allBackups []backupItem) []backupItem {
	if len(allBackups) == 0 {
		return nil
	}

	sort.Sort(byCreatedAt(allBackups))

	backups := make([]backupItem, 0)
	for _, backup := range allBackups {
		if backup.CreatedAt.Unix() > sinceTimestamp.Unix() {
			continue
		}

		backups = append(backups, backup)

		if backup.BackupType == backupTypeFull {
			break
		}
	}

	if len(backups) == 0 {
		return nil
	}

	// Only found incremental backups, return nothing because we don't have anything to go on
	if backups[len(backups)-1].BackupType != backupTypeFull {
		return nil
	}

	return backups
}

func newBackupItemFromMinioObject(minioObject minio.ObjectInfo) (backupItem, error) {
	createdAt, backupType, err := parseBackupName(minioObject.Key)
	if err != nil {
		return backupItem{}, err
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
		createdAt, err = parseBackupTimestamp(timestampString)
	}

	if err == nil {
		backupTypePiece := pieces[1]
		if backupTypePiece != backupTypeFull && backupTypePiece != backupTypeIncremental {
			err = errors.New("Incorrect backup type: " + backupTypePiece)
		} else {
			backupType = backupTypePiece
		}
	}

	return createdAt, backupType, err
}

func parseBackupTimestamp(timestamp string) (time.Time, error) {
	return time.Parse("200601021504", timestamp)
}

type byCreatedAt []backupItem

func (sorter byCreatedAt) Len() int {
	return len(sorter)
}

func (sorter byCreatedAt) Swap(i, j int) {
	sorter[i], sorter[j] = sorter[j], sorter[i]
}

func (sorter byCreatedAt) Less(i, j int) bool {
	return sorter[i].CreatedAt.Unix() > sorter[j].CreatedAt.Unix()
}
