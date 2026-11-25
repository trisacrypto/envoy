package config_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/config"
)

func TestTRISAConfig(t *testing.T) {
	// A valid TRISA config requires a path to the certs; the pool is optional.
	t.Run("Valid", func(t *testing.T) {
		tests := map[string]config.TRISAConfig{
			"CertsOnly": {
				MTLSConfig: config.MTLSConfig{Certs: certsFixture},
			},
			"CertsAndPool": {
				MTLSConfig: config.MTLSConfig{Certs: certsFixture, Pool: certsFixture},
			},
		}

		for name, conf := range tests {
			require.NoError(t, conf.Validate(), "expected %s config to be valid", name)
		}
	})

	t.Run("Invalid", func(t *testing.T) {
		tests := map[string]config.TRISAConfig{
			"Empty": {},
			"NoCerts": {
				Enabled:  true,
				BindAddr: ":9300",
				Endpoint: "trisa.example.dev:443",
				Directory: config.DirectoryConfig{
					Endpoint:        "api.testnet.directory:443",
					MembersEndpoint: "members.testnet.directory:443",
				},
			},
			"PoolOnly": {
				MTLSConfig: config.MTLSConfig{Pool: certsFixture},
			},
		}

		for name, conf := range tests {
			require.EqualError(t, conf.Validate(), "invalid configuration: trisa.certs is required but not set", "expected %s config to be invalid", name)
		}
	})
}

func TestDirectoryConfig(t *testing.T) {
	// Test directory network names
	testCases := []struct {
		endpoint string
		expected string
	}{
		{"", ""},
		{":443", ""},
		{"bufconn", "bufconn"},
		{"testing:123", "testing"},
		{"localhost:4433", "localhost"},
		{"testnet.directory", "testnet.directory"},
		{"testnet.directory:456", "testnet.directory"},
		{"api.testnet.directory", "testnet.directory"},
		{"api.testnet.directory:443", "testnet.directory"},
		{"api.trisa.directory", "trisa.directory"},
		{"api.trisa.directory:443", "trisa.directory"},
		{"testing.api.trisa.directory", "trisa.directory"},
		{"testing.api.trisa.directory:443", "trisa.directory"},
		{"https://api.trisa.directory", "trisa.directory"},
		{"https://api.trisa.directory:443", "trisa.directory"},
		{"grpc://api.testnet.directory:443/", "testnet.directory"},
	}

	for i, tc := range testCases {
		conf := config.DirectoryConfig{Endpoint: tc.endpoint}
		require.Equal(t, tc.expected, conf.Network(), "network name test case %d failed", i)
	}
}
