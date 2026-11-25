package config

import (
	"os"
	"sync"

	"github.com/trisacrypto/envoy/pkg/emails"
	"github.com/trisacrypto/envoy/pkg/logger"
	"github.com/trisacrypto/envoy/pkg/store/dsn"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"go.rtnl.ai/confire"
	"go.rtnl.ai/confire/validate"
)

// All environment variables will have this prefix unless otherwise defined in struct
// tags. For example, the conf.LogLevel environment variable will be TRISA_LOG_LEVEL
// because of this prefix and the split_words struct tag in the conf below.
const Prefix = "trisa"

// If any of the keys are not set, they will be remapped to the value of the specified
// environment variable. This is primarily used when we want to be able to override a
// required environment variable with a different specified value without modifying how
// a configuration is validated. E.g. to use the same certs for TRISA, TRP, and Webhook
// if those environment variables are not set.
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
	Mode            string              `default:"release" desc:"specify the mode of the server (release, debug, test)"`
	LogLevel        logger.LevelDecoder `split_words:"true" default:"info" desc:"specify the verbosity of logging (trace, debug, info, warn, error, fatal panic)"`
	ConsoleLog      bool                `split_words:"true" default:"false" desc:"if true logs colorized human readable output instead of json"`
	DatabaseURL     string              `split_words:"true" default:"sqlite3:///trisa.db" desc:"dsn containing backend database configuration"`
	SearchThreshold float64             `split_words:"true" default:"0.0" desc:"specify the threshold for fuzzy search (0.0 to 1.0)"`
	Boot            BootConfig          `split_words:"true"`
	Web             WebConfig           `split_words:"true"`
	Webhook         WebhookConfig       `split_words:"true"`
	Node            TRISAConfig         `split_words:"true"`
	DirectorySync   DirectorySyncConfig `split_words:"true"`
	TRP             TRPConfig           `split_words:"true"`
	Sunrise         SunriseConfig       `split_words:"true"`
	Daybreak        DaybreakConfig      `split_words:"true"`
	Email           emails.Config       `split_words:"true"`
	RegionInfo      RegionInfo          `split_words:"true"`
}

type BootConfig struct {
	RegenerateTravelAddresses bool `default:"false" split_words:"true" desc:"if true, the travel addresses will be regenerated when envoy starts"`
}

// Optional region and deployment information associated with the node.
type RegionInfo struct {
	ID      int32  `env:"REGION_INFO_ID" desc:"the 7 digit region identifier code"`
	Name    string `env:"REGION_INFO_NAME" desc:"the name of the region"`
	Country string `env:"REGION_INFO_COUNTRY" desc:"the alpha-2 country code of the region"`
	Cloud   string `env:"REGION_INFO_CLOUD" desc:"the cloud service provider"`
	Cluster string `env:"REGION_INFO_CLUSTER" desc:"the name of the cluster the node is hosted in"`
	Zone    string `env:"REGION_INFO_ZONE" desc:"the zone the node is hosted in"`
}

func New() (conf *Config, err error) {
	// Use TRISA certs for TRP and Webhook if not otherwise specified.
	for k, v := range environRemap {
		if _, ok := os.LookupEnv(k); !ok {
			os.Setenv(k, os.Getenv(v))
		}
	}

	conf = &Config{}
	if err = confire.Process(Prefix, conf); err != nil {
		return nil, err
	}
	return conf, nil
}

func (c Config) Validate() (err error) {
	if c.Mode != gin.ReleaseMode && c.Mode != gin.DebugMode && c.Mode != gin.TestMode {
		err = confire.Join(err, confire.Invalid("", "mode", "%q is not a valid gin mode", c.Mode))
	}
	return err
}

func (c Config) GetLogLevel() zerolog.Level {
	return zerolog.Level(c.LogLevel)
}

// DatabaseDSN returns the database dsn for the config.
func (c Config) DatabaseDSN() *dsn.DSN {
	uri, _ := dsn.Parse(c.DatabaseURL)
	return uri
}

// Determines if region info is available or not.
func (c RegionInfo) Available() bool {
	return c.ID > 0 || c.Name != "" || c.Country != "" || c.Cloud != "" || c.Cluster != ""
}

//============================================================================
// Config Package Management
//============================================================================

var (
	mu     sync.RWMutex
	err    error
	loaded bool
	conf   *Config
)

func Get() (Config, error) {
	mu.RLock()
	if !loaded {
		mu.RUnlock()

		// Perform double checked locking to avoid race conditions.
		mu.Lock()
		if !loaded {
			conf, err = New()
			loaded = true
		}
		mu.Unlock()
		mu.RLock()
	}

	defer mu.RUnlock()
	if conf != nil {
		return *conf, err
	}
	return Config{}, err
}

func MustGet() Config {
	conf, err := Get()
	if err != nil {
		panic(err)
	}
	return conf
}

// Set the config as the global configuration, overriding the current configuration. If
// the configuration is invalid, an error will be returned and the global configuration
// will remain unchanged.
func Set(c Config) error {
	if verr := validate.Validate(&c); verr != nil {
		return verr
	}

	mu.Lock()
	defer mu.Unlock()

	conf = &c
	err = nil
	loaded = true

	return nil
}

func Reset() {
	mu.Lock()
	defer mu.Unlock()
	conf = nil
	err = nil
	loaded = false
}
