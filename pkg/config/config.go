package config

import (
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/trisacrypto/envoy/pkg/emails"
	"github.com/trisacrypto/envoy/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/rotationalio/confire"
	"github.com/rs/zerolog"
)

// All environment variables will have this prefix unless otherwise defined in struct
// tags. For example, the conf.LogLevel environment variable will be TRISA_LOG_LEVEL
// because of this prefix and the split_words struct tag in the conf below.
const Prefix = "trisa"

// If any of the keys are not set or are empty, they will be remapped to the value
// of the specified environment variable. This is primarily used when we want to be
// able to override a required environment variable with a different specified value
// without modifying how a configuration is validated. E.g. to use the same certs for
// TRISA, TRP, and Webhook if those environment variables are not set.
var environRemap = map[string]string{
	"TRISA_TRP_CERTS":     "TRISA_NODE_CERTS",
	"TRISA_TRP_POOL":      "TRISA_NODE_POOL",
	"TRISA_WEBHOOK_CERTS": "TRISA_NODE_CERTS",
	"TRISA_WEBHOOK_POOL":  "TRISA_NODE_POOL",
}

// Config contains all of the configuration parameters for the trisa node and is
// loaded from the environment or a configuration file with reasonable defaults for
// values that are omitted. The Config should be validated in preparation for running
// the server to ensure that all server operations work as expected.
type Config struct {
	Maintenance     bool                `default:"false" desc:"if true, the node will start in maintenance mode"`
	Organization    string              `default:"Envoy" desc:"specify the name of the organization of the Envoy node for display purposes"`
	Mode            string              `default:"release" desc:"specify the mode of the server (release, debug, testing)"`
	LogLevel        logger.LevelDecoder `split_words:"true" default:"info" desc:"specify the verbosity of logging (trace, debug, info, warn, error, fatal panic)"`
	ConsoleLog      bool                `split_words:"true" default:"false" desc:"if true logs colorized human readable output instead of json"`
	DatabaseURL     string              `split_words:"true" default:"sqlite3:///trisa.db" desc:"dsn containing backend database configuration"`
	SearchThreshold float64             `split_words:"true" default:"0.0" desc:"specify the threshold for fuzzy search (0.0 to 1.0)"`
	Web             WebConfig           `split_words:"true"`
	Webhook         WebhookConfig       `split_words:"true"`
	Node            TRISAConfig         `split_words:"true"`
	DirectorySync   DirectorySyncConfig `split_words:"true"`
	TRP             TRPConfig           `split_words:"true"`
	Sunrise         SunriseConfig       `split_words:"true"`
	Email           emails.Config       `split_words:"true"`
	RegionInfo      RegionInfo          `split_words:"true"`
	processed       bool
}

// WebConfig specifies the configuration for the web UI to manage the TRISA node and
// TRISA transactions. The web UI can be enabled or disabled and runs independently of
// the other servers on the node.
type WebConfig struct {
	Maintenance   bool           `env:"TRISA_MAINTENANCE" desc:"if true sets the web UI to maintenance mode; inherited from parent"`
	Enabled       bool           `default:"true" desc:"if false, the web UI server will not be run"`
	APIEnabled    bool           `default:"true" split_words:"true" desc:"if false, the API server will return unavailable when accessed; subordinate to the enabled flag"`
	UIEnabled     bool           `default:"true" split_words:"true" desc:"if false, the UI server will return unavailable when accessed; subordinate to the enabled flag"`
	BindAddr      string         `default:":8000" split_words:"true" desc:"the ip address and port to bind the web server on"`
	Origin        string         `default:"http://localhost:8000" desc:"origin (url) of the web ui for creating endpoints and CORS access"`
	TRISAEndpoint string         `env:"TRISA_ENDPOINT" desc:"trisa endpoint as assigned to the mTLS certificates for the trisa node"`
	TRPEndpoint   string         `env:"TRISA_TRP_ENDPOINT" desc:"trp endpoint as assigned to the mTLS certificates for the trp node"`
	DocsName      string         `split_words:"true" desc:"the display name for the API docs server in the Swagger app"`
	Auth          AuthConfig     `split_words:"true"`
	Daybreak      DaybreakConfig `split_words:"true"`
}

// AuthConfig specifies the configuration for authenticating WebUI requests
type AuthConfig struct {
	Keys            map[string]string `required:"false" desc:"optional static key configuration as a map of keyID to path on disk"`
	Audience        string            `default:"http://localhost:8000" desc:"value for the aud jwt claim"`
	Issuer          string            `default:"http://localhost:8000" desc:"value for the iss jwt claim"`
	CookieDomain    string            `split_words:"true" default:"localhost" desc:"limit cookies to the specified domain (exclude port)"`
	AccessTokenTTL  time.Duration     `split_words:"true" default:"1h" desc:"the amount of time before an access token expires"`
	RefreshTokenTTL time.Duration     `split_words:"true" default:"2h" desc:"the amount of time before a refresh token expires"`
	TokenOverlap    time.Duration     `split_words:"true" default:"-15m" desc:"the amount of overlap between the access and refresh token"`
}

