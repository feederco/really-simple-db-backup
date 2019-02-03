package cmd

import (
	"path"
	"strings"
	"time"

	"github.com/digitalocean/godo"
	"github.com/feederco/really-simple-db-backup/pkg"
)

func createAndMountVolumeForUse(volumePrefix string, sizeInGb int64, digitalOceanClient *pkg.DigitalOceanClient, existingVolumeID string, existingDirectory string) (*godo.Volume, string, error) {
	if existingDirectory != "" {
		return nil, existingDirectory, nil
	}

	// - Fetch myself
	thisHost, err := pkg.GetRunningInstanceData()
	if err != nil {
		return nil, "", err
	}

	var volume *godo.Volume

	if existingVolumeID == "" {
		timeID := time.Now().Format("200601021504")

		volumeName := volumePrefix + timeID
		volumeDescription := "Volume created for a full MySQL backup on " + thisHost.Region + "." + thisHost.Hostname + " at " + timeID

		pkg.Log.Printf("Creating volume named %s with %d GB capacity.\n", volumeName, sizeInGb)

		// - Create volume the same size as MySQL data directory
		createRequest := &godo.VolumeCreateRequest{
			Region:         thisHost.Region,
			Name:           volumeName,
			Description:    volumeDescription,
			SizeGigaBytes:  sizeInGb,
			FilesystemType: fileSystemForVolume,
		}

		volume, err = pkg.CreateVolume(createRequest, digitalOceanClient)
		if err != nil {
			return volume, "", err
		}

		pkg.Log.Printf("Volume %s created.\n", volume.ID)
	} else {
		volume, err = pkg.FindVolume(existingVolumeID, digitalOceanClient)
		if err != nil {
			return nil, "", err
		}

		pkg.Log.Printf("Volume %s details retrieved.\n", volume.ID)
	}

	pkg.Log.Println("Volume is being mounted.")

	// - Mount that volume
	mountDirectory := path.Join("/mnt/", strings.Replace(volume.Name, "-", "_", -1))
	pkg.Log.Println("Volume is being mounted.", volume.DropletIDs)

	if len(volume.DropletIDs) == 0 || volume.DropletIDs[0] != thisHost.DropletID {
		err = pkg.MountVolume(volume.Name, mountDirectory, volume.ID, thisHost.DropletID, digitalOceanClient)

		if err != nil {
			pkg.AlertError(configStruct.Alerting, "Could not mount volume "+volume.ID, err)
			return volume, mountDirectory, err
		}

		if len(volume.DropletIDs) == 0 {
			volume.DropletIDs = append(volume.DropletIDs, thisHost.DropletID)
		}

		pkg.Log.Printf("Volume %s mounted on this host under %s.\n", volume.ID, mountDirectory)
	}

	return volume, mountDirectory, nil
}
