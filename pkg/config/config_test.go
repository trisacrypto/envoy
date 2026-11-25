package config_test

import (
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/trisacrypto/envoy/pkg/config"
	"github.com/trisacrypto/envoy/pkg/emails"
	"github.com/trisacrypto/envoy/pkg/logger"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"go.rtnl.ai/confire"
	"go.rtnl.ai/confire/contest"
	"go.rtnl.ai/confire/env"
	"go.rtnl.ai/confire/validate"
)

// cSpell: ignore webp GTFZ00N6IHYMHHNT8S51

// The values in the test environment variable should result a configuration that is
// equal to the validConfig fixture when processed. Anytime a configuration value is
// added to this package, it should be added to this map and have a non-default value
// for testing and validation purposes; TestEnvironmentSpec enforces both rules.
//
// NOTE: both SMTP and SendGrid are configured so that every variable is exercised;
// this is only valid because TRISA_EMAIL_TESTING skips email validation.
var testEnv = contest.Env{
	"TRISA_MAINTENANCE":                      "true",
	"TRISA_ORGANIZATION":                     "Testing Organization",
	"TRISA_MODE":                             "test",
	"TRISA_LOG_LEVEL":                        "debug",
	"TRISA_CONSOLE_LOG":                      "true",
	"TRISA_DATABASE_URL":                     "sqlite3:///tmp/trisa.db",
	"TRISA_ENDPOINT":                         "testing.tr-envoy.com:443",
	"TRISA_SEARCH_THRESHOLD":                 "0.75",
	"TRISA_BOOT_REGENERATE_TRAVEL_ADDRESSES": "true",
	"TRISA_WEB_ENABLED":                      "false",
	"TRISA_WEB_API_ENABLED":                  "false",
	"TRISA_WEB_UI_ENABLED":                   "false",
	"TRISA_WEB_BIND_ADDR":                    ":4000",
	"TRISA_WEB_ORIGIN":                       "https://envoy.example.com",
	"TRISA_WEB_DOCS_NAME":                    "Test Server",
	"TRISA_WEB_LOGO_URI":                     "/static/img/blockpass-logo.webp",
	"TRISA_WEB_AUTH_KEYS":                    "foo:/path/to/foo.pem,bar:/path/to/bar.pem",
	"TRISA_WEB_AUTH_AUDIENCE":                "https://example.com",
	"TRISA_WEB_AUTH_ISSUER":                  "https://auth.example.com",
	"TRISA_WEB_AUTH_COOKIE_DOMAIN":           "example.com",
	"TRISA_WEB_AUTH_ACCESS_TOKEN_TTL":        "24h",
	"TRISA_WEB_AUTH_REFRESH_TOKEN_TTL":       "48h",
	"TRISA_WEB_AUTH_TOKEN_OVERLAP":           "-12h",
	"TRISA_WEBHOOK_URL":                      "https://example.com/callback",
	"TRISA_WEBHOOK_USE_MTLS":                 "true",
	"TRISA_WEBHOOK_CERTS":                    "fixtures/certs/webhook/certs.pem",
	"TRISA_WEBHOOK_POOL":                     "fixtures/certs/webhook/pool.pem",
	"TRISA_WEBHOOK_AUTH_KEY_ID":              "01JT4B3R5Z6AHJXV87QHPPKRBM",
	"TRISA_WEBHOOK_AUTH_KEY_SECRET":          "cfbabc4715b4759d45ba26953dd2fc0bfc2344ef70a2005432e7f16b5081610d",
	"TRISA_WEBHOOK_REQUIRE_SERVER_AUTH":      "true",
	"TRISA_NODE_ENABLED":                     "false",
	"TRISA_NODE_BIND_ADDR":                   ":556",
	"TRISA_NODE_POOL":                        "fixtures/certs/pool.gz",
	"TRISA_NODE_CERTS":                       "fixtures/certs/certs.gz",
	"TRISA_NODE_KEY_EXCHANGE_CACHE_TTL":      "5m",
	"TRISA_NODE_DIRECTORY_INSECURE":          "true",
	"TRISA_NODE_DIRECTORY_ENDPOINT":          "localhost:2525",
	"TRISA_NODE_DIRECTORY_MEMBERS_ENDPOINT":  "localhost:2526",
	"TRISA_DIRECTORY_SYNC_ENABLED":           "false",
	"TRISA_DIRECTORY_SYNC_INTERVAL":          "10m",
	"TRISA_TRP_ENDPOINT":                     "https://trp.example.com",
	"TRISA_TRP_ENABLED":                      "false",
	"TRISA_TRP_BIND_ADDR":                    ":8012",
	"TRISA_TRP_USE_MTLS":                     "false",
	"TRISA_TRP_POOL":                         "fixtures/certs/trp/pool.pem",
	"TRISA_TRP_CERTS":                        "fixtures/certs/trp/certs.pem",
	"TRISA_TRP_IDENTITY_VASP_NAME":           "Testing VASP",
	"TRISA_TRP_IDENTITY_LEI":                 "GTFZ00N6IHYMHHNT8S51",
	"TRISA_SUNRISE_ENABLED":                  "false",
	"TRISA_SUNRISE_INVITE_ENDPOINT":          "/sunrise/review",
	"TRISA_SUNRISE_REQUIRE_OTP":              "false",
	"TRISA_DAYBREAK_ENABLED":                 "true",
	"TRISA_DAYBREAK_ENDPOINT":                "https://example.directory",
	"TRISA_DAYBREAK_CLIENT_ID":               "1234567890",
	"TRISA_DAYBREAK_CLIENT_SECRET":           "0987654321",
	"TRISA_DAYBREAK_SYNC_INTERVAL":           "15m",
	"TRISA_DAYBREAK_SYNC_JITTER":             "30s",
	"TRISA_EMAIL_SENDER":                     "Envoy Testing <envoy@example.com>",
	"TRISA_EMAIL_SENDER_NAME":                "Envoy Testing",
	"TRISA_EMAIL_SUPPORT_EMAIL":              "support@example.com",
	"TRISA_EMAIL_COMPLIANCE_EMAIL":           "compliance@example.com",
	"TRISA_EMAIL_TESTING":                    "true",
	"TRISA_EMAIL_SMTP_HOST":                  "smtp.example.com",
	"TRISA_EMAIL_SMTP_PORT":                  "25",
	"TRISA_EMAIL_SMTP_USERNAME":              "envoy",
	"TRISA_EMAIL_SMTP_PASSWORD":              "supersecret",
	"TRISA_EMAIL_SMTP_USE_CRAM_MD5":          "true",
	"TRISA_EMAIL_SMTP_POOL_SIZE":             "16",
	"TRISA_EMAIL_SENDGRID_API_KEY":           "sg:fakeapikey",
	"REGION_INFO_ID":                         "2840302",
	"REGION_INFO_NAME":                       "us-east4c",
	"REGION_INFO_COUNTRY":                    "US",
	"REGION_INFO_CLOUD":                      "GCP",
	"REGION_INFO_CLUSTER":                    "rotational-testing-gke-9",
	"REGION_INFO_ZONE":                       "us-east4-c",
}

