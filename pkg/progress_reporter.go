package pkg

import (
	"time"

	"github.com/cheggaaa/pb"
)

const progressBarRecheckTime = 1

// ReportProgressOnFileSize will start printing the size of a file in relation to what the expected size is
func ReportProgressOnFileSize(location string, expectedSize int64, doneChan chan bool) {
	bar := pb.StartNew(int(expectedSize))
	bar.SetUnits(pb.U_BYTES)
	closed := false

	go func() {
		<-doneChan
		closed = true
	}()

	for !closed {
		size, err := FileOrDirSize(location)
		if err == nil {
			bar.Set(int(size))
		}

		time.Sleep(progressBarRecheckTime * time.Second)
	}
}

// ReportProgressOnCopy will start printing the size of a file/directory in relation to another file/directory
// If source file/directory does not exist it will return immediately
func ReportProgressOnCopy(source string, destination string, doneChan chan bool) {
	sourceSize, err := FileOrDirSize(source)
	if err != nil {
		return
	}

	ReportProgressOnFileSize(destination, sourceSize, doneChan)
}
