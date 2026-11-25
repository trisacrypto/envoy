package config

import (
	"encoding/hex"
	"net/url"
	"strings"

	"go.rtnl.ai/confire"
)

// WebConfig specifies the configuration for the web UI to manage the TRISA node and
// TRISA transactions. The web UI can be enabled or disabled and runs independently of
// the other servers on the node.
type WebConfig struct {
	Maintenance   bool       `env:"TRISA_MAINTENANCE" desc:"if true sets the web UI to maintenance mode; inherited from parent"`
	Enabled       bool       `default:"true" desc:"if false, the web UI server will not be run"`
	APIEnabled    bool       `default:"true" split_words:"true" desc:"if false, the API server will return unavailable when accessed; subordinate to the enabled flag"`
	UIEnabled     bool       `default:"true" split_words:"true" desc:"if false, the UI server will return unavailable when accessed; subordinate to the enabled flag"`
	BindAddr      string     `default:":8000" split_words:"true" desc:"the ip address and port to bind the web server on"`
	Origin        string     `default:"http://localhost:8000" desc:"origin (url) of the web ui for creating endpoints and CORS access"`
	TRISAEndpoint string     `env:"TRISA_ENDPOINT" desc:"trisa endpoint as assigned to the mTLS certificates for the trisa node"`
	TRPEndpoint   string     `env:"TRISA_TRP_ENDPOINT" desc:"trp endpoint as assigned to the mTLS certificates for the trp node"`
	DocsName      string     `default:"Envoy API Reference" split_words:"true" desc:"the display name for the API docs server in the Swagger app"`
	LogoURI       string     `split_words:"true" desc:"use a custom logo for the web UI"`
	Auth          AuthConfig `split_words:"true"`
}

type WebhookConfig struct {
	MTLSConfig
	URL               string `required:"false" desc:"specify a callback webhook that incoming travel rule messages will be posted to"`
	UseMTLS           bool   `default:"false" split_words:"true" desc:"if true, the webhook client will use mTLS to secure its connection to the webhook server"`
	AuthKeyID         string `required:"false" split_words:"true" desc:"used to identify the shared secret key used for hmac authorization"`
	AuthKeySecret     string `required:"false" split_words:"true" desc:"specify a shared secret key to use for hmac authorization"`
	RequireServerAuth bool   `default:"false" split_words:"true" desc:"if true, the webhook server will require a valid hmac authorization header in webhook responses"`
}

func (c *WebConfig) Validate() (err error) {
	// If not enabled, do not validate the config.
	if !c.Enabled {
		return nil
	}

	// If enabled but neither UI or API is enabled, return a warning
	if c.Enabled && !c.APIEnabled && !c.UIEnabled {
		err = confire.Join(err, confire.Invalid("web", "enabled", "if enabled, either the api, ui, or both need to be enabled"))
	}

	if c.BindAddr == "" {
		err = confire.Join(err, confire.Required("web", "bindaddr"))
	}

	if c.Origin == "" {
		err = confire.Join(err, confire.Required("web", "origin"))
	} else {
		if uri, perr := url.Parse(c.Origin); perr != nil || uri.Scheme == "" || uri.Host == "" {
			err = confire.Join(err, confire.Invalid("web", "origin", "%q is not a valid url", c.Origin))
		}
	}

	return err
}

func (c WebConfig) ResetPasswordURL() *url.URL {
	u, _ := url.Parse(c.Origin)
	if u == nil {
		u = &url.URL{}
	}

	u.Path = "/reset-password"
	return u
}

func (c *WebhookConfig) Validate() (err error) {
	// If not enabled, do not validate the config.
	if !c.Enabled() {
		return nil
	}

	if uri, perr := url.Parse(c.URL); perr != nil || uri.Scheme == "" || uri.Host == "" {
		err = confire.Join(err, confire.Invalid("webhook", "url", "is not a valid url"))
	}

	if c.AuthKeySecret != "" {
		if _, perr := hex.DecodeString(strings.ToLower(c.AuthKeySecret)); perr != nil {
			err = confire.Join(err, confire.Invalid("webhook", "authkeysecret", "could not hex decode auth secret"))
		}

		if c.AuthKeyID == "" {
			err = confire.Join(err, confire.Invalid("webhook", "authkeyid", "auth key id is required when auth key secret is set"))
		}
	}

	if c.RequireServerAuth && (c.AuthKeySecret == "" || c.AuthKeyID == "") {
		err = confire.Join(err, confire.Invalid("webhook", "requireserverauth", "webhook server auth is enabled but no auth key is specified"))
	}

	if c.UseMTLS {
		if c.Certs == "" {
			err = confire.Join(err, confire.Invalid("webhook", "certs", "certificates are required when use mtls is specified"))
		}
	}

	return err
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
