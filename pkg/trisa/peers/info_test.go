package peers_test

import (
	"testing"

	"github.com/trisacrypto/envoy/pkg/trisa/peers"

	"github.com/stretchr/testify/require"
)

func TestInfo(t *testing.T) {
	// Build a valid info from cratch
	info := &peers.Info{}
	require.Error(t, info.Validate(), "empty info should not be valid")

	info.CommonName = ""
	info.Endpoint = "trisa.example.com:443"
	require.ErrorIs(t, info.Validate(), peers.ErrNoCommonName, "common name should be required")

	info.CommonName = "trisa.example.com"
	info.Endpoint = ""
	require.ErrorIs(t, info.Validate(), peers.ErrNoEndpoint, "endpoint should be required")

	info.CommonName = "trisa.example.com"
	info.Endpoint = "trisa.example.com:443"
	require.NoError(t, info.Validate(), "expected valid info object")
}
