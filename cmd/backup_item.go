package cmd

import "time"

type backupItem struct {
	Path       string
	Size       int64
	BackupType string
	CreatedAt  time.Time

	LineageID int64 // An internal identifier to map the full backups and incrementals to the same lineage
}
