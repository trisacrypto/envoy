package config

import (
	"errors"
	"fmt"
	"self-hosted-node/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/rotationalio/confire"
	"github.com/rs/zerolog"
)

// All environment variables will have this prefix unless otherwise defined in struct
// tags. For example, the conf.LogLevel environment variable will be TRISA_LOG_LEVEL
// because of this prefix and the split_words struct tag in the conf below.
const Prefix = "trisa"

// Config contains all of the configuration parameters for the trisa node and is
// loaded from the environment or a configuration file with reasonable defaults for
// values that are omitted. The Config should be validated in preparation for running
// the server to ensure that all server operations work as expected.
type Config struct {
	Maintenance bool                `default:"false" desc:"if true, the node will start in maintenance mode"`
	Mode        string              `default:"release" desc:"specify the mode of the server (release, debug, testing)"`
	LogLevel    logger.LevelDecoder `split_words:"true" default:"info" desc:"specify the verbosity of logging (trace, debug, info, warn, error, fatal panic)"`
	ConsoleLog  bool                `split_words:"true" default:"false" desc:"if true logs colorized human readable output instead of json"`
	Web         WebConfig           `split_words:"true"`
	processed   bool
}

// WebConfig specifies the configuration for the web UI to manage the TRISA node and
// TRISA transactions. The web UI can be enabled or disabled and runs independently of
// the other servers on the node.
type WebConfig struct {
	Maintenance bool   `env:"TRISA_MAINTENANCE" desc:"if true sets the web UI to maintenance mode; inherited from parent"`
	Enabled     bool   `default:"true" desc:"if false, the web UI server will not be run"`
	BindAddr    string `default:":8000" split_words:"true" desc:"the ip address and port to bind the web server on"`
	Origin      string `default:"http://localhost:8000" desc:"origin (url) of the web ui for creating endpoints and CORS access"`
}

func New() (conf Config, err error) {
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

	if c.BindAddr == "" {
		return errors.New("invalid configuration: bindaddr is required")
	}

	if c.Origin == "" {
		return errors.New("invalid configuration: origin is required")
	}

	return nil
}