// This config should always pass validation and should match the testEnv.
var validConfig = config.Config{
	Maintenance:     true,
	Organization:    "Testing Organization",
	Mode:            "test",
	LogLevel:        logger.LevelDecoder(zerolog.DebugLevel),
	ConsoleLog:      true,
	DatabaseURL:     "sqlite3:///tmp/trisa.db",
	SearchThreshold: 0.75,
	Boot: config.BootConfig{
		RegenerateTravelAddresses: true,
	},
	Web: config.WebConfig{
		Maintenance:   true,
		Enabled:       false,
		APIEnabled:    false,
		UIEnabled:     false,
		BindAddr:      ":4000",
		Origin:        "https://envoy.example.com",
		TRISAEndpoint: "testing.tr-envoy.com:443",
		TRPEndpoint:   "https://trp.example.com",
		DocsName:      "Test Server",
		LogoURI:       "/static/img/blockpass-logo.webp",
		Auth: config.AuthConfig{
			Keys: map[string]string{
				"foo": "/path/to/foo.pem",
				"bar": "/path/to/bar.pem",
			},
			Audience:        "https://example.com",
			Issuer:          "https://auth.example.com",
			CookieDomain:    "example.com",
			AccessTokenTTL:  24 * time.Hour,
			RefreshTokenTTL: 48 * time.Hour,
			TokenOverlap:    -12 * time.Hour,
		},
	},
	Webhook: config.WebhookConfig{
		MTLSConfig: config.MTLSConfig{
			Pool:  "fixtures/certs/webhook/pool.pem",
			Certs: "fixtures/certs/webhook/certs.pem",
		},
		URL:               "https://example.com/callback",
		UseMTLS:           true,
		AuthKeyID:         "01JT4B3R5Z6AHJXV87QHPPKRBM",
		AuthKeySecret:     "cfbabc4715b4759d45ba26953dd2fc0bfc2344ef70a2005432e7f16b5081610d",
		RequireServerAuth: true,
	},
	Node: config.TRISAConfig{
		MTLSConfig: config.MTLSConfig{
			Pool:  "fixtures/certs/pool.gz",
			Certs: "fixtures/certs/certs.gz",
		},
		Maintenance:         true,
		Endpoint:            "testing.tr-envoy.com:443",
		Enabled:             false,
		BindAddr:            ":556",
		KeyExchangeCacheTTL: 5 * time.Minute,
		Directory: config.DirectoryConfig{
			Insecure:        true,
			Endpoint:        "localhost:2525",
			MembersEndpoint: "localhost:2526",
		},
	},
	DirectorySync: config.DirectorySyncConfig{
		Enabled:  false,
		Interval: 10 * time.Minute,
	},
	TRP: config.TRPConfig{
		MTLSConfig: config.MTLSConfig{
			Pool:  "fixtures/certs/trp/pool.pem",
			Certs: "fixtures/certs/trp/certs.pem",
		},
		Maintenance: true,
		Endpoint:    "https://trp.example.com",
		Enabled:     false,
		BindAddr:    ":8012",
		UseMTLS:     false,
		Identity: config.TRPIdentityConfig{
			VASPName: "Testing VASP",
			LEI:      "GTFZ00N6IHYMHHNT8S51",
		},
	},
	Sunrise: config.SunriseConfig{
		Enabled:        false,
		BaseURL:        "https://envoy.example.com", // inherited from TRISA_WEB_ORIGIN
		InviteEndpoint: "/sunrise/review",
		RequireOTP:     false,
	},
	Daybreak: config.DaybreakConfig{
		Enabled:      true,
		Endpoint:     "https://example.directory",
		ClientID:     "1234567890",
		ClientSecret: "0987654321",
		SyncInterval: 15 * time.Minute,
		SyncJitter:   30 * time.Second,
	},
	Email: emails.Config{
		Sender:          "Envoy Testing <envoy@example.com>",
		SenderName:      "Envoy Testing",
		SupportEmail:    "support@example.com",
		ComplianceEmail: "compliance@example.com",
		Testing:         true,
		SMTP: emails.SMTPConfig{
			Host:       "smtp.example.com",
			Port:       25,
			Username:   "envoy",
			Password:   "supersecret",
			UseCRAMMD5: true,
			PoolSize:   16,
		},
		SendGrid: emails.SendGridConfig{
			APIKey: "sg:fakeapikey",
		},
	},
	RegionInfo: config.RegionInfo{
		ID:      2840302,
		Name:    "us-east4c",
		Country: "US",
		Cloud:   "GCP",
		Cluster: "rotational-testing-gke-9",
		Zone:    "us-east4-c",
	},
}

