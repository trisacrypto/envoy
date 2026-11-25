package config

import (
	"net/url"
	"strings"
	"time"

	"go.rtnl.ai/confire"
)

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

// Validate that the TRISA config has mTLS certificates for operation.
func (c *TRISAConfig) Validate() (err error) {
	if c.Certs == "" {
		err = confire.Join(err, confire.Required("trisa", "certs"))
	}
	return err
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
