package config

import (
	"go.rtnl.ai/confire"
)

type TRPConfig struct {
	MTLSConfig
	Maintenance bool   `env:"TRISA_MAINTENANCE" desc:"if true sets the trp node to maintenance mode; inherited from parent"`
	Endpoint    string `env:"TRISA_TRP_ENDPOINT" desc:"trp endpoint as assigned to the mTLS certificates for the trp node"`
	Enabled     bool   `default:"true" desc:"if false, the trp server will not be run"`
	BindAddr    string `default:":8200" split_words:"true" desc:"the ip address and port to bind the trp server on"`
	UseMTLS     bool   `default:"true" split_words:"true" desc:"if true the trp server will require mTLS authentication"`
	Identity    TRPIdentityConfig
}

type TRPIdentityConfig struct {
	VASPName string `split_words:"true" desc:"specify the name response in a trp identity request"`
	LEI      string `required:"false" desc:"the lei of your vasp to respond to a trp identity request"`
}

// Validate that the TRP config is suitable for operation of the server
func (c *TRPConfig) Validate() (err error) {
	// Do not validate if the TRP server is not enabled.
	if !c.Enabled {
		return nil
	}

	if c.Endpoint == "" {
		err = confire.Join(err, confire.Required("trp", "endpoint"))
	}

	if c.BindAddr == "" {
		err = confire.Join(err, confire.Required("trp", "bindaddr"))
	}

	// If use mTLS is specified then a path to the certificates must be available
	if c.UseMTLS {
		if c.Certs == "" {
			err = confire.Join(err, confire.Invalid("trp", "certs", "certificates are required when use mtls is specified"))
		}
	}
	return err
}
