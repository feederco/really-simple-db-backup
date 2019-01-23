package cmd

import (
	"os"
	"path"

	"github.com/feederco/really-simple-db-backup/pkg"

	"github.com/minio/minio-go"
)

func backupMysqlUpload(backupFile string, backupsBucket string, minioClient *minio.Client) error {
	hostname, _ := os.Hostname()

	fileName := path.Base(backupFile)
	targetFileName := path.Join(hostname, fileName)

	return pkg.UploadFileToBucket(backupsBucket, targetFileName, backupFile, minioClient)
}