// The smallest environment that produces a valid configuration: everything else has
// a usable default.
var minimalEnv = contest.Env{
	"TRISA_NODE_CERTS":   "fixtures/certs/certs.gz",
	"TRISA_TRP_ENDPOINT": "https://trp.example.com",
}

// The configuration that results from the minimalEnv, e.g. the defaults.
var defaultConfig = config.Config{
	Maintenance:     false,
	Organization:    "Envoy",
	Mode:            "release",
	LogLevel:        logger.LevelDecoder(zerolog.InfoLevel),
	ConsoleLog:      false,
	DatabaseURL:     "sqlite3:///trisa.db",
	SearchThreshold: 0.0,
	Web: config.WebConfig{
		Enabled:     true,
		APIEnabled:  true,
		UIEnabled:   true,
		BindAddr:    ":8000",
		Origin:      "http://localhost:8000",
		TRPEndpoint: "https://trp.example.com",
		DocsName:    "Envoy API Reference",
		Auth: config.AuthConfig{
			Audience:        "http://localhost:8000",
			Issuer:          "http://localhost:8000",
			CookieDomain:    "localhost",
			AccessTokenTTL:  1 * time.Hour,
			RefreshTokenTTL: 2 * time.Hour,
			TokenOverlap:    -15 * time.Minute,
		},
	},
	Webhook: config.WebhookConfig{
		// Webhook certs fall back to the TRISA node certs
		MTLSConfig: config.MTLSConfig{Certs: "fixtures/certs/certs.gz"},
	},
	Node: config.TRISAConfig{
		MTLSConfig:          config.MTLSConfig{Certs: "fixtures/certs/certs.gz"},
		Enabled:             true,
		BindAddr:            ":8100",
		KeyExchangeCacheTTL: 24 * time.Hour,
		Directory: config.DirectoryConfig{
			Endpoint:        "api.trisa.directory:443",
			MembersEndpoint: "members.trisa.directory:443",
		},
	},
	DirectorySync: config.DirectorySyncConfig{
		Enabled:  true,
		Interval: 6 * time.Hour,
	},
	TRP: config.TRPConfig{
		// TRP certs fall back to the TRISA node certs
		MTLSConfig: config.MTLSConfig{Certs: "fixtures/certs/certs.gz"},
		Endpoint:   "https://trp.example.com",
		Enabled:    true,
		BindAddr:   ":8200",
		UseMTLS:    true,
	},
	Sunrise: config.SunriseConfig{
		Enabled:        true,
		BaseURL:        "http://localhost:8000", // same default as TRISA_WEB_ORIGIN
		InviteEndpoint: "/sunrise/verify",
		RequireOTP:     true,
	},
	Daybreak: config.DaybreakConfig{
		Enabled:      false,
		Endpoint:     "https://daybreaks.directory",
		SyncInterval: 5 * time.Minute,
		SyncJitter:   1 * time.Minute,
	},
	Email: emails.Config{
		SMTP: emails.SMTPConfig{
			Port:     587,
			PoolSize: 2,
		},
	},
}

