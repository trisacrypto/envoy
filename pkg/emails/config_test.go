package emails_test

import (
	"os"
	"testing"

	"github.com/rotationalio/confire"
	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/emails"
)

var testEnv = map[string]string{
	"EMAIL_SENDER":            "Jane Szack <jane@example.com>",
	"EMAIL_TESTING":           "true",
	"EMAIL_SMTP_HOST":         "smtp.example.com",
	"EMAIL_SMTP_PORT":         "25",
	"EMAIL_SMTP_USERNAME":     "jszack",
	"EMAIL_SMTP_PASSWORD":     "supersecret",
	"EMAIL_SMTP_USE_CRAM_MD5": "true",
	"EMAIL_SMTP_POOL_SIZE":    "16",
	"EMAIL_SENDGRID_API_KEY":  "sg:fakeapikey",
}

func TestConfig(t *testing.T) {
	// Set required environment variables and cleanup after the test is complete.
	t.Cleanup(cleanupEnv())
	setEnv()

	// NOTE: no validation is run while creating the config from the environment
	conf, err := config()
	require.Equal(t, testEnv["EMAIL_SENDER"], conf.Sender)
	require.True(t, conf.Testing)
	require.Equal(t, testEnv["EMAIL_SMTP_HOST"], conf.SMTP.Host)
	require.Equal(t, uint16(25), conf.SMTP.Port)
	require.Equal(t, testEnv["EMAIL_SMTP_USERNAME"], conf.SMTP.Username)
	require.Equal(t, testEnv["EMAIL_SMTP_PASSWORD"], conf.SMTP.Password)
	require.True(t, conf.SMTP.UseCRAMMD5)
	require.Equal(t, 16, conf.SMTP.PoolSize)
	require.Equal(t, testEnv["EMAIL_SENDGRID_API_KEY"], conf.SendGrid.APIKey)
	require.NoError(t, err, "could not process configuration from the environment")
}

func TestConfigAvailable(t *testing.T) {
	testCases := []struct {
		conf   emails.Config
		assert require.BoolAssertionFunc
	}{
		{
			emails.Config{},
			require.False,
		},
		{
			emails.Config{
				SMTP: emails.SMTPConfig{Host: "email.example.com"},
			},
			require.True,
		},
		{
			emails.Config{
				SendGrid: emails.SendGridConfig{APIKey: "sg:fakeapikey"},
			},
			require.True,
		},
	}

	for i, tc := range testCases {
		tc.assert(t, tc.conf.Available(), "test case %d failed", i)
	}
}

func TestConfigValidation(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		testCases := []emails.Config{
			{
				Testing: false,
			},
			{
				Sender:  "peony@example.com",
				Testing: false,
				SendGrid: emails.SendGridConfig{
					APIKey: "sg:fakeapikey",
				},
			},
			{
				Sender:  "peony@example.com",
				Testing: false,
				SMTP: emails.SMTPConfig{
					Host:       "smtp.example.com",
					Port:       587,
					Username:   "admin",
					Password:   "supersecret",
					UseCRAMMD5: false,
					PoolSize:   4,
				},
			},
			{
				Testing: true,
			},
		}

		for i, conf := range testCases {
			require.NoError(t, conf.Validate(), "test case %d failed", i)
		}

	})

	t.Run("Invalid", func(t *testing.T) {
		testCases := []struct {
			conf emails.Config
			err  error
		}{
			{
				emails.Config{
					Testing: false,
					SMTP:    emails.SMTPConfig{Host: "email.example.com"},
				},
				emails.ErrConfigMissingSender,
			},
			{
				emails.Config{
					Testing:  false,
					SendGrid: emails.SendGridConfig{APIKey: "sg:fakeapikey"},
				},
				emails.ErrConfigMissingSender,
			},
			{
				emails.Config{
					Sender:  "foo",
					Testing: false,
					SMTP:    emails.SMTPConfig{Host: "smtp.example.com"},
				},
				emails.ErrConfigInvalidSender,
			},
			{
				emails.Config{
					Sender:  "orchid@example.com",
					Testing: false,
					SMTP: emails.SMTPConfig{
						Host: "smtp.example.com",
					},
					SendGrid: emails.SendGridConfig{
						APIKey: "sg:fakeapikey",
					},
				},
				emails.ErrConfigConflict,
			},
			{
				emails.Config{
					Sender:  "orchid@example.com",
					Testing: false,
					SMTP: emails.SMTPConfig{
						Host: "smtp.example.com",
						Port: 0,
					},
				},
				emails.ErrConfigMissingPort,
			},
			{
				emails.Config{
					Sender:  "orchid@example.com",
					Testing: false,
					SMTP: emails.SMTPConfig{
						Host: "smtp.example.com",
						Port: 527,
					},
				},
				emails.ErrConfigPoolSize,
			},
			{
				emails.Config{
					Sender:  "orchid@example.com",
					Testing: false,
					SMTP: emails.SMTPConfig{
						Host:       "smtp.example.com",
						Port:       527,
						PoolSize:   4,
						UseCRAMMD5: true,
					},
				},
				emails.ErrConfigCRAMMD5Auth,
			},
		}

		for i, tc := range testCases {
			require.ErrorIs(t, tc.conf.Validate(), tc.err, "test case %d failed", i)
		}
	})
}

func TestSMTPConfig(t *testing.T) {
	t.Run("Addr", func(t *testing.T) {
		conf := emails.SMTPConfig{
			Host: "smtp.example.com",
			Port: 527,
		}
		require.Equal(t, "smtp.example.com:527", conf.Addr())
	})
}

// Creates a new email config from the current environment.
func config() (conf emails.Config, err error) {
	if err = confire.Process("email", &conf); err != nil {
		return conf, err
	}
	return conf, nil
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
