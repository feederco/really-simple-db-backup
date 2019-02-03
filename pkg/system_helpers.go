package pkg

import (
	"os"
	"path/filepath"
)

// FileOrDirSize gets the size of a directory or file
func FileOrDirSize(path string) (int64, error) {
	fileStat, err := os.Stat(path)
	if err != nil {
		return 0, err
	}

	if fileStat.IsDir() {
		return DirSize(path)
	}

	return fileStat.Size(), nil
}

// DirSize get size of directory
func DirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	return size, err
}