// Sets the testEnv for the duration of the test, overriding any variables in set and
// removing any variables in clear from the environment.
func setTestEnv(t *testing.T, set contest.Env, clear ...string) {
	t.Helper()
	t.Cleanup(testEnv.Set())
	t.Cleanup(set.Set())

	// NOTE: Clear with no keys clears the entire testEnv, which is not intended here.
	if len(clear) > 0 {
		t.Cleanup(testEnv.Clear(clear...))
	}
}

func TestConfig(t *testing.T) {
	t.Run("Environment", func(t *testing.T) {
		t.Cleanup(testEnv.Set())

		conf, err := config.New()
		require.NoError(t, err, "could not process config from environment")
		require.Equal(t, validConfig, *conf, "expected the config to match the test environment")
	})

	t.Run("Defaults", func(t *testing.T) {
		t.Cleanup(testEnv.Clear())
		t.Cleanup(minimalEnv.Set())

		conf, err := config.New()
		require.NoError(t, err, "expected the minimal environment to be a valid config")
		require.Equal(t, defaultConfig, *conf, "expected the config to match the defaults")
	})

	t.Run("Invalid", func(t *testing.T) {
		tests := []struct {
			name  string
			set   contest.Env // variables to override in the testEnv
			clear []string    // variables to remove from the testEnv
			err   string      // expected to be contained in the error
		}{
			{
				name: "Mode",
				set:  contest.Env{"TRISA_MODE": "testing"},
				err:  `"testing" is not a valid gin mode`,
			},
			{
				name:  "NodeCerts",
				clear: []string{"TRISA_NODE_CERTS"},
				err:   "trisa.certs is required but not set",
			},
			{
				name: "RequiredDirectoryEndpoint",
				set:  contest.Env{"TRISA_NODE_DIRECTORY_ENDPOINT": ""},
				err:  "Endpoint is required but not set",
			},
			{
				name: "RequiredMembersEndpoint",
				set:  contest.Env{"TRISA_NODE_DIRECTORY_MEMBERS_ENDPOINT": ""},
				err:  "MembersEndpoint is required but not set",
			},
			{
				name: "Web",
				set:  contest.Env{"TRISA_WEB_ENABLED": "true", "TRISA_WEB_UI_ENABLED": "true", "TRISA_WEB_BIND_ADDR": ""},
				err:  "web.bindaddr is required but not set",
			},
			{
				name: "WebOrigin",
				set:  contest.Env{"TRISA_WEB_ENABLED": "true", "TRISA_WEB_UI_ENABLED": "true", "TRISA_WEB_ORIGIN": "http://[::1"},
				err:  `web.origin "http://[::1" is not a valid url`,
			},
			{
				name: "Webhook",
				set:  contest.Env{"TRISA_WEBHOOK_AUTH_KEY_SECRET": "not hex"},
				err:  "webhook.authkeysecret could not hex decode auth secret",
			},
			{
				name:  "TRP",
				set:   contest.Env{"TRISA_TRP_ENABLED": "true"},
				clear: []string{"TRISA_TRP_ENDPOINT"},
				err:   "trp.endpoint is required but not set",
			},
			{
				name:  "Daybreak",
				clear: []string{"TRISA_DAYBREAK_CLIENT_ID"},
				err:   "daybreak.client_id is required but not set",
			},
			{
				name: "Email",
				set:  contest.Env{"TRISA_EMAIL_TESTING": "false"},
				err:  "emails.smtp cannot specify both smtp and sendgrid configurations",
			},
			{
				name:  "EmailSMTP",
				set:   contest.Env{"TRISA_EMAIL_TESTING": "false"},
				clear: []string{"TRISA_EMAIL_SENDGRID_API_KEY", "TRISA_EMAIL_SMTP_USERNAME"},
				err:   "smtp.use_cram_md5 cram-md5 authentication requires a username and password",
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				setTestEnv(t, tc.set, tc.clear...)

				conf, err := config.New()
				require.ErrorContains(t, err, tc.err)
				require.Nil(t, conf, "expected no config to be returned when invalid")

				// Each of these environments has a single problem, which should be
				// reported once. A problem is reported more than once if a nested
				// config is validated more than once, e.g. because its Validate method
				// has a value receiver or because it is also called by its parent.
				count := strings.Count(err.Error(), tc.err)
				require.Equal(t, 1, count, "expected the error to be reported once but it was reported %d times:\n%s", count, err)
			})
		}
	})

	t.Run("Unparseable", func(t *testing.T) {
		tests := []contest.Env{
			{"TRISA_MAINTENANCE": "maybe"},
			{"TRISA_LOG_LEVEL": "loud"},
			{"TRISA_SEARCH_THRESHOLD": "high"},
			{"TRISA_WEB_AUTH_ACCESS_TOKEN_TTL": "soon"},
			{"TRISA_EMAIL_SMTP_PORT": "-1"},
			{"REGION_INFO_ID": "us-east4c"},
		}

		t.Cleanup(testEnv.Set())
		for _, tc := range tests {
			restore := tc.Set()

			conf, err := config.New()
			require.True(t, confire.IsParseError(err), "expected a parse error for %v, got %v", tc, err)
			require.Nil(t, conf, "expected no config to be returned when unparseable")
			restore()
		}
	})

	t.Run("LogLevel", func(t *testing.T) {
		require.Equal(t, zerolog.DebugLevel, validConfig.GetLogLevel())
		require.Equal(t, zerolog.InfoLevel, defaultConfig.GetLogLevel())
	})

	t.Run("DatabaseDSN", func(t *testing.T) {
		uri := validConfig.DatabaseDSN()
		require.NotNil(t, uri, "expected the database url to be parsed")
		require.Equal(t, "sqlite3", uri.Scheme)
		require.Equal(t, "tmp/trisa.db", uri.Path)
	})
}

