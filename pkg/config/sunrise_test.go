package config_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/config"
	"go.rtnl.ai/confire"
)

func TestSunriseConfig(t *testing.T) {
	t.Run("InviteURL", func(t *testing.T) {
		tests := []struct {
			conf     config.SunriseConfig
			expected string
		}{
			{
				config.SunriseConfig{BaseURL: "https://example.com", InviteEndpoint: "/sunrise/review"},
				"https://example.com/sunrise/review",
			},
			{
				config.SunriseConfig{BaseURL: "http://localhost:8000", InviteEndpoint: "/sunrise/verify"},
				"http://localhost:8000/sunrise/verify",
			},
			{
				config.SunriseConfig{BaseURL: "https://example.com/", InviteEndpoint: "/sunrise/verify"},
				"https://example.com/sunrise/verify",
			},
			{
				// A base url that cannot be parsed results in a url with no host
				// rather than a panic.
				config.SunriseConfig{BaseURL: "http://[::1", InviteEndpoint: "/sunrise/verify"},
				"/sunrise/verify",
			},
		}

		for i, tc := range tests {
			require.Equal(t, tc.expected, tc.conf.InviteURL().String(), "test case %d failed", i)

			// The invite URL should not change on subsequent calls.
			require.Equal(t, tc.expected, tc.conf.InviteURL().String(), "test case %d changed on second call", i)
		}
	})

	// The caller adds the verification token to the invite URL, so each call must
	// return a new URL: one invitation's token must not appear in the next one.
	t.Run("NotShared", func(t *testing.T) {
		conf := config.SunriseConfig{BaseURL: "https://example.com", InviteEndpoint: "/sunrise/verify"}

		invite := conf.InviteURL()
		invite.RawQuery = "token=secret"

		require.NotSame(t, invite, conf.InviteURL(), "expected a new url on each call")
		require.Equal(t, "https://example.com/sunrise/verify", conf.InviteURL().String())
	})
}

func TestDaybreakConfig(t *testing.T) {
	// Returns a valid, enabled Daybreak config for the tests to modify.
	valid := func() config.DaybreakConfig {
		return config.DaybreakConfig{
			Enabled:      true,
			Endpoint:     "https://daybreaks.directory",
			ClientID:     "1234567890",
			ClientSecret: "0987654321",
			SyncInterval: 5 * time.Minute,
			SyncJitter:   1 * time.Minute,
		}
	}

	t.Run("Disabled", func(t *testing.T) {
		// If Daybreak is disabled then nothing else is validated.
		tests := map[string]config.DaybreakConfig{
			"Empty": {},
			"OtherwiseInvalid": {
				Enabled:      false,
				Endpoint:     "http://[::1",
				SyncInterval: 1 * time.Second,
			},
		}

		for name, conf := range tests {
			require.NoError(t, conf.Validate(), "expected %s config to be valid", name)
		}
	})

	t.Run("Valid", func(t *testing.T) {
		conf := valid()
		require.NoError(t, conf.Validate(), "expected the config to be valid")

		// The sync interval and jitter must be greater than their limits.
		conf.SyncInterval = time.Minute + time.Second
		conf.SyncJitter = 2 * time.Second
		require.NoError(t, conf.Validate(), "expected the config to be valid above the sync limits")
	})

	t.Run("Invalid", func(t *testing.T) {
		tests := map[string]struct {
			mutate func(*config.DaybreakConfig)
			err    string
		}{
			"NoEndpoint": {
				func(c *config.DaybreakConfig) { c.Endpoint = "" },
				"invalid configuration: daybreak.endpoint is required but not set",
			},
			"BadEndpoint": {
				func(c *config.DaybreakConfig) { c.Endpoint = "http://[::1" },
				"invalid configuration: daybreak.endpoint not a valid url",
			},
			"EndpointNoScheme": {
				func(c *config.DaybreakConfig) { c.Endpoint = "daybreaks.directory" },
				"invalid configuration: daybreak.endpoint not a valid url",
			},
			"EndpointNoHost": {
				func(c *config.DaybreakConfig) { c.Endpoint = "https:///v1" },
				"invalid configuration: daybreak.endpoint not a valid url",
			},
			"NoClientID": {
				func(c *config.DaybreakConfig) { c.ClientID = "" },
				"invalid configuration: daybreak.client_id is required but not set",
			},
			"NoClientSecret": {
				func(c *config.DaybreakConfig) { c.ClientSecret = "" },
				"invalid configuration: daybreak.client_secret is required but not set",
			},
			"NoSyncInterval": {
				func(c *config.DaybreakConfig) { c.SyncInterval = 0 },
				"invalid configuration: daybreak.sync_interval sync interval must be greater than one minute",
			},
			"SyncIntervalAtLimit": {
				func(c *config.DaybreakConfig) { c.SyncInterval = time.Minute },
				"invalid configuration: daybreak.sync_interval sync interval must be greater than one minute",
			},
			"NoSyncJitter": {
				func(c *config.DaybreakConfig) { c.SyncJitter = 0 },
				"invalid configuration: daybreak.sync_jitter sync jitter must be greater than one second",
			},
			"SyncJitterAtLimit": {
				func(c *config.DaybreakConfig) { c.SyncJitter = time.Second },
				"invalid configuration: daybreak.sync_jitter sync jitter must be greater than one second",
			},
		}

		for name, tc := range tests {
			conf := valid()
			tc.mutate(&conf)
			require.EqualError(t, conf.Validate(), tc.err, "expected %s config to be invalid", name)
		}
	})

	// Every problem with the config should be reported, not only the first one.
	t.Run("Multiple", func(t *testing.T) {
		conf := config.DaybreakConfig{Enabled: true}

		errs, ok := confire.ValidationErrors(conf.Validate())
		require.True(t, ok, "expected validation errors to be returned")
		require.Len(t, errs, 5)
		require.EqualError(t, errs[0], "invalid configuration: daybreak.endpoint is required but not set")
		require.EqualError(t, errs[1], "invalid configuration: daybreak.client_id is required but not set")
		require.EqualError(t, errs[2], "invalid configuration: daybreak.client_secret is required but not set")
		require.EqualError(t, errs[3], "invalid configuration: daybreak.sync_interval sync interval must be greater than one minute")
		require.EqualError(t, errs[4], "invalid configuration: daybreak.sync_jitter sync jitter must be greater than one second")
	})
}
