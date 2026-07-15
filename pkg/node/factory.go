package node

import (
	"strings"

	"github.com/trisacrypto/envoy/pkg/config"
	"github.com/trisacrypto/envoy/pkg/store/models"
)

func TravelAddressFactory(conf config.Config) (models.TravelAddressFactory, error) {
	if conf.TRP.Enabled {
		// Remove any http:// or https:// prefix from the endpoint
		endpoint := strings.TrimPrefix(conf.TRP.Endpoint, "http://")
		endpoint = strings.TrimPrefix(endpoint, "https://")

		return models.NewTravelAddressFactory(endpoint, "trp")
	}
	return models.NewTravelAddressFactory(conf.Node.Endpoint, "trisa")
}
