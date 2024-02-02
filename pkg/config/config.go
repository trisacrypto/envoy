package config

import "self-hosted-node/pkg/logger"

// All environment variables will have this prefix unless otherwise defined in struct
// tags. For example, the conf.LogLevel environment variable will be TRISA_LOG_LEVEL
// because of this prefix and the split_words struct tag in the conf below.
const prefix = "trisa"

// Config contains all of the configuration parameters for an rtnl server and is
// loaded from the environment or a configuration file with reasonable defaults for
// values that are omitted. The Config should be validated in preparation for running
// the server to ensure that all server operations work as expected.
type Config struct {
	Maintenance  bool                `default:"false" yaml:"maintenance"`
	Mode         string              `default:"release"`
	LogLevel     logger.LevelDecoder `split_words:"true" default:"info"`
	ConsoleLog   bool                `split_words:"true" default:"false"`
	BindAddr     string              `split_words:"true" default:":4444"`
	AllowOrigins []string            `split_words:"true" default:"http://localhost:4444"`
	Origin       string              `default:"https://localhost:4444"`
	processed    bool
}
