package emails_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/emails"
	"go.rtnl.ai/confire"
	"go.rtnl.ai/confire/validate"
)

var testEnv = map[string]string{
	"EMAIL_SENDER":            "Jane Szack <jane@example.com>",
	"EMAIL_SENDER_NAME":       "Jane Szack",
	"EMAIL_SUPPORT_EMAIL":     "support@example.com",
	"EMAIL_COMPLIANCE_EMAIL":  "compliance@example.com",
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
	require.Equal(t, testEnv["EMAIL_SENDER_NAME"], conf.SenderName)
	require.Equal(t, testEnv["EMAIL_SUPPORT_EMAIL"], conf.SupportEmail)
	require.Equal(t, testEnv["EMAIL_COMPLIANCE_EMAIL"], conf.ComplianceEmail)
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
			{
				Sender:  "Compliance Peony <peony@example.com>",
				Testing: false,
				SendGrid: emails.SendGridConfig{
					APIKey: "sg:fakeapikey",
				},
			},
			{
				Sender:       "peony@example.com",
				SupportEmail: "support@example.com",
				Testing:      false,
				SendGrid: emails.SendGridConfig{
					APIKey: "sg:fakeapikey",
				},
			},
			{
				Sender:       "peony@example.com",
				SupportEmail: "Support Peony <support@example.com>",
				Testing:      false,
				SendGrid: emails.SendGridConfig{
					APIKey: "sg:fakeapikey",
				},
			},
			{
				Sender:          "peony@example.com",
				ComplianceEmail: "compliance@example.com",
				Testing:         false,
				SendGrid: emails.SendGridConfig{
					APIKey: "sg:fakeapikey",
				},
			},
			{
				Sender:          "peony@example.com",
				ComplianceEmail: "Compliance Peony <compliance@example.com>",
				Testing:         false,
				SendGrid: emails.SendGridConfig{
					APIKey: "sg:fakeapikey",
				},
			},
		}

		for i, conf := range testCases {
			require.NoError(t, conf.Validate(), "test case %d failed", i)
		}

	})

	// If the config is in testing mode or neither SMTP or SendGrid is configured then
	// the config is not validated.
	t.Run("NotValidated", func(t *testing.T) {
		testCases := []emails.Config{
			{
				Sender:          "foo",
				SupportEmail:    "foo",
				ComplianceEmail: "foo",
				Testing:         false,
			},
			{
				Sender:          "foo",
				SupportEmail:    "foo",
				ComplianceEmail: "foo",
				Testing:         true,
				SMTP:            emails.SMTPConfig{Host: "smtp.example.com"},
				SendGrid:        emails.SendGridConfig{APIKey: "sg:fakeapikey"},
			},
		}

		for i, conf := range testCases {
			require.NoError(t, conf.Validate(), "test case %d failed", i)
		}
	})

	// NOTE: the email config validates its SMTP and SendGrid configs itself (confire is
	// told to ignore them), so their problems are reported after its own.
	t.Run("Invalid", func(t *testing.T) {
		smtp := emails.SMTPConfig{Host: "smtp.example.com", Port: 587, PoolSize: 2}

		testCases := []struct {
			conf emails.Config
			errs []string
		}{
			{
				// A sender that is not set is only reported as required.
				emails.Config{
					Testing: false,
					SMTP:    smtp,
				},
				[]string{"invalid configuration: emails.sender is required but not set"},
			},
			{
				emails.Config{
					Testing:  false,
					SendGrid: emails.SendGridConfig{APIKey: "sg:fakeapikey"},
				},
				[]string{"invalid configuration: emails.sender is required but not set"},
			},
			{
				emails.Config{
					Sender:  "foo",
					Testing: false,
					SMTP:    smtp,
				},
				[]string{"invalid configuration: emails.sender not a valid email address"},
			},
			{
				emails.Config{
					Sender:       "orchid@example.com",
					SupportEmail: "foo",
					Testing:      false,
					SMTP:         smtp,
				},
				[]string{"invalid configuration: emails.support_email not a valid email address"},
			},
			{
				emails.Config{
					Sender:          "orchid@example.com",
					ComplianceEmail: "foo",
					Testing:         false,
					SMTP:            smtp,
				},
				[]string{"invalid configuration: emails.compliance_email not a valid email address"},
			},
			{
				emails.Config{
					Sender:   "orchid@example.com",
					Testing:  false,
					SMTP:     smtp,
					SendGrid: emails.SendGridConfig{APIKey: "sg:fakeapikey"},
				},
				[]string{"invalid configuration: emails.smtp cannot specify both smtp and sendgrid configurations"},
			},
			{
				// The SMTP config is validated as part of the email config.
				emails.Config{
					Sender:  "orchid@example.com",
					Testing: false,
					SMTP:    emails.SMTPConfig{Host: "smtp.example.com", Port: 587, PoolSize: 0},
				},
				[]string{"invalid configuration: smtp.pool_size pool size must be greater than zero"},
			},
			{
				// Every problem with the config should be reported, not only the first.
				emails.Config{
					Sender:          "foo",
					SupportEmail:    "foo",
					ComplianceEmail: "foo",
					Testing:         false,
					SMTP:            emails.SMTPConfig{Host: "smtp.example.com", UseCRAMMD5: true},
					SendGrid:        emails.SendGridConfig{APIKey: "sg:fakeapikey"},
				},
				[]string{
					"invalid configuration: emails.sender not a valid email address",
					"invalid configuration: emails.support_email not a valid email address",
					"invalid configuration: emails.compliance_email not a valid email address",
					"invalid configuration: emails.smtp cannot specify both smtp and sendgrid configurations",
					"invalid configuration: smtp.port is required but not set",
					"invalid configuration: smtp.pool_size pool size must be greater than zero",
					"invalid configuration: smtp.use_cram_md5 cram-md5 authentication requires a username and password",
				},
			},
		}

		for i, tc := range testCases {
			require.Equal(t, tc.errs, validationErrors(tc.conf.Validate()), "test case %d failed", i)
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

	t.Run("Valid", func(t *testing.T) {
		testCases := []emails.SMTPConfig{
			// If no host is specified then SMTP is not enabled and is not validated.
			{},
			{
				Port:       0,
				PoolSize:   0,
				UseCRAMMD5: true,
			},
			{
				Host:     "smtp.example.com",
				Port:     587,
				PoolSize: 4,
			},
			{
				Host:       "smtp.example.com",
				Port:       587,
				Username:   "admin",
				Password:   "supersecret",
				UseCRAMMD5: true,
				PoolSize:   1,
			},
		}

		for i, conf := range testCases {
			require.NoError(t, conf.Validate(), "test case %d failed", i)
		}
	})

	t.Run("Invalid", func(t *testing.T) {
		testCases := []struct {
			conf emails.SMTPConfig
			errs []string
		}{
			{
				emails.SMTPConfig{
					Host:     "smtp.example.com",
					Port:     0,
					PoolSize: 4,
				},
				[]string{"invalid configuration: smtp.port is required but not set"},
			},
			{
				emails.SMTPConfig{
					Host:     "smtp.example.com",
					Port:     527,
					PoolSize: 0,
				},
				[]string{"invalid configuration: smtp.pool_size pool size must be greater than zero"},
			},
			{
				emails.SMTPConfig{
					Host:     "smtp.example.com",
					Port:     527,
					PoolSize: -1,
				},
				[]string{"invalid configuration: smtp.pool_size pool size must be greater than zero"},
			},
			{
				emails.SMTPConfig{
					Host:       "smtp.example.com",
					Port:       527,
					PoolSize:   4,
					UseCRAMMD5: true,
				},
				[]string{"invalid configuration: smtp.use_cram_md5 cram-md5 authentication requires a username and password"},
			},
			{
				emails.SMTPConfig{
					Host:       "smtp.example.com",
					Port:       527,
					Username:   "admin",
					PoolSize:   4,
					UseCRAMMD5: true,
				},
				[]string{"invalid configuration: smtp.use_cram_md5 cram-md5 authentication requires a username and password"},
			},
			{
				emails.SMTPConfig{
					Host:       "smtp.example.com",
					Port:       527,
					Password:   "supersecret",
					PoolSize:   4,
					UseCRAMMD5: true,
				},
				[]string{"invalid configuration: smtp.use_cram_md5 cram-md5 authentication requires a username and password"},
			},
			{
				// Every problem with the config should be reported, not only the first.
				emails.SMTPConfig{
					Host:       "smtp.example.com",
					UseCRAMMD5: true,
				},
				[]string{
					"invalid configuration: smtp.port is required but not set",
					"invalid configuration: smtp.pool_size pool size must be greater than zero",
					"invalid configuration: smtp.use_cram_md5 cram-md5 authentication requires a username and password",
				},
			},
		}

		for i, tc := range testCases {
			require.Equal(t, tc.errs, validationErrors(tc.conf.Validate()), "test case %d failed", i)
		}
	})
}

func TestSendGridConfig(t *testing.T) {
	// If no API key is specified then SendGrid is not enabled; there is nothing else
	// to validate.
	testCases := []emails.SendGridConfig{
		{},
		{APIKey: "sg:fakeapikey"},
	}

	for i, conf := range testCases {
		require.NoError(t, conf.Validate(), "test case %d failed", i)
		require.Equal(t, conf.APIKey != "", conf.Enabled(), "test case %d failed", i)
	}
}

// When the email config is nested in another config (e.g. the Envoy config) then the
// email, SMTP, and SendGrid configs should each be validated when the parent config is
// processed, and each problem should only be reported once: the SMTP and SendGrid
// configs are validated by the email config and must not also be validated by confire.
func TestNestedValidation(t *testing.T) {
	type parent struct {
		Email emails.Config
	}

	t.Run("Valid", func(t *testing.T) {
		conf := &parent{
			Email: emails.Config{
				Sender: "orchid@example.com",
				SMTP:   emails.SMTPConfig{Host: "smtp.example.com", Port: 587, PoolSize: 2},
			},
		}
		require.NoError(t, validate.Validate(conf))
	})

	t.Run("Invalid", func(t *testing.T) {
		conf := &parent{
			Email: emails.Config{
				Sender:   "foo",
				SMTP:     emails.SMTPConfig{Host: "smtp.example.com", Port: 587, PoolSize: 0},
				SendGrid: emails.SendGridConfig{APIKey: "sg:fakeapikey"},
			},
		}

		require.Equal(t, []string{
			"invalid configuration: emails.sender not a valid email address",
			"invalid configuration: emails.smtp cannot specify both smtp and sendgrid configurations",
			"invalid configuration: smtp.pool_size pool size must be greater than zero",
		}, validationErrors(validate.Validate(conf)))
	})

	// In testing mode nothing is validated, including the SMTP config.
	t.Run("Testing", func(t *testing.T) {
		conf := &parent{
			Email: emails.Config{
				Sender:   "foo",
				Testing:  true,
				SMTP:     emails.SMTPConfig{Host: "smtp.example.com", UseCRAMMD5: true},
				SendGrid: emails.SendGridConfig{APIKey: "sg:fakeapikey"},
			},
		}
		require.NoError(t, validate.Validate(conf))
	})

	// The testing flag and the SMTP config are read from the environment under the
	// prefix of the parent, so the same applies to a config that is processed.
	t.Run("Environment", func(t *testing.T) {
		t.Setenv("NESTED_EMAIL_SENDER", "orchid@example.com")
		t.Setenv("NESTED_EMAIL_SMTP_HOST", "smtp.example.com")
		t.Setenv("NESTED_EMAIL_SMTP_POOL_SIZE", "0")

		t.Setenv("NESTED_EMAIL_TESTING", "true")
		require.NoError(t, confire.Process("nested", &parent{}), "expected the smtp config not to be validated in testing mode")

		t.Setenv("NESTED_EMAIL_TESTING", "false")
		require.EqualError(t, confire.Process("nested", &parent{}), "invalid configuration: smtp.pool_size pool size must be greater than zero")
	})
}

func TestGetSenderName(t *testing.T) {
	testCases := []struct {
		conf     emails.Config
		expected string
	}{
		{
			emails.Config{
				Sender: "Jane Szack <jane@example.com>",
			},
			"Jane Szack",
		},
		{
			emails.Config{
				Sender: "jane@example.com",
			},
			"",
		},
		{
			emails.Config{
				Sender:     "",
				SenderName: "Jane Szack",
			},
			"Jane Szack",
		},
		{
			emails.Config{
				Sender:     "",
				SenderName: "",
			},
			"",
		},
		{
			emails.Config{
				Sender:     "foo",
				SenderName: "",
			},
			"",
		},
		{
			emails.Config{
				Sender:     "John Doe <john.doe@example.com>",
				SenderName: "Jane Szack",
			},
			"Jane Szack",
		},
		{
			emails.Config{
				Sender:     "john.doe@example.com",
				SenderName: "John Doe",
			},
			"John Doe",
		},
	}

	for i, tc := range testCases {
		require.Equal(t, tc.expected, tc.conf.GetSenderName(), "test case %d failed", i)
	}
}

// Returns the message of each validation error in the error returned by a validator,
// in the order that they were reported; returns nil if the error is nil.
func validationErrors(err error) (msgs []string) {
	if err == nil {
		return nil
	}

	if errs, ok := confire.ValidationErrors(err); ok {
		for _, verr := range errs {
			msgs = append(msgs, verr.Error())
		}
		return msgs
	}

	return []string{err.Error()}
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
