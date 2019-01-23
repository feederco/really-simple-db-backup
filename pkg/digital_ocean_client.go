package pkg

import (
	"context"

	"github.com/digitalocean/godo"
	"golang.org/x/oauth2"
)

// DigitalOceanClient is a wrapper to be able to make calls to the DigitalOcean API
type DigitalOceanClient struct {
	AccessToken string
	Context     context.Context
	Client      *godo.Client
}

// Token exists to conform to the oauth2.TokenSource interface
func (t *DigitalOceanClient) Token() (*oauth2.Token, error) {
	token := &oauth2.Token{
		AccessToken: t.AccessToken,
	}
	return token, nil
}

// NewDigitalOceanClient creates a client
func NewDigitalOceanClient(accessToken string) *DigitalOceanClient {
	digitalOceanClient := &DigitalOceanClient{
		AccessToken: accessToken,
		Context:     context.Background(),
	}

	oauthClient := oauth2.NewClient(digitalOceanClient.Context, digitalOceanClient)
	digitalOceanClient.Client = godo.NewClient(oauthClient)

	return digitalOceanClient
}
