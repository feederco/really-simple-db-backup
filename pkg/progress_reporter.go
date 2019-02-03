package pkg

import (
	"time"

	"github.com/cheggaaa/pb"
)

const progressBarRecheckTime = 5

// ReportProgressOnFileSize will start printing the size of a file in relation to what the expected size is
func ReportProgressOnFileSize(location string, expectedSize int64) func() {
	bar := pb.StartNew(int(expectedSize))
	closed := true

	for !closed {
		size, err := FileOrDirSize(location)
		if err == nil {
			bar.Set(int(size))
		}

		time.Sleep(progressBarRecheckTime * time.Second)
	}

	return func() {
		closed = true
	}
}

// ReportProgressOnCopy will start printing the size of a file/directory in relation to another file/directory
// If source file/directory does not exist it will return immediately
func ReportProgressOnCopy(source string, destination string) func() {
	sourceSize, err := FileOrDirSize(source)
	if err != nil {
		return func() {}
	}

	return ReportProgressOnFileSize(destination, sourceSize)
}
