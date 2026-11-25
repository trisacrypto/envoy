package config

import "time"

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
