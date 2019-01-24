package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"strings"

	"github.com/feederco/really-simple-db-backup/pkg"
)

const requiredMysqlVersion = "8"

func prerequisites(persistentStorageDirectory string) error {
	var err error

	// Make sure we are running as a DigitalOcean droplet
	_, err = pkg.GetRunningInstanceData()
	if err != nil {
		return err
	}

	var dirInfo os.FileInfo
	dirInfo, err = os.Stat(persistentStorageDirectory)
	if err != nil && os.IsNotExist(err) {
		pkg.Log.Println("Persistent storage directory did not exist. Attempting to create", persistentStorageDirectory)
		err = os.Mkdir(persistentStorageDirectory, 0755)
		if err != nil {
			pkg.ErrorLog.Fatalf("Could not create persistent storage directory at %s: %s", persistentStorageDirectory, err)
		}

		dirInfo, _ = os.Stat(persistentStorageDirectory)
	}

	if err != nil {
		pkg.ErrorLog.Println("Could not stat persistent storage directory.", err)
		return err
	}

	if !dirInfo.IsDir() {
		pkg.ErrorLog.Println("Persistent storage directory is not a directory.")
		return err
	}

	return nil
}

func backupPrerequisites() error {
	mysqlVersion, err := pkg.PerformCommand("mysqld", "--version")
	if err != nil {
		return err
	}

	versionString := strings.Split(mysqlVersion, " ")

	if versionString[2] == "Ver" {
		versionPieces := strings.Split(versionString[3], ".")
		if versionPieces[0] != requiredMysqlVersion {
			return fmt.Errorf("Incorrect MySQL version installed. error version. %s found, %s required", versionPieces[0], requiredMysqlVersion)
		}
	}

	// Prerequisite: percona-xtrabackup installed
	isXtrabackupInstalled, err := isBinaryInstalled("xtrabackup")
	if err != nil {
		return err
	}

	if !isXtrabackupInstalled {
		err = installXtrabackup()
		if err != nil {
			return err
		}
	}

	// Check if running as root
	err = checkCorrectUser()
	if err != nil {
		return err
	}
	return nil
}

func checkCorrectUser() error {
	requiredRunAs := "root"
	requiredUserID := "0" // Root in POSIX

	currentUser, err := user.Current()
	if err != nil {
		return err
	}

	if currentUser.Name != requiredRunAs && currentUser.Gid != requiredUserID {
		return errors.New("This program can only be run as " + requiredRunAs)
	}

	return nil
}

func installXtrabackup() error {
	outputFile := "/tmp/percona.deb"

	releaseName, err := pkg.PerformCommand("lsb_release", "-sc")
	if err != nil {
		return err
	}

	debURL := "https://repo.percona.com/apt/percona-release_latest." + strings.TrimSpace(releaseName) + "_all.deb"

	_, err = pkg.PerformCommand("wget", debURL, "--output-document", outputFile)

	if err != nil {
		return err
	}

	_, err = pkg.PerformCommand("dpkg", "-i", outputFile)
	if err != nil {
		return err
	}

	_, err = pkg.PerformCommand("percona-release", "enable-only", "tools", "release")
	if err != nil {
		return err
	}

	_, err = pkg.PerformCommand("apt-get", "update")
	if err != nil {
		return err
	}

	_, err = pkg.PerformCommand("apt-get", "install", "percona-xtrabackup-80", "-y")
	if err != nil {
		return err
	}

	return nil
}

func isBinaryInstalled(binaryName string) (bool, error) {
	_, err := pkg.PerformCommand("which", binaryName)
	return err == nil, nil
}
