package pkg

import (
	"os"

	"github.com/cheggaaa/pb"
	"github.com/minio/minio-go"
)

// UploadFileToBucket uploads a file to a DigitalOcean bucket
func UploadFileToBucket(bucketName string, objectName string, filePath string, minioClient *minio.Client) error {
	stat, err := os.Stat(filePath)
	if err != nil {
		return err
	}

	progress := pb.New64(stat.Size())
	progress.Start()

	_, err = minioClient.FPutObject(bucketName, objectName, filePath, minio.PutObjectOptions{
		Progress: progress,
	})

	// github.com/cheggaaa/pb does not end output with a newline. Add one here
	Log.Print("\n")

	return err
}

// DownloadFileFromBucket downloads a file
func DownloadFileFromBucket() error {
	return nil
}