// The environment specification is defined by the struct tags of the Config; these
// tests ensure the fixtures in this file are kept in sync with that specification so
// that a new configuration value cannot be added without also being tested.
func TestEnvironmentSpec(t *testing.T) {
	// A field with an env tag is read from either its prefixed name (the key) or from
	// the name in the tag (the alt), so either one may be used in the testEnv.
	envvar := func(info env.Info) string {
		if _, ok := testEnv[info.Key]; ok || info.Alt == "" {
			return info.Key
		}
		if _, ok := testEnv[info.Alt]; ok {
			return info.Alt
		}
		return info.Alt + " (or " + info.Key + ")"
	}

	t.Run("Complete", func(t *testing.T) {
		spec, err := env.Gather(config.Prefix, &config.Config{})
		require.NoError(t, err, "could not gather environment specification")

		var missing []string
		known := make(map[string]struct{}, len(spec))

		for _, info := range spec {
			known[info.Key] = struct{}{}
			known[info.Alt] = struct{}{}

			if _, ok := testEnv[envvar(info)]; !ok {
				missing = append(missing, envvar(info))
			}
		}
		require.Empty(t, missing, "variables in the config spec must be added to testEnv and validConfig")

		var unknown []string
		for key := range testEnv {
			if _, ok := known[key]; !ok {
				unknown = append(unknown, key)
			}
		}
		slices.Sort(unknown)
		require.Empty(t, unknown, "variables in testEnv are not read by the config and should be removed")
	})

	t.Run("NonDefault", func(t *testing.T) {
		t.Cleanup(testEnv.Set())

		defaults := &config.Config{}
		require.NoError(t, confire.Process(config.Prefix, defaults, confire.NoEnv, confire.NoValidate))

		conf, err := config.New()
		require.NoError(t, err, "could not process config from environment")

		expected, err := env.Gather(config.Prefix, defaults)
		require.NoError(t, err, "could not gather default fields")

		actual, err := env.Gather(config.Prefix, conf)
		require.NoError(t, err, "could not gather config fields")
		require.Len(t, actual, len(expected))

		var same []string
		for i, info := range actual {
			if reflect.DeepEqual(expected[i].Field.Value(), info.Field.Value()) {
				same = append(same, envvar(info))
			}
		}
		require.Empty(t, same, "variables in testEnv must be set to a non-default value")
	})
}