// TRISAConfig is a generic configuration for the TRISA node options
type TRISAConfig struct {
	MTLSConfig
	Maintenance         bool            `env:"TRISA_MAINTENANCE" desc:"if true sets the TRISA node to maintenance mode; inherited from parent"`
	Endpoint            string          `env:"TRISA_ENDPOINT" desc:"trisa endpoint as assigned to the mTLS certificates for the trisa node"`
	Enabled             bool            `default:"true" desc:"if false, the TRISA node server will not be run"`
	BindAddr            string          `split_words:"true" default:":8100" desc:"the ip address and port to bind the trisa grpc server on"`
	KeyExchangeCacheTTL time.Duration   `split_words:"true" default:"24h"`
	Directory           DirectoryConfig `split_words:"true"`
}

// DirectoryConfig is a generic configuration for connecting to a TRISA GDS service.
// By default the configuration connects to the MainNet GDS, replace trisa.directory
// with testnet.directory to connect to the TestNet instead.
type DirectoryConfig struct {
	Insecure        bool   `default:"false" desc:"if true, do not connect using TLS"`
	Endpoint        string `default:"api.trisa.directory:443" required:"true" desc:"the endpoint of the public GDS service"`
	MembersEndpoint string `default:"members.trisa.directory:443" required:"true" split_words:"true" desc:"the endpoint of the members only GDS service"`
}

// DirectorySyncConfig manages the behavior of synchronizing counterparty VASPs with the
// TRISA Global Directory Service (GDS).
type DirectorySyncConfig struct {
	Enabled  bool          `default:"true" desc:"if false, the sync background service will not be run"`
	Interval time.Duration `default:"6h" desc:"the interval synchronization is run"`
}

type TRPConfig struct {
	MTLSConfig
	Maintenance bool   `env:"TRISA_MAINTENANCE" desc:"if true sets the trp node to maintenance mode; inherited from parent"`
	Enabled     bool   `default:"true" desc:"if false, the trp server will not be run"`
	BindAddr    string `default:":8200" split_words:"true" desc:"the ip address and port to bind the trp server on"`
	UseMTLS     bool   `default:"true" split_words:"true" desc:"if true the trp server will require mTLS authentication"`
	Identity    TRPIdentityConfig
}

type TRPIdentityConfig struct {
	VASPName string `split_words:"true" desc:"specify the name response in a trp identity request"`
	LEI      string `required:"false" desc:"the lei of your vasp to respond to a trp identity request"`
}

type SunriseConfig struct {
	Enabled        bool   `default:"true" desc:"if false, it will not be possible to send sunrise emails and sunrise endpoints will return a 404"`
	BaseURL        string `split_words:"true" env:"TRISA_WEB_ORIGIN" desc:"the base url for handling sunrise requests, to put into outgoing emails"`
	InviteEndpoint string `split_words:"true" default:"/sunrise/verify" desc:"the path of the endpoint for sunrise token verification"`
	RequireOTP     bool   `split_words:"true" default:"true" desc:"if true, the sunrise verification process will require an OTP to be entered by the user even on successful verification token validation"`
	inviteURL      *url.URL
}

type WebhookConfig struct {
	MTLSConfig
	URL               string `required:"false" desc:"specify a callback webhook that incoming travel rule messages will be posted to"`
	UseMTLS           bool   `default:"false" split_words:"true" desc:"if true, the webhook client will use mTLS to secure its connection to the webhook server"`
	AuthKeyID         string `required:"false" split_words:"true" desc:"used to identify the shared secret key used for hmac authorization"`
	AuthKeySecret     string `required:"false" split_words:"true" desc:"specify a shared secret key to use for hmac authorization"`
	RequireServerAuth bool   `default:"false" split_words:"true" desc:"if true, the webhook server will require a valid hmac authorization header in webhook responses"`
}

type DaybreakConfig struct {
	Enabled bool `default:"false" desc:"if true, the Daybreak counterparties list tab will be shown, otherwise not shown"`
}

// Optional region and deployment information associated with the node.
type RegionInfo struct {
	ID      int32  `env:"REGION_INFO_ID" desc:"the 7 digit region identifier code"`
	Name    string `env:"REGION_INFO_NAME" desc:"the name of the region"`
	Country string `env:"REGION_INFO_COUNTRY" desc:"the alpha-2 country code of the region"`
	Cloud   string `env:"REGION_INFO_CLOUD" desc:"the cloud service provider"`
	Cluster string `env:"REGION_INFO_CLUSTER" desc:"the name of the cluster the node is hosted in"`
}

func New() (conf Config, err error) {
	// Use TRISA certs for TRP and Webhook if not otherwise specified.
	for k, v := range environRemap {
		if _, ok := os.LookupEnv(k); !ok {
			os.Setenv(k, os.Getenv(v))
		}
	}

	if err = confire.Process(Prefix, &conf); err != nil {
		return Config{}, err
	}

	if err = conf.Validate(); err != nil {
		return Config{}, err
	}

	conf.processed = true
	return conf, nil
}

