package node

import (
	"github.com/trisacrypto/envoy/pkg/config"
	"github.com/trisacrypto/envoy/pkg/store/models"
)

func TravelAddressFactory(conf config.Config) (models.TravelAddressFactory, error) {
	if conf.TRP.Enabled {
		return models.NewTravelAddressFactory(conf.TRP.Endpoint, "trp")
	}
	return models.NewTravelAddressFactory(conf.Node.Endpoint, "trisa")
}
