package config

import (
	"net/url"
	"time"

	"go.rtnl.ai/confire"
)

type SunriseConfig struct {
	Enabled        bool   `default:"true" desc:"if false, it will not be possible to send sunrise emails and sunrise endpoints will return a 404"`
	BaseURL        string `split_words:"true" env:"TRISA_WEB_ORIGIN" default:"http://localhost:8000" desc:"the base url for handling sunrise requests, to put into outgoing emails"`
	InviteEndpoint string `split_words:"true" default:"/sunrise/verify" desc:"the path of the endpoint for sunrise token verification"`
	RequireOTP     bool   `split_words:"true" default:"true" desc:"if true, the sunrise verification process will require an OTP to be entered by the user even on successful verification token validation"`
}

type DaybreakConfig struct {
	Enabled      bool          `default:"false" desc:"if true, the Daybreak counterparties will sync and be available in the web UI"`
	Endpoint     string        `default:"https://daybreaks.directory" desc:"the endpoint of the Daybreak service"`
	ClientID     string        `split_words:"true" desc:"the client id of your license key, required if enabled"`
	ClientSecret string        `split_words:"true" desc:"the client secret of your license key, required if enabled"`
	SyncInterval time.Duration `split_words:"true" default:"5m" desc:"the interval at which to sync the Daybreak counterparties"`
	SyncJitter   time.Duration `split_words:"true" default:"1m" desc:"the window around the interval to randomize interval timing"`
}

func (c *SunriseConfig) InviteURL() *url.URL {
	invite, _ := url.Parse(c.BaseURL)
	if invite == nil {
		invite = &url.URL{}
	}

	invite.Path = c.InviteEndpoint
	return invite
}

func (c *DaybreakConfig) Validate() (err error) {
	if !c.Enabled {
		return nil
	}

	if c.Endpoint == "" {
		err = confire.Join(err, confire.Required("daybreak", "endpoint"))
	} else if uri, perr := url.Parse(c.Endpoint); perr != nil || uri.Scheme == "" || uri.Host == "" {
		err = confire.Join(err, confire.Invalid("daybreak", "endpoint", "not a valid url"))
	}

	if c.ClientID == "" {
		err = confire.Join(err, confire.Required("daybreak", "client_id"))
	}

	if c.ClientSecret == "" {
		err = confire.Join(err, confire.Required("daybreak", "client_secret"))
	}

	if c.SyncInterval <= time.Minute {
		err = confire.Join(err, confire.Invalid("daybreak", "sync_interval", "sync interval must be greater than one minute"))
	}

	if c.SyncJitter <= time.Second {
		err = confire.Join(err, confire.Invalid("daybreak", "sync_jitter", "sync jitter must be greater than one second"))
	}

	return err
}
