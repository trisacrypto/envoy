package config_test

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/config"
	"go.rtnl.ai/confire"
)

const (
	webhookKeyID     = "01JT4B3R5Z6AHJXV87QHPPKRBM"
	webhookKeySecret = "cfbabc4715b4759d45ba26953dd2fc0bfc2344ef70a2005432e7f16b5081610d"
)

func TestWebConfig(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		tests := map[string]config.WebConfig{
			// If the web server is disabled then nothing else is validated.
			"Disabled": {
				Enabled: false,
			},
			"DisabledWithAddr": {
				Enabled:    false,
				APIEnabled: false,
				UIEnabled:  false,
				BindAddr:   "127.0.0.1:0",
				Origin:     "http://localhost",
			},
			"APIAndUI": {
				Enabled:    true,
				APIEnabled: true,
				UIEnabled:  true,
				BindAddr:   "127.0.0.1:0",
				Origin:     "http://localhost",
			},
			"UIOnly": {
				Enabled:    true,
				APIEnabled: false,
				UIEnabled:  true,
				BindAddr:   "127.0.0.1:0",
				Origin:     "http://localhost",
			},
			"APIOnly": {
				Enabled:    true,
				APIEnabled: true,
				UIEnabled:  false,
				BindAddr:   "127.0.0.1:0",
				Origin:     "http://localhost",
			},
		}

		for name, conf := range tests {
			require.NoError(t, conf.Validate(), "expected %s config to be valid", name)
		}
	})

	t.Run("Invalid", func(t *testing.T) {
		tests := map[string]struct {
			conf config.WebConfig
			err  string
		}{
			"NoBindAddr": {
				conf: config.WebConfig{
					Enabled:    true,
					APIEnabled: true,
					UIEnabled:  true,
					Origin:     "http://localhost",
				},
				err: "invalid configuration: web.bindaddr is required but not set",
			},
			"NoOrigin": {
				conf: config.WebConfig{
					Enabled:    true,
					APIEnabled: true,
					UIEnabled:  true,
					BindAddr:   "127.0.0.1:0",
				},
				err: "invalid configuration: web.origin is required but not set",
			},
			"BadOrigin": {
				conf: config.WebConfig{
					Enabled:    true,
					APIEnabled: true,
					UIEnabled:  true,
					BindAddr:   "127.0.0.1:0",
					Origin:     "http://[::1",
				},
				err: `invalid configuration: web.origin "http://[::1" is not a valid url`,
			},
			"OriginNoScheme": {
				conf: config.WebConfig{
					Enabled:    true,
					APIEnabled: true,
					UIEnabled:  true,
					BindAddr:   "127.0.0.1:0",
					Origin:     "envoy.example.com",
				},
				err: `invalid configuration: web.origin "envoy.example.com" is not a valid url`,
			},
			"OriginNoHost": {
				// Without a scheme the host is parsed as the scheme of the url.
				conf: config.WebConfig{
					Enabled:    true,
					APIEnabled: true,
					UIEnabled:  true,
					BindAddr:   "127.0.0.1:0",
					Origin:     "localhost:8000",
				},
				err: `invalid configuration: web.origin "localhost:8000" is not a valid url`,
			},
			"NoAPIOrUI": {
				conf: config.WebConfig{
					Enabled:    true,
					APIEnabled: false,
					UIEnabled:  false,
					BindAddr:   "127.0.0.1:0",
					Origin:     "http://localhost",
				},
				err: "invalid configuration: web.enabled if enabled, either the api, ui, or both need to be enabled",
			},
		}

		for name, tc := range tests {
			require.EqualError(t, tc.conf.Validate(), tc.err, "expected %s config to be invalid", name)
		}
	})

	// Every problem with the config should be reported, not only the first one.
	t.Run("Multiple", func(t *testing.T) {
		conf := config.WebConfig{Enabled: true}

		errs, ok := confire.ValidationErrors(conf.Validate())
		require.True(t, ok, "expected validation errors to be returned")
		require.Len(t, errs, 3)
		require.EqualError(t, errs[0], "invalid configuration: web.enabled if enabled, either the api, ui, or both need to be enabled")
		require.EqualError(t, errs[1], "invalid configuration: web.bindaddr is required but not set")
		require.EqualError(t, errs[2], "invalid configuration: web.origin is required but not set")
	})

	t.Run("ResetPasswordURL", func(t *testing.T) {
		tests := []struct {
			origin   string
			expected string
		}{
			{"https://example.com", "https://example.com/reset-password"},
			{"https://example.com/", "https://example.com/reset-password"},
			{"http://localhost:8000", "http://localhost:8000/reset-password"},

			// An origin that cannot be parsed is rejected by validation if the web
			// server is enabled; otherwise the url has no host rather than panicking.
			{"http://[::1", "/reset-password"},
		}

		for _, tc := range tests {
			conf := config.WebConfig{Origin: tc.origin}
			require.Equal(t, tc.expected, conf.ResetPasswordURL().String(), "unexpected reset password url for origin %q", tc.origin)
		}
	})
}

