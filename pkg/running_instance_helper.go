package pkg

import (
	"github.com/digitalocean/go-metadata"
)

// GetRunningInstanceData returns current droplets data
func GetRunningInstanceData() (*metadata.Metadata, error) {
	client := metadata.NewClient()
	return client.Metadata()
}