// Returns true if the config has not been correctly processed from the environment.
func (c Config) IsZero() bool {
	return !c.processed
}

// Custom validations are added here, particularly validations that require one or more
// fields to be processed before the validation occurs.
// NOTE: ensure that all nested config validation methods are called here.
func (c Config) Validate() (err error) {
	if c.Mode != gin.ReleaseMode && c.Mode != gin.DebugMode && c.Mode != gin.TestMode {
		return fmt.Errorf("invalid configuration: %q is not a valid gin mode", c.Mode)
	}

	if err = c.Web.Validate(); err != nil {
		return err
	}

	if err = c.Webhook.Validate(); err != nil {
		return err
	}

	if err = c.Node.Validate(); err != nil {
		return err
	}

	if err = c.TRP.Validate(); err != nil {
		return err
	}

	if err = c.Email.Validate(); err != nil {
		return err
	}

	return nil
}

func (c Config) GetLogLevel() zerolog.Level {
	return zerolog.Level(c.LogLevel)
}

func (c WebConfig) Validate() (err error) {
	// If not enabled, do not validate the config.
	if !c.Enabled {
		return nil
	}

	// If enabled but neither UI or API is enabled, return a warning
	if c.Enabled && !c.APIEnabled && !c.UIEnabled {
		return errors.New("invalid configuration: if enabled, either the api, ui, or both need to be enabled")
	}

	if c.BindAddr == "" {
		return errors.New("invalid configuration: bindaddr is required")
	}

	if c.Origin == "" {
		return errors.New("invalid configuration: origin is required")
	}

	return nil
}

func (c WebConfig) ResetPasswordURL() *url.URL {
	u, _ := url.Parse(c.Origin)
	u.Path = "/reset-password"
	return u
}

func (c WebhookConfig) Validate() (err error) {
	if c.Enabled() {
		if _, err = url.Parse(c.URL); err != nil {
			return fmt.Errorf("invalid configuration: could not parse webhook url: %w", err)
		}

		if c.AuthKeySecret != "" {
			if _, err := hex.DecodeString(strings.ToLower(c.AuthKeySecret)); err != nil {
				return fmt.Errorf("invalid configuration: could not decode webhook auth key: %w", err)
			}

			if c.AuthKeyID == "" {
				return errors.New("invalid configuration: webhook auth key id is required when auth key secret is set")
			}
		}

		if c.RequireServerAuth && (c.AuthKeySecret == "" || c.AuthKeyID == "") {
			return errors.New("invalid configuration: webhook server auth is enabled but no auth key is specified")
		}

		if c.UseMTLS {
			if c.Certs == "" {
				return errors.New("invalid configuration: specify certificates path for webhook mTLS")
			}
		}
	}
	return nil
}

func (c WebhookConfig) Enabled() bool {
	return c.URL != ""
}

func (c WebhookConfig) RequireClientAuth() bool {
	return c.AuthKeySecret != "" || c.AuthKeyID != ""
}

func (c WebhookConfig) Endpoint() *url.URL {
	if c.URL == "" {
		return nil
	}

	u, _ := url.Parse(c.URL)
	return u
}

func (c WebhookConfig) DecodeAuthKey() []byte {
	if c.AuthKeySecret == "" {
		return nil
	}

	key, _ := hex.DecodeString(strings.ToLower(c.AuthKeySecret))
	return key
}

// Validate that the TRISA config has mTLS certificates for operation.
func (c *TRISAConfig) Validate() error {
	if c.Certs == "" {
		return errors.New("invalid configuration: specify certificates path")
	}
	return nil
}

// Network parses the directory service endpoint to identify the network of the directory.
func (c DirectoryConfig) Network() string {
	endpoint := c.Endpoint
	if endpoint == "" {
		return ""
	}

	// Strip off any scheme if present
	if uri, err := url.Parse(c.Endpoint); err == nil {
		if uri.Host != "" {
			endpoint = uri.Host
		}
	}

	endpoint = strings.Split(endpoint, ":")[0] // strip the port from the endpoint
	parts := strings.Split(endpoint, ".")
	if len(parts) < 2 {
		return endpoint
	}
	return strings.Join(parts[len(parts)-2:], ".")
}

// Validate that the TRP config is suitable for operation of the server
func (c *TRPConfig) Validate() error {
	if c.Enabled {
		if c.BindAddr == "" {
			return errors.New("invalid configuration: missing bind address")
		}

		// If use mTLS is specified then a path to the certificates must be available
		if c.UseMTLS {
			if c.Certs == "" {
				return errors.New("invalid configuration: specify certificates path")
			}
		}
	}
	return nil
}

func (c *SunriseConfig) InviteURL() *url.URL {
	if c.inviteURL == nil {
		c.inviteURL, _ = url.Parse(c.BaseURL)
		c.inviteURL.Path = c.InviteEndpoint
	}
	return c.inviteURL
}

// Determines if region info is available or not.
func (c RegionInfo) Available() bool {
	return c.ID > 0 || c.Name != "" || c.Country != "" || c.Cloud != "" || c.Cluster != ""
}
