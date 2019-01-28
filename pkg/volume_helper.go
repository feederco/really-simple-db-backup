package pkg

import (
	"errors"
	"os"
	"strings"
	"time"

	"github.com/digitalocean/godo"
)

// FindVolume finds a DigitalOcean volume by ID
func FindVolume(id string, digitalOceanClient *DigitalOceanClient) (*godo.Volume, error) {
	volume, _, err := digitalOceanClient.Client.Storage.GetVolume(
		digitalOceanClient.Context,
		id,
	)

	return volume, err
}

// CreateVolume creates a DigitalOcean volume
func CreateVolume(createRequest *godo.VolumeCreateRequest, digitalOceanClient *DigitalOceanClient) (*godo.Volume, error) {
	volume, _, err := digitalOceanClient.Client.Storage.CreateVolume(
		digitalOceanClient.Context,
		createRequest,
	)

	return volume, err
}

// MountVolume mounts volume on machine and won't return until complete
func MountVolume(volumeName string, volumeID string, dropletID int, digitalOceanClient *DigitalOceanClient) (string, error) {
	action, _, err := digitalOceanClient.Client.StorageActions.Attach(
		digitalOceanClient.Context,
		volumeID,
		dropletID,
	)

	if err != nil {
		return "", err
	}

	if action.Status == "errored" {
		return "", errors.New("Attach action had a status of errored")
	}

	err = digitalOceanWaitForAction(action, digitalOceanClient)
	if err != nil {
		return "", err
	}

	// Wait for system to settle
	Log.Println("Mounted volume on host.")
	Log.Println("Waiting for system to settle down.")
	time.Sleep(30 * time.Second)

	// Volume is now available in /dev/disk/by-id/scsi-0DO_Volume_$VOLUME_NAME
	diskLocation := "/dev/disk/by-id/scsi-0DO_Volume_" + volumeName
	mountPoint := "/mnt/" + strings.Replace(volumeName, "-", "_", -1)

	err = os.MkdirAll(mountPoint, 0755)
	if err != nil {
		return "", err
	}

	_, err = PerformCommand("mount", "-o", "discard,defaults,noatime", diskLocation, mountPoint)

	// In my tests this command returned 32 for any reason even though the mount was correct
	if err != nil {
		Log.Println("mount indicated it did not succeed. Running manual test to confirm", err)
		time.Sleep(30 * time.Second)

		file, testErr := os.Create(mountPoint + "/test-mount")
		if testErr != nil {
			return mountPoint, testErr
		}
		file.Close()

		Log.Println("Manual test succeeded! Continuing anyway.")

		err = nil
	}

	return mountPoint, err
}

// UnmountVolume unmounts a volume
func UnmountVolume(mountPoint string, volumeID string, dropletID int, digitalOceanClient *DigitalOceanClient) error {
	// - Unmount mountpoint
	_, err := PerformCommand("umount", mountPoint)
	if err != nil {
		return err
	}

	// - Remove mountpoint directory
	err = os.Remove(mountPoint)
	if err != nil {
		Log.Println("Warning: Could not remove mount directory. Continuing anyway.", err)
	}

	// - Detach volume
	action, _, err := digitalOceanClient.Client.StorageActions.DetachByDropletID(
		digitalOceanClient.Context,
		volumeID,
		dropletID,
	)

	if err != nil {
		return err
	}

	err = digitalOceanWaitForAction(action, digitalOceanClient)

	return err
}

// DestroyVolume destroys a volume
func DestroyVolume(volumeID string, digitalOceanClient *DigitalOceanClient) error {
	_, err := digitalOceanClient.Client.Storage.DeleteVolume(
		digitalOceanClient.Context,
		volumeID,
	)

	return err
}

func digitalOceanWaitForAction(action *godo.Action, digitalOceanClient *DigitalOceanClient) error {
	var err error

	times := 0
	for times < 10 {
		updatedAction, _, err := digitalOceanClient.Client.Actions.Get(
			digitalOceanClient.Context,
			action.ID,
		)

		if err != nil || updatedAction.Status != "completed" {
			times++
			time.Sleep(15 * time.Second)
			continue
		}

		break
	}

	return err
}
