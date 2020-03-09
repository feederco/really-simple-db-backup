package pkg

import (
	"time"

	"github.com/digitalocean/go-metadata"
)

// GetRunningInstanceData returns current droplets data
func GetRunningInstanceData() (*metadata.Metadata, error) {
	var err error
	var result *metadata.Metadata
	for tries := 0; tries < 5; tries++ {
		client := metadata.NewClient()
		result, err = client.Metadata()
		if err == nil {
			return result, nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return nil, err
}
