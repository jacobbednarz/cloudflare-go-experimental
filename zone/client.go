package zone

import (
	"context"

	"github.com/jacobbednarz/cloudflare-go-experimental"
)

func Get() string {
	r, err := getClient().Call(context.Background(), "GET", "/zones", nil)
	if err != nil {
		return err.Error()
	}

	return string(r)
}

func getClient() *cloudflare.APIClient {
	return cloudflare.FetchClient()
}