// Config.Validate only checks the top level values of the config; the nested configs
// are validated when the whole config is validated, which is how config.New and
// config.Set validate a config, so that is how these tests validate it as well.
func TestConfigValidation(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		valid, defaults := validConfig, defaultConfig
		require.NoError(t, validate.Validate(&valid), "expected valid config to be valid")
		require.NoError(t, validate.Validate(&defaults), "expected default config to be valid")
	})

	t.Run("Mode", func(t *testing.T) {
		for _, mode := range []string{"release", "debug", "test"} {
			conf := config.Config{Mode: mode}
			require.NoError(t, conf.Validate(), "expected %q to be a valid mode", mode)
		}

		for _, mode := range []string{"", "foo", "testing", "RELEASE"} {
			conf := config.Config{Mode: mode}
			require.EqualError(t, conf.Validate(), `invalid configuration: mode "`+mode+`" is not a valid gin mode`)
		}
	})

	// Ensures that each nested config is validated as part of the whole config.
	t.Run("Invalid", func(t *testing.T) {
		tests := []struct {
			name   string
			mutate func(*config.Config)
			err    string // expected to be contained in the error
		}{
			{
				"Mode",
				func(c *config.Config) { c.Mode = "foo" },
				`mode "foo" is not a valid gin mode`,
			},
			{
				"Web",
				func(c *config.Config) { c.Web.Enabled = true },
				"web.enabled if enabled, either the api, ui, or both need to be enabled",
			},
			{
				"Webhook",
				func(c *config.Config) { c.Webhook.AuthKeyID = "" },
				"webhook.authkeyid auth key id is required when auth key secret is set",
			},
			{
				"Node",
				func(c *config.Config) { c.Node.Certs = "" },
				"trisa.certs is required but not set",
			},
			{
				"Directory",
				func(c *config.Config) { c.Node.Directory.Endpoint = "" },
				"Endpoint is required but not set",
			},
			{
				"TRP",
				func(c *config.Config) { c.TRP.Enabled = true; c.TRP.Endpoint = "" },
				"trp.endpoint is required but not set",
			},
			{
				"Daybreak",
				func(c *config.Config) { c.Daybreak.ClientSecret = "" },
				"daybreak.client_secret is required but not set",
			},
			{
				"Email",
				func(c *config.Config) { c.Email.Testing = false },
				"emails.smtp cannot specify both smtp and sendgrid configurations",
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				conf := validConfig
				tc.mutate(&conf)
				require.ErrorContains(t, validate.Validate(&conf), tc.err)
			})
		}
	})

	// A config that is not loaded from the environment does not have any defaults.
	t.Run("Empty", func(t *testing.T) {
		conf := config.Config{Mode: "test"}

		err := validate.Validate(&conf)
		require.ErrorContains(t, err, "trisa.certs is required but not set")
		require.ErrorContains(t, err, "Endpoint is required but not set")
		require.ErrorContains(t, err, "MembersEndpoint is required but not set")
	})
}

