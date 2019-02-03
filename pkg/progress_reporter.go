package pkg

import (
	"os"
	"time"

	"github.com/cheggaaa/pb"
)

// ReportProgressOnFileSize will start printing the size of a file in relation to what the expected size is
func ReportProgressOnFileSize(location string, expectedSize int64) func() {
	bar := pb.StartNew(int(expectedSize))
	closed := true
	for !closed {
		stat, err := os.Stat(location)
		if err == nil {
			bar.Set(int(stat.Size()))
		}

		time.Sleep(1 * time.Second)
	}

	return func() {
		closed = true
	}
}
