package cmd

import (
	"os"
	"path"
	"time"

	"github.com/feederco/really-simple-db-backup/pkg"
	minio "github.com/minio/minio-go"
)

func backupMysqlUpload(backupFile string, backupsBucket string, minioClient *minio.Client) error {
	pkg.Log.Println("Backup started", time.Now().Format(time.RFC3339))
	defer pkg.Log.Println("Backup ended", time.Now().Format(time.RFC3339))

	hostname, _ := os.Hostname()

	fileName := path.Base(backupFile)
	targetFileName := path.Join(hostname, fileName)

	return pkg.UploadFileToBucket(backupsBucket, targetFileName, backupFile, minioClient)
}
