package config_test

import (
	"io"
	"os"
	"path/filepath"
	"self-hosted-node/pkg/config"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

var testEnv = map[string]string{
	"TRISA_MAINTENANCE":                     "true",
	"TRISA_MODE":                            "test",
	"TRISA_LOG_LEVEL":                       "debug",
	"TRISA_CONSOLE_LOG":                     "true",
	"TRISA_DATABASE_URL":                    "sqlite3:///tmp/trisa.db",
	"TRISA_WEB_ENABLED":                     "true",
	"TRISA_WEB_BIND_ADDR":                   ":4000",
	"TRISA_WEB_ORIGIN":                      "https://example.com",
	"TRISA_WEB_TRISA_ENDPOINT":              "testing.tr-envoy.com:443",
	"TRISA_WEB_TRP_ENDPOINT":                "https://trp.tr-envoy.com/",
	"TRISA_NODE_BIND_ADDR":                  ":556",
	"TRISA_NODE_POOL":                       "fixtures/certs/pool.gz",
	"TRISA_NODE_CERTS":                      "fixtures/certs/certs.gz",
	"TRISA_NODE_KEY_EXCHANGE_CACHE_TTL":     "5m",
	"TRISA_NODE_DIRECTORY_INSECURE":         "true",
	"TRISA_NODE_DIRECTORY_ENDPOINT":         "localhost:2525",
	"TRISA_NODE_DIRECTORY_MEMBERS_ENDPOINT": "localhost:2526",
	"TRISA_DIRECTORY_SYNC_ENABLED":          "true",
	"TRISA_DIRECTORY_SYNC_INTERVAL":         "10m",
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
	require.Equal(t, testEnv["TRISA_MODE"], conf.Mode)
	require.Equal(t, zerolog.DebugLevel, conf.GetLogLevel())
	require.True(t, conf.ConsoleLog)
	require.Equal(t, testEnv["TRISA_DATABASE_URL"], conf.DatabaseURL)
	require.True(t, conf.Web.Maintenance)
	require.True(t, conf.Web.Enabled)
	require.Equal(t, testEnv["TRISA_WEB_BIND_ADDR"], conf.Web.BindAddr)
	require.Equal(t, testEnv["TRISA_WEB_ORIGIN"], conf.Web.Origin)
	require.Equal(t, testEnv["TRISA_WEB_TRISA_ENDPOINT"], conf.Web.TRISAEndpoint)
	require.Equal(t, testEnv["TRISA_WEB_TRP_ENDPOINT"], conf.Web.TRPEndpoint)
	require.True(t, conf.Node.Maintenance)
	require.Equal(t, testEnv["TRISA_NODE_BIND_ADDR"], conf.Node.BindAddr)
	require.Equal(t, testEnv["TRISA_NODE_POOL"], conf.Node.Pool)
	require.Equal(t, testEnv["TRISA_NODE_CERTS"], conf.Node.Certs)
	require.Equal(t, 5*time.Minute, conf.Node.KeyExchangeCacheTTL)
	require.True(t, conf.Node.Directory.Insecure)
	require.Equal(t, testEnv["TRISA_NODE_DIRECTORY_ENDPOINT"], conf.Node.Directory.Endpoint)
	require.Equal(t, testEnv["TRISA_NODE_DIRECTORY_MEMBERS_ENDPOINT"], conf.Node.Directory.MembersEndpoint)
	require.True(t, conf.DirectorySync.Enabled)
	require.Equal(t, 10*time.Minute, conf.DirectorySync.Interval)
}

func TestWebConfig(t *testing.T) {
	t.Run("Disabled", func(t *testing.T) {
		conf := config.WebConfig{Enabled: false}
		require.NoError(t, conf.Validate(), "expected disabled config to be valid")
	})

	t.Run("Valid", func(t *testing.T) {
		conf := config.WebConfig{
			Enabled:  true,
			BindAddr: "127.0.0.1:0",
			Origin:   "http://localhost",
		}
		require.NoError(t, conf.Validate(), "expected valid config to be valid")
	})

	t.Run("Invalid", func(t *testing.T) {
		testCases := []struct {
			conf      config.WebConfig
			errString string
		}{
			{
				conf: config.WebConfig{
					Enabled: true,
					Origin:  "http://localhost",
				},
				errString: "invalid configuration: bindaddr is required",
			},
			{
				conf: config.WebConfig{
					Enabled:  true,
					BindAddr: "127.0.0.1:0",
				},
				errString: "invalid configuration: origin is required",
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
			Endpoint:        "api.trisatest.net:443",
			MembersEndpoint: "members.trisatest.net:443",
		},
	}

	// A valid TRISA config requires paths to certs and a pool
	require.EqualError(t, conf.Validate(), "invalid configuration: specify pool and cert paths")
	conf.Pool = "testdata/trisa.example.dev.pem"
	require.EqualError(t, conf.Validate(), "invalid configuration: specify pool and cert paths")
	conf.Certs = conf.Pool
	require.NoError(t, conf.Validate(), "expected configuration was valid")

	certs, err := conf.LoadCerts()
	require.NoError(t, err, "was unable to load certs")
	require.True(t, certs.IsPrivate(), "certs do not contain private key")

	pool, err := conf.LoadPool()
	require.NoError(t, err, "was unable to load pool")
	require.Len(t, pool, 1, "unexpected cert pool length")
}

func TestTRISAConfigCache(t *testing.T) {
	// Ensure TRISAConfig value caches with pointer receiver
	conf := config.TRISAConfig{
		Maintenance: false,
		BindAddr:    ":9300",
		Directory: config.DirectoryConfig{
			Insecure:        false,
			Endpoint:        "api.trisatest.net:443",
			MembersEndpoint: "members.trisatest.net:443",
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
		{"trisatest.net", "trisatest.net"},
		{"trisatest.net:456", "trisatest.net"},
		{"api.trisatest.net", "trisatest.net"},
		{"api.trisatest.net:443", "trisatest.net"},
		{"api.vaspdirectory.net", "vaspdirectory.net"},
		{"api.vaspdirectory.net:443", "vaspdirectory.net"},
		{"testing.api.vaspdirectory.net", "vaspdirectory.net"},
		{"testing.api.vaspdirectory.net:443", "vaspdirectory.net"},
	}

	for i, tc := range testCases {
		conf := config.DirectoryConfig{Endpoint: tc.endpoint}
		require.Equal(t, tc.expected, conf.Network(), "network name test case %d failed", i)
	}
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