func TestWebhookConfig(t *testing.T) {
	t.Run("Disabled", func(t *testing.T) {
		// If no webhook url is specified then nothing else is validated.
		tests := map[string]config.WebhookConfig{
			"Empty": {},
			"OtherwiseInvalid": {
				UseMTLS:           true,
				AuthKeySecret:     "not hex",
				RequireServerAuth: true,
			},
		}

		for name, conf := range tests {
			require.NoError(t, conf.Validate(), "expected %s config to be valid", name)
			require.False(t, conf.Enabled(), "expected %s config to be disabled", name)
			require.Nil(t, conf.Endpoint(), "expected %s config to have no endpoint", name)
		}
	})

	t.Run("Enabled", func(t *testing.T) {
		conf := config.WebhookConfig{URL: "https://example.com/callback"}
		require.NoError(t, conf.Validate(), "expected no error when webhook is specified")
		require.True(t, conf.Enabled(), "expected webhook enabled to be true with webhook specified")
		require.NotNil(t, conf.Endpoint())
		require.Equal(t, conf.URL, conf.Endpoint().String())
	})

	t.Run("Valid", func(t *testing.T) {
		tests := map[string]config.WebhookConfig{
			"NoAuth": {
				URL: "https://example.com/callback",
			},
			"ClientAuth": {
				URL:               "https://example.com/callback",
				AuthKeyID:         webhookKeyID,
				AuthKeySecret:     webhookKeySecret,
				RequireServerAuth: false,
			},
			"ClientAndServerAuth": {
				URL:               "https://example.com/callback",
				AuthKeyID:         webhookKeyID,
				AuthKeySecret:     strings.ToUpper(webhookKeySecret),
				RequireServerAuth: true,
			},
			"MTLS": {
				URL:        "https://example.com/callback",
				UseMTLS:    true,
				MTLSConfig: config.MTLSConfig{Certs: "fixtures/certs/webhook/certs.pem"},
			},
		}

		for name, conf := range tests {
			require.NoError(t, conf.Validate(), "expected %s config to be valid", name)
		}
	})

	t.Run("Invalid", func(t *testing.T) {
		tests := map[string]struct {
			conf config.WebhookConfig
			err  string
		}{
			"URL": {
				config.WebhookConfig{
					URL: "http://[::1/callback",
				},
				"invalid configuration: webhook.url is not a valid url",
			},
			"URLNoScheme": {
				config.WebhookConfig{
					URL: "example.com/callback",
				},
				"invalid configuration: webhook.url is not a valid url",
			},
			"URLNoHost": {
				config.WebhookConfig{
					URL: "https:///callback",
				},
				"invalid configuration: webhook.url is not a valid url",
			},
			"SecretWithoutKeyID": {
				config.WebhookConfig{
					URL:           "https://example.com/callback",
					AuthKeyID:     "",
					AuthKeySecret: webhookKeySecret,
				},
				"invalid configuration: webhook.authkeyid auth key id is required when auth key secret is set",
			},
			"SecretNotHex": {
				config.WebhookConfig{
					URL:           "https://example.com/callback",
					AuthKeyID:     webhookKeyID,
					AuthKeySecret: "foo",
				},
				"invalid configuration: webhook.authkeysecret could not hex decode auth secret",
			},
			"ServerAuthWithoutSecret": {
				config.WebhookConfig{
					URL:               "https://example.com/callback",
					AuthKeyID:         webhookKeyID,
					AuthKeySecret:     "",
					RequireServerAuth: true,
				},
				"invalid configuration: webhook.requireserverauth webhook server auth is enabled but no auth key is specified",
			},
			"ServerAuthWithoutKey": {
				config.WebhookConfig{
					URL:               "https://example.com/callback",
					RequireServerAuth: true,
				},
				"invalid configuration: webhook.requireserverauth webhook server auth is enabled but no auth key is specified",
			},
			"MTLSWithoutCerts": {
				config.WebhookConfig{
					URL:     "https://example.com/callback",
					UseMTLS: true,
				},
				"invalid configuration: webhook.certs certificates are required when use mtls is specified",
			},
			"MTLSPoolOnly": {
				config.WebhookConfig{
					URL:        "https://example.com/callback",
					UseMTLS:    true,
					MTLSConfig: config.MTLSConfig{Pool: "fixtures/certs/webhook/pool.pem"},
				},
				"invalid configuration: webhook.certs certificates are required when use mtls is specified",
			},
		}

		for name, tc := range tests {
			require.EqualError(t, tc.conf.Validate(), tc.err, "expected %s config to be invalid", name)
		}
	})

	// Every problem with the config should be reported, not only the first one.
	t.Run("Multiple", func(t *testing.T) {
		conf := config.WebhookConfig{
			URL:               "http://[::1/callback",
			UseMTLS:           true,
			AuthKeySecret:     "foo",
			RequireServerAuth: true,
		}

		errs, ok := confire.ValidationErrors(conf.Validate())
		require.True(t, ok, "expected validation errors to be returned")
		require.Len(t, errs, 5)
		require.EqualError(t, errs[0], "invalid configuration: webhook.url is not a valid url")
		require.EqualError(t, errs[1], "invalid configuration: webhook.authkeysecret could not hex decode auth secret")
		require.EqualError(t, errs[2], "invalid configuration: webhook.authkeyid auth key id is required when auth key secret is set")
		require.EqualError(t, errs[3], "invalid configuration: webhook.requireserverauth webhook server auth is enabled but no auth key is specified")
		require.EqualError(t, errs[4], "invalid configuration: webhook.certs certificates are required when use mtls is specified")
	})

	t.Run("RequireClientAuth", func(t *testing.T) {
		tests := []struct {
			keyID, secret string
			expected      bool
		}{
			{webhookKeyID, webhookKeySecret, true},
			{webhookKeyID, "", true},
			{"", webhookKeySecret, true},
			{"", "", false},
		}

		for i, tc := range tests {
			conf := config.WebhookConfig{
				URL:           "https://example.com/callback",
				AuthKeyID:     tc.keyID,
				AuthKeySecret: tc.secret,
			}
			require.Equal(t, tc.expected, conf.RequireClientAuth(), "test case %d failed", i)
		}
	})

	t.Run("DecodeAuthKey", func(t *testing.T) {
		expected, err := hex.DecodeString(webhookKeySecret)
		require.NoError(t, err, "could not decode auth key fixture")

		// The auth key should be decoded regardless of the case of the hex string.
		for _, secret := range []string{webhookKeySecret, strings.ToUpper(webhookKeySecret)} {
			conf := config.WebhookConfig{
				URL:           "https://example.com/callback",
				AuthKeyID:     webhookKeyID,
				AuthKeySecret: secret,
			}
			require.Equal(t, expected, conf.DecodeAuthKey())
		}

		conf := config.WebhookConfig{URL: "https://example.com/callback"}
		require.Nil(t, conf.DecodeAuthKey(), "expected nil auth key when not set")
	})
}