// If TRP or Webhook certs are not specified they should fallback to the TRISA certs.
func TestCertsFallback(t *testing.T) {
	var (
		trp     = []string{"TRISA_TRP_CERTS", "TRISA_TRP_POOL"}
		webhook = []string{"TRISA_WEBHOOK_CERTS", "TRISA_WEBHOOK_POOL"}
	)

	tests := []struct {
		name    string
		clear   []string
		trp     config.MTLSConfig
		webhook config.MTLSConfig
	}{
		{
			name:    "Specified",
			trp:     validConfig.TRP.MTLSConfig,
			webhook: validConfig.Webhook.MTLSConfig,
		},
		{
			name:    "Unspecified",
			clear:   append(trp, webhook...),
			trp:     validConfig.Node.MTLSConfig,
			webhook: validConfig.Node.MTLSConfig,
		},
		{
			name:    "TRPOnly",
			clear:   trp,
			trp:     validConfig.Node.MTLSConfig,
			webhook: validConfig.Webhook.MTLSConfig,
		},
		{
			name:    "WebhookOnly",
			clear:   webhook,
			trp:     validConfig.TRP.MTLSConfig,
			webhook: validConfig.Node.MTLSConfig,
		},
		{
			name:    "CertsOnly",
			clear:   []string{"TRISA_TRP_CERTS", "TRISA_WEBHOOK_CERTS"},
			trp:     config.MTLSConfig{Certs: validConfig.Node.Certs, Pool: validConfig.TRP.Pool},
			webhook: config.MTLSConfig{Certs: validConfig.Node.Certs, Pool: validConfig.Webhook.Pool},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			setTestEnv(t, nil, tc.clear...)

			conf, err := config.New()
			require.NoError(t, err, "could not process configuration from the environment")

			require.Equal(t, validConfig.Node.MTLSConfig, conf.Node.MTLSConfig, "expected trisa certs to be unchanged")
			require.Equal(t, tc.trp, conf.TRP.MTLSConfig, "unexpected trp certs")
			require.Equal(t, tc.webhook, conf.Webhook.MTLSConfig, "unexpected webhook certs")
		})
	}
}

func TestRegionInfo(t *testing.T) {
	t.Run("Unavailable", func(t *testing.T) {
		conf := config.RegionInfo{}
		require.False(t, conf.Available(), "expected region info to be unavailable")
	})

	t.Run("Available", func(t *testing.T) {
		tests := map[string]config.RegionInfo{
			"All":     validConfig.RegionInfo,
			"ID":      {ID: 2840302},
			"Name":    {Name: "us-east4c"},
			"Country": {Country: "US"},
			"Cloud":   {Cloud: "GCP"},
			"Cluster": {Cluster: "rotational-testing-gke-9"},
		}

		for name, tc := range tests {
			require.True(t, tc.Available(), "expected region info with %s to be available", name)
		}
	})
}

//============================================================================
// Config Package Management
//============================================================================

// Resets the package config before the test and after it completes so that the
// globally cached config cannot leak between tests.
func resetConfig(t *testing.T) {
	t.Helper()
	config.Reset()
	t.Cleanup(config.Reset)
}

