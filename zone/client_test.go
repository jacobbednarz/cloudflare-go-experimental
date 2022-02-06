package zone_test

import (
	"os"
	"testing"

	"github.com/jacobbednarz/cloudflare-go-experimental"
	"github.com/jacobbednarz/cloudflare-go-experimental/zone"
	"github.com/stretchr/testify/assert"
)

func TestGet(t *testing.T) {
	cloudflare.Key = os.Getenv("CLOUDFLARE_API_KEY")
	cloudflare.Email = os.Getenv("CLOUDFLARE_EMAIL")

	zones := zone.Get() // Outputs JSON but eventually, a real struct.

	assert.NotEmpty(t, zones)
}
