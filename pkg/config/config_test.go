package config_test

import (
	"os"
	"self-hosted-node/pkg/config"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

var testEnv = map[string]string{
	"TRISA_MAINTENANCE": "true",
	"TRISA_MODE":        "test",
	"TRISA_LOG_LEVEL":   "debug",
	"TRISA_CONSOLE_LOG": "true",
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
