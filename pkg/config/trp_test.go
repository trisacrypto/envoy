package config_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/config"
	"go.rtnl.ai/confire"
)

func TestTRPConfig(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		tests := map[string]config.TRPConfig{
			// If the TRP server is disabled then nothing else is validated.
			"Disabled": {
				Enabled: false,
				UseMTLS: true,
			},
			"NoMTLS": {
				Endpoint: "https://trp.example.com",
				Enabled:  true,
				BindAddr: ":8012",
			},
			"MTLSCertsOnly": {
				Endpoint: "https://trp.example.com",
				Enabled:  true,
				BindAddr: ":8012",
				UseMTLS:  true,
				MTLSConfig: config.MTLSConfig{
					Certs: "fixtures/certs/trp/certs.pem",
				},
			},
			"MTLSCertsAndPool": {
				Endpoint: "https://trp.example.com",
				Enabled:  true,
				BindAddr: ":8012",
				UseMTLS:  true,
				MTLSConfig: config.MTLSConfig{
					Certs: "fixtures/certs/trp/certs.pem",
					Pool:  "fixtures/certs/trp/pool.pem",
				},
			},
		}

		for name, conf := range tests {
			require.NoError(t, conf.Validate(), "expected %s config to be valid", name)
		}
	})

	t.Run("Invalid", func(t *testing.T) {
		tests := map[string]struct {
			conf config.TRPConfig
			err  string
		}{
			"NoEndpoint": {
				conf: config.TRPConfig{
					Endpoint: "",
					Enabled:  true,
					BindAddr: ":8012",
				},
				err: "invalid configuration: trp.endpoint is required but not set",
			},
			"NoBindAddr": {
				conf: config.TRPConfig{
					Endpoint: "https://trp.example.com",
					Enabled:  true,
					BindAddr: "",
				},
				err: "invalid configuration: trp.bindaddr is required but not set",
			},
			"MTLSNoCerts": {
				conf: config.TRPConfig{
					Endpoint: "https://trp.example.com",
					Enabled:  true,
					BindAddr: ":8012",
					UseMTLS:  true,
				},
				err: "invalid configuration: trp.certs certificates are required when use mtls is specified",
			},
			"MTLSPoolOnly": {
				conf: config.TRPConfig{
					Endpoint: "https://trp.example.com",
					Enabled:  true,
					BindAddr: ":8012",
					UseMTLS:  true,
					MTLSConfig: config.MTLSConfig{
						Pool: "fixtures/certs/trp/pool.pem",
					},
				},
				err: "invalid configuration: trp.certs certificates are required when use mtls is specified",
			},
		}

		for name, tc := range tests {
			require.EqualError(t, tc.conf.Validate(), tc.err, "expected %s config to be invalid", name)
		}
	})

	// Every problem with the config should be reported, not only the first one.
	t.Run("Multiple", func(t *testing.T) {
		conf := config.TRPConfig{Enabled: true, UseMTLS: true}

		errs, ok := confire.ValidationErrors(conf.Validate())
		require.True(t, ok, "expected validation errors to be returned")
		require.Len(t, errs, 3)
		require.EqualError(t, errs[0], "invalid configuration: trp.endpoint is required but not set")
		require.EqualError(t, errs[1], "invalid configuration: trp.bindaddr is required but not set")
		require.EqualError(t, errs[2], "invalid configuration: trp.certs certificates are required when use mtls is specified")
	})
}