func TestGet(t *testing.T) {
	t.Run("LoadsOnce", func(t *testing.T) {
		resetConfig(t)
		t.Cleanup(testEnv.Set())

		conf, err := config.Get()
		require.NoError(t, err, "could not load config from the environment")
		require.Equal(t, validConfig, conf)

		// Changes to the environment should not be observed until reset.
		t.Cleanup(contest.Env{"TRISA_ORGANIZATION": "Reloaded"}.Set())

		conf, err = config.Get()
		require.NoError(t, err, "could not get cached config")
		require.Equal(t, validConfig, conf, "expected the config to be loaded only once")

		config.Reset()
		conf, err = config.Get()
		require.NoError(t, err, "could not reload config from the environment")
		require.Equal(t, "Reloaded", conf.Organization, "expected the config to be reloaded after reset")
	})

	t.Run("Error", func(t *testing.T) {
		resetConfig(t)
		t.Cleanup(testEnv.Set())
		restore := contest.Env{"TRISA_MODE": "invalid"}.Set()

		conf, err := config.Get()
		require.ErrorContains(t, err, `"invalid" is not a valid gin mode`)
		require.Equal(t, config.Config{}, conf, "expected a zero valued config on error")
		require.PanicsWithValue(t, err, func() { config.MustGet() })

		// The load error is also only determined once, until reset.
		restore()
		_, again := config.Get()
		require.Equal(t, err, again, "expected the same load error without a reset")

		config.Reset()
		require.Equal(t, validConfig, config.MustGet(), "expected a valid config after reset")
	})

	t.Run("Concurrent", func(t *testing.T) {
		resetConfig(t)
		t.Cleanup(testEnv.Set())

		var (
			wg    sync.WaitGroup
			confs = make([]config.Config, 16)
			errs  = make([]error, len(confs))
		)

		for i := range confs {
			wg.Go(func() { confs[i], errs[i] = config.Get() })
		}
		wg.Wait()

		for i, conf := range confs {
			require.NoError(t, errs[i])
			require.Equal(t, validConfig, conf)
		}
	})
}

func TestSet(t *testing.T) {
	t.Run("BeforeGet", func(t *testing.T) {
		// With no environment, the config can only come from Set.
		resetConfig(t)
		t.Cleanup(testEnv.Clear())

		require.NoError(t, config.Set(validConfig), "could not set a valid config")

		conf, err := config.Get()
		require.NoError(t, err, "could not get config after set")
		require.Equal(t, validConfig, conf)
		require.Equal(t, validConfig, config.MustGet())
	})

	t.Run("AfterGet", func(t *testing.T) {
		resetConfig(t)
		t.Cleanup(testEnv.Set())

		conf, err := config.Get()
		require.NoError(t, err, "could not load config from the environment")
		require.Equal(t, validConfig, conf)

		require.NoError(t, config.Set(defaultConfig), "could not set a valid config")
		require.Equal(t, defaultConfig, config.MustGet(), "expected the loaded config to be replaced")
	})

	// Set validates the nested configs as well as the top level config.
	t.Run("Invalid", func(t *testing.T) {
		nocerts := validConfig
		nocerts.Node.Certs = ""

		tests := []struct {
			name string
			conf config.Config
			err  string // expected to be contained in the error
		}{
			{"Mode", config.Config{Mode: "invalid"}, `mode "invalid" is not a valid gin mode`},
			{"Nested", nocerts, "trisa.certs is required but not set"},
			{"Empty", config.Config{Mode: "test"}, "trisa.certs is required but not set"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				resetConfig(t)
				t.Cleanup(testEnv.Set())

				// An invalid config should not prevent the config from being loaded.
				require.ErrorContains(t, config.Set(tc.conf), tc.err)

				conf, err := config.Get()
				require.NoError(t, err, "expected the config to be loaded after a failed set")
				require.Equal(t, validConfig, conf)

				// An invalid config should not replace the current config.
				require.NoError(t, config.Set(defaultConfig), "could not set a valid config")
				require.ErrorContains(t, config.Set(tc.conf), tc.err)

				conf, err = config.Get()
				require.NoError(t, err, "expected the config to be unchanged after a failed set")
				require.Equal(t, defaultConfig, conf)
			})
		}
	})
}
