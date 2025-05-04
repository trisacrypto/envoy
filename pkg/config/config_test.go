package config_test

import (
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/trisacrypto/envoy/pkg/config"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

var testEnv = map[string]string{
	"TRISA_MAINTENANCE":                     "true",
	"TRISA_ORGANIZATION":                    "Testing Organization",
	"TRISA_MODE":                            "test",
	"TRISA_LOG_LEVEL":                       "debug",
	"TRISA_CONSOLE_LOG":                     "true",
	"TRISA_DATABASE_URL":                    "sqlite3:///tmp/trisa.db",
	"TRISA_ENDPOINT":                        "testing.tr-envoy.com:443",
	"TRISA_SEARCH_THRESHOLD":                "0.75",
	"TRISA_WEB_ENABLED":                     "false",
	"TRISA_WEB_API_ENABLED":                 "false",
	"TRISA_WEB_UI_ENABLED":                  "false",
	"TRISA_WEB_BIND_ADDR":                   ":4000",
	"TRISA_WEB_ORIGIN":                      "https://example.com",
	"TRISA_WEB_DOCS_NAME":                   "Test Server",
	"TRISA_WEB_AUTH_KEYS":                   "foo:/path/to/foo.pem,bar:/path/to/bar.pem",
	"TRISA_WEB_AUTH_AUDIENCE":               "https://example.com",
	"TRISA_WEB_AUTH_ISSUER":                 "https://auth.example.com",
	"TRISA_WEB_AUTH_COOKIE_DOMAIN":          "example.com",
	"TRISA_WEB_AUTH_ACCESS_TOKEN_TTL":       "24h",
	"TRISA_WEB_AUTH_REFRESH_TOKEN_TTL":      "48h",
	"TRISA_WEB_AUTH_TOKEN_OVERLAP":          "-12h",
	"TRISA_WEBHOOK_URL":                     "https://example.com/callback",
	"TRISA_WEBHOOK_AUTH_KEY_ID":             "01JT4B3R5Z6AHJXV87QHPPKRBM",
	"TRISA_WEBHOOK_AUTH_KEY_SECRET":         "cfbabc4715b4759d45ba26953dd2fc0bfc2344ef70a2005432e7f16b5081610d",
	"TRISA_WEBHOOK_REQUIRE_SERVER_AUTH":     "true",
	"TRISA_NODE_ENABLED":                    "true",
	"TRISA_NODE_BIND_ADDR":                  ":556",
	"TRISA_NODE_POOL":                       "fixtures/certs/pool.gz",
	"TRISA_NODE_CERTS":                      "fixtures/certs/certs.gz",
	"TRISA_NODE_KEY_EXCHANGE_CACHE_TTL":     "5m",
	"TRISA_NODE_DIRECTORY_INSECURE":         "true",
	"TRISA_NODE_DIRECTORY_ENDPOINT":         "localhost:2525",
	"TRISA_NODE_DIRECTORY_MEMBERS_ENDPOINT": "localhost:2526",
	"TRISA_DIRECTORY_SYNC_ENABLED":          "true",
	"TRISA_DIRECTORY_SYNC_INTERVAL":         "10m",
	"TRISA_TRP_ENABLED":                     "true",
	"TRISA_TRP_BIND_ADDR":                   ":8012",
	"TRISA_TRP_USE_MTLS":                    "false",
	"TRISA_TRP_POOL":                        "fixtures/certs/trp/pool.pem",
	"TRISA_TRP_CERTS":                       "fixtures/certs/trp/certs.pem",
	"TRISA_TRP_IDENTITY_VASP_NAME":          "Testing VASP",
	"TRISA_TRP_IDENTITY_LEI":                "GTFZ00N6IHYMHHNT8S51",
	"TRISA_SUNRISE_ENABLED":                 "false",
	"TRISA_EMAIL_TESTING":                   "true",
	"REGION_INFO_ID":                        "2840302",
	"REGION_INFO_NAME":                      "us-east4c",
	"REGION_INFO_COUNTRY":                   "US",
	"REGION_INFO_CLOUD":                     "GCP",
	"REGION_INFO_CLUSTER":                   "rotational-testing-gke-9",
}

func TestConfig(t *testing.T) {
	// Set required environment variables and cleanup after the test is complete.
	t.Cleanup(cleanupEnv())
	setEnv()

	conf, err := config.New()
	require.NoError(t, err, "could not process configuration from the environment")
	require.False(t, conf.IsZero(), "processed config should not be zero valued")

	// Ensure configuration is correctly set from the environment
	require.True(t, conf.Maintenance)
	require.Equal(t, testEnv["TRISA_ORGANIZATION"], conf.Organization)
	require.Equal(t, testEnv["TRISA_MODE"], conf.Mode)
	require.Equal(t, zerolog.DebugLevel, conf.GetLogLevel())
	require.True(t, conf.ConsoleLog)
	require.Equal(t, testEnv["TRISA_DATABASE_URL"], conf.DatabaseURL)
	require.Equal(t, 0.75, conf.SearchThreshold)
	require.True(t, conf.Web.Maintenance)
	require.False(t, conf.Web.Enabled)
	require.False(t, conf.Web.APIEnabled)
	require.False(t, conf.Web.UIEnabled)
	require.Equal(t, testEnv["TRISA_WEB_BIND_ADDR"], conf.Web.BindAddr)
	require.Equal(t, testEnv["TRISA_WEB_ORIGIN"], conf.Web.Origin)
	require.Equal(t, testEnv["TRISA_ENDPOINT"], conf.Web.TRISAEndpoint)
	require.Equal(t, testEnv["TRISA_WEB_DOCS_NAME"], conf.Web.DocsName)
	require.Len(t, conf.Web.Auth.Keys, 2)
	require.Equal(t, testEnv["TRISA_WEB_AUTH_AUDIENCE"], conf.Web.Auth.Audience)
	require.Equal(t, testEnv["TRISA_WEB_AUTH_ISSUER"], conf.Web.Auth.Issuer)
	require.Equal(t, testEnv["TRISA_WEB_AUTH_COOKIE_DOMAIN"], conf.Web.Auth.CookieDomain)
	require.Equal(t, 24*time.Hour, conf.Web.Auth.AccessTokenTTL)
	require.Equal(t, 48*time.Hour, conf.Web.Auth.RefreshTokenTTL)
	require.Equal(t, -12*time.Hour, conf.Web.Auth.TokenOverlap)
	require.True(t, conf.Webhook.Enabled())
	require.Equal(t, testEnv["TRISA_WEBHOOK_URL"], conf.Webhook.URL)
	require.Equal(t, testEnv["TRISA_WEBHOOK_AUTH_KEY_ID"], conf.Webhook.AuthKeyID)
	require.Equal(t, testEnv["TRISA_WEBHOOK_AUTH_KEY_SECRET"], conf.Webhook.AuthKeySecret)
	require.True(t, conf.Webhook.RequireServerAuth)
	require.True(t, conf.Node.Maintenance)
	require.Equal(t, testEnv["TRISA_ENDPOINT"], conf.Node.Endpoint)
	require.Equal(t, testEnv["TRISA_NODE_BIND_ADDR"], conf.Node.BindAddr)
	require.Equal(t, testEnv["TRISA_NODE_POOL"], conf.Node.Pool)
	require.Equal(t, testEnv["TRISA_NODE_CERTS"], conf.Node.Certs)
	require.Equal(t, 5*time.Minute, conf.Node.KeyExchangeCacheTTL)
	require.True(t, conf.Node.Directory.Insecure)
	require.Equal(t, testEnv["TRISA_NODE_DIRECTORY_ENDPOINT"], conf.Node.Directory.Endpoint)
	require.Equal(t, testEnv["TRISA_NODE_DIRECTORY_MEMBERS_ENDPOINT"], conf.Node.Directory.MembersEndpoint)
	require.True(t, conf.DirectorySync.Enabled)
	require.Equal(t, 10*time.Minute, conf.DirectorySync.Interval)
	require.Equal(t, int32(2840302), conf.RegionInfo.ID)
	require.True(t, conf.TRP.Maintenance)
	require.True(t, conf.TRP.Enabled)
	require.Equal(t, testEnv["TRISA_TRP_BIND_ADDR"], conf.TRP.BindAddr)
	require.False(t, conf.TRP.UseMTLS)
	require.Equal(t, testEnv["TRISA_TRP_POOL"], conf.TRP.Pool)
	require.Equal(t, testEnv["TRISA_TRP_CERTS"], conf.TRP.Certs)
	require.Equal(t, testEnv["TRISA_TRP_IDENTITY_VASP_NAME"], conf.TRP.Identity.VASPName)
	require.Equal(t, testEnv["TRISA_TRP_IDENTITY_LEI"], conf.TRP.Identity.LEI)
	require.False(t, conf.Sunrise.Enabled)
	require.Equal(t, testEnv["REGION_INFO_NAME"], conf.RegionInfo.Name)
	require.Equal(t, testEnv["REGION_INFO_COUNTRY"], conf.RegionInfo.Country)
	require.Equal(t, testEnv["REGION_INFO_CLOUD"], conf.RegionInfo.Cloud)
	require.Equal(t, testEnv["REGION_INFO_CLUSTER"], conf.RegionInfo.Cluster)
}

func TestWebConfig(t *testing.T) {
	t.Run("Disabled", func(t *testing.T) {
		conf := config.WebConfig{Enabled: false}
		require.NoError(t, conf.Validate(), "expected disabled config to be valid")
	})

	t.Run("Valid", func(t *testing.T) {
		testCases := []config.WebConfig{
			{
				Enabled: false,
			},
			{
				Enabled:    true,
				APIEnabled: true,
				UIEnabled:  true,
				BindAddr:   "127.0.0.1:0",
				Origin:     "http://localhost",
			},
			{
				Enabled:    true,
				APIEnabled: false,
				UIEnabled:  true,
				BindAddr:   "127.0.0.1:0",
				Origin:     "http://localhost",
			},
			{
				Enabled:    true,
				APIEnabled: true,
				UIEnabled:  false,
				BindAddr:   "127.0.0.1:0",
				Origin:     "http://localhost",
			},
			{
				Enabled:    false,
				APIEnabled: false,
				UIEnabled:  false,
				BindAddr:   "127.0.0.1:0",
				Origin:     "http://localhost",
			},
		}

		for _, conf := range testCases {
			require.NoError(t, conf.Validate(), "expected valid config to be valid")
		}
	})

	t.Run("Invalid", func(t *testing.T) {
		testCases := []struct {
			conf      config.WebConfig
			errString string
		}{
			{
				conf: config.WebConfig{
					Enabled:    true,
					APIEnabled: true,
					UIEnabled:  true,
					Origin:     "http://localhost",
				},
				errString: "invalid configuration: bindaddr is required",
			},
			{
				conf: config.WebConfig{
					Enabled:    true,
					APIEnabled: true,
					UIEnabled:  true,
					BindAddr:   "127.0.0.1:0",
				},
				errString: "invalid configuration: origin is required",
			},
			{
				conf: config.WebConfig{
					Enabled:    true,
					APIEnabled: false,
					UIEnabled:  false,
					BindAddr:   "127.0.0.1:0",
					Origin:     "http://localhost",
				},
				errString: "invalid configuration: if enabled, either the api, ui, or both need to be enabled",
			},
		}

		for i, tc := range testCases {
			require.EqualError(t, tc.conf.Validate(), tc.errString, "test case %d failed", i)
		}
	})
}

func TestTRISAConfig(t *testing.T) {
	conf := config.TRISAConfig{
		Maintenance: false,
		BindAddr:    ":9300",
		Directory: config.DirectoryConfig{
			Insecure:        false,
			Endpoint:        "api.testnet.directory:443",
			MembersEndpoint: "members.testnet.directory:443",
		},
	}

	// A valid TRISA config requires paths to certs and a pool
	require.EqualError(t, conf.Validate(), "invalid configuration: specify certificates path")
	conf.Pool = "testdata/trisa.example.dev.pem"
	require.EqualError(t, conf.Validate(), "invalid configuration: specify certificates path")
	conf.Certs = conf.Pool
	require.NoError(t, conf.Validate(), "expected configuration was valid")

	certs, err := conf.LoadCerts()
	require.NoError(t, err, "was unable to load certs")
	require.True(t, certs.IsPrivate(), "certs do not contain private key")

	pool, err := conf.LoadPool()
	require.NoError(t, err, "was unable to load pool")
	require.Len(t, pool, 1, "unexpected cert pool length")
}

func TestCertsConfigCache(t *testing.T) {
	// Ensure TRISAConfig value caches with pointer receiver
	conf := config.TRISAConfig{
		Maintenance: false,
		BindAddr:    ":9300",
		Directory: config.DirectoryConfig{
			Insecure:        false,
			Endpoint:        "api.testnet.directory:443",
			MembersEndpoint: "members.testnet.directory:443",
		},
	}

	// Copy the fixture data from testdata into a temporary directory.
	src, err := os.Open("testdata/trisa.example.dev.pem")
	require.NoError(t, err, "could not read testdata certificate fixture")

	// Create a temporary directory and a path to a copy of the certs and pool
	// This directory will be automatically cleaned up at the end of the test.
	path := filepath.Join(t.TempDir(), "copy.example.dev.pem")
	conf.Certs = path
	conf.Pool = path

	createTest := func(conf config.CertsCacheLoader) func(t *testing.T) {
		return func(t *testing.T) {

			// Copy the testdata fixture to the temporary directory fixture
			dst, err := os.Create(path)
			require.NoError(t, err, "could not create temporary certificate fixture")
			_, err = io.Copy(dst, src)
			require.NoError(t, err, "could not copy the testdata fixture to the temporary fixture")
			require.NoError(t, dst.Close(), "could not flush and close temporary certificate fixture")

			// Should be able to load certs from the temporary fixture
			_, err = conf.LoadCerts()
			require.NoError(t, err, "was unable to load certs")
			_, err = conf.LoadPool()
			require.NoError(t, err, "was unable to load pool")

			// Delete the fixture, the certs and pool should be cached
			require.NoError(t, os.Remove(path), "could not delete the temporary fixture")
			require.NoFileExists(t, path, "was unable to delete temporary fixture")

			_, err = conf.LoadCerts()
			require.NoError(t, err, "was unable to load certs")
			_, err = conf.LoadPool()
			require.NoError(t, err, "was unable to load pool")

		}
	}

	t.Run("TRISAConfig", createTest(&conf))
	t.Run("TRPConfig", createTest(&config.TRPConfig{MTLSConfig: config.MTLSConfig{Certs: path, Pool: path}}))

	t.Run("ByReference", func(t *testing.T) {
		var (
			wg sync.WaitGroup
			mu sync.Mutex
		)

		wg.Add(2)

		// Passing by value into a new go routine not should clear the cache
		// NOTE: loading certs is not thread safe, mu protection to get through race check.
		go func(c config.TRISAConfig, wg *sync.WaitGroup) {
			defer wg.Done()
			mu.Lock()
			defer mu.Unlock()

			_, err = conf.LoadCerts()
			require.NoError(t, err, "was unable to load certs")
			_, err = conf.LoadPool()
			require.NoError(t, err, "was unable to load pool")
		}(conf, &wg)

		// Passing by reference into a new go routine should not clear the cache
		// NOTE: loading certs is not thread safe, mu protection to get through race check.
		go func(c *config.TRISAConfig, wg *sync.WaitGroup) {
			defer wg.Done()
			mu.Lock()
			defer mu.Unlock()

			_, err = conf.LoadCerts()
			require.NoError(t, err, "was unable to load certs")
			_, err = conf.LoadPool()
			require.NoError(t, err, "was unable to load pool")
		}(&conf, &wg)

		wg.Wait()

		// If we clear the cache, then loading the certs should error
		conf.Reset()
		_, err = conf.LoadCerts()
		require.Error(t, err, "magic certs still cached or test is broken")
		_, err = conf.LoadPool()
		require.Error(t, err, "magic pool still cached or test is broken")
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
		{"testnet.directory", "testnet.directory"},
		{"testnet.directory:456", "testnet.directory"},
		{"api.testnet.directory", "testnet.directory"},
		{"api.testnet.directory:443", "testnet.directory"},
		{"api.trisa.directory", "trisa.directory"},
		{"api.trisa.directory:443", "trisa.directory"},
		{"testing.api.trisa.directory", "trisa.directory"},
		{"testing.api.trisa.directory:443", "trisa.directory"},
	}

	for i, tc := range testCases {
		conf := config.DirectoryConfig{Endpoint: tc.endpoint}
		require.Equal(t, tc.expected, conf.Network(), "network name test case %d failed", i)
	}
}

func TestWebhookConfig(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		conf := config.WebhookConfig{}
		require.NoError(t, conf.Validate(), "expected empty config to be valid")
		require.False(t, conf.Enabled(), "expected empty config to be disabled")
	})

	t.Run("Endpoint", func(t *testing.T) {
		conf := config.WebhookConfig{
			URL: "",
		}
		require.NoError(t, conf.Validate(), "expected no error when no webhook is specified")
		require.False(t, conf.Enabled(), "expected webhook enabled to be false with no webhook specified")
		require.Nil(t, conf.Endpoint())

		conf.URL = "https://example.com/callback"
		require.NoError(t, conf.Validate(), "expected no error when webhook is specified")
		require.True(t, conf.Enabled(), "expected webhook enabled to be true with webhook specified")
		require.NotNil(t, conf.Endpoint())
		require.Equal(t, conf.URL, conf.Endpoint().String())
	})

	t.Run("Valid", func(t *testing.T) {
		tests := []config.WebhookConfig{
			{
				URL: "",
			},
			{
				URL: "https://example.com/callback",
			},
			{
				URL:               "https://example.com/callback",
				AuthKeyID:         "01JT4B3R5Z6AHJXV87QHPPKRBM",
				AuthKeySecret:     "cfbabc4715b4759d45ba26953dd2fc0bfc2344ef70a2005432e7f16b5081610d",
				RequireServerAuth: false,
			},
			{
				URL:               "https://example.com/callback",
				AuthKeyID:         "01JT4B3R5Z6AHJXV87QHPPKRBM",
				AuthKeySecret:     "cfbabc4715b4759d45ba26953dd2fc0bfc2344ef70a2005432e7f16b5081610d",
				RequireServerAuth: true,
			},
		}

		for i, tc := range tests {
			require.NoError(t, tc.Validate(), "test case %d: expected valid config to be valid", i)
		}
	})

	t.Run("Invalid", func(t *testing.T) {
		tests := []struct {
			conf config.WebhookConfig
			err  string
		}{
			{
				config.WebhookConfig{
					URL:               "https://example.com/callback",
					AuthKeyID:         "",
					AuthKeySecret:     "cfbabc4715b4759d45ba26953dd2fc0bfc2344ef70a2005432e7f16b5081610d",
					RequireServerAuth: false,
				},
				"invalid configuration: webhook auth key id is required when auth key secret is set",
			},
			{
				config.WebhookConfig{
					URL:               "https://example.com/callback",
					AuthKeyID:         "01JT4B3R5Z6AHJXV87QHPPKRBM",
					AuthKeySecret:     "foo",
					RequireServerAuth: false,
				},
				"invalid configuration: could not decode webhook auth key: encoding/hex: invalid byte: U+006F 'o'",
			},
			{
				config.WebhookConfig{
					URL:               "https://example.com/callback",
					AuthKeyID:         "01JT4B3R5Z6AHJXV87QHPPKRBM",
					AuthKeySecret:     "",
					RequireServerAuth: true,
				},
				"invalid configuration: webhook server auth is enabled but no auth key is specified",
			},
		}

		for i, tc := range tests {
			require.Error(t, tc.conf.Validate(), "test case %d: expected valid config to be invalid", i)
			require.EqualError(t, tc.conf.Validate(), tc.err, "test case %d: unexpected error", i)
		}
	})

	t.Run("RequireClientAuth", func(t *testing.T) {
		conf := config.WebhookConfig{
			URL:           "https://example.com/callback",
			AuthKeyID:     "01JT4B3R5Z6AHJXV87QHPPKRBM",
			AuthKeySecret: "cfbabc4715b4759d45ba26953dd2fc0bfc2344ef70a2005432e7f16b5081610d",
		}

		require.True(t, conf.RequireClientAuth(), "expected client auth to be required")

		conf.AuthKeyID = ""
		conf.AuthKeySecret = ""
		require.False(t, conf.RequireClientAuth(), "expected client auth to not be required")
	})

	t.Run("DecodeAuthKey", func(t *testing.T) {
		conf := config.WebhookConfig{
			URL:           "https://example.com/callback",
			AuthKeyID:     "01JT4B3R5Z6AHJXV87QHPPKRBM",
			AuthKeySecret: strings.ToUpper("cfbabc4715b4759d45ba26953dd2fc0bfc2344ef70a2005432e7f16b5081610d"),
		}

		expected, _ := hex.DecodeString("cfbabc4715b4759d45ba26953dd2fc0bfc2344ef70a2005432e7f16b5081610d")
		require.Equal(t, expected, conf.DecodeAuthKey())

		conf.AuthKeyID = ""
		conf.AuthKeySecret = ""
		require.Nil(t, conf.DecodeAuthKey(), "expected nil auth key when not set")
	})
}

// Returns the current environment for the specified keys, or if no keys are specified
// then it returns the current environment for all keys in the testEnv variable.
func curEnv(keys ...string) map[string]string {
	env := make(map[string]string)
	if len(keys) > 0 {
		for _, key := range keys {
			if val, ok := os.LookupEnv(key); ok {
				env[key] = val
			}
		}
	} else {
		for key := range testEnv {
			env[key] = os.Getenv(key)
		}
	}

	return env
}

// Sets the environment variables from the testEnv variable. If no keys are specified,
// then this function sets all environment variables from the testEnv.
func setEnv(keys ...string) {
	if len(keys) > 0 {
		for _, key := range keys {
			if val, ok := testEnv[key]; ok {
				os.Setenv(key, val)
			}
		}
	} else {
		for key, val := range testEnv {
			os.Setenv(key, val)
		}
	}
}

// Cleanup helper function that can be run when the tests are complete to reset the
// environment back to its previous state before the test was run.
func cleanupEnv(keys ...string) func() {
	prevEnv := curEnv(keys...)
	return func() {
		for key, val := range prevEnv {
			if val != "" {
				os.Setenv(key, val)
			} else {
				os.Unsetenv(key)
			}
		}
	}
}
