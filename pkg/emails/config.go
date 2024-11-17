package emails

import (
	"errors"
	"fmt"
	"net/smtp"

	"github.com/jordan-wright/email"
)

// The emails config allows users to either send messages via SendGrid or via SMTP.
type Config struct {
	FromEmail string         `split_words:"true" desc:"the email address that messages are sent from"`
	Testing   bool           `split_words:"true" default:"false" desc:"set the emailer to testing mode to ensure no live emails are sent"`
	SMTP      SMTPConfig     `split_words:"true"`
	SendGrid  SendGridConfig `split_words:"false"`
}

// Configuration for sending emails via SMTP.
type SMTPConfig struct {
	Host       string `required:"false" desc:"the smtp host without the port e.g. smtp.example.com; if set SMTP will be used, cannot be set with sendgrid api key"`
	Port       uint16 `default:"587" desc:"the port to access the smtp server on"`
	Username   string `required:"false" desc:"the username for authentication with the smtp server"`
	Password   string `required:"false" desc:"the password for authentication with the smtp server"`
	UseCRAMMD5 bool   `env:"USE_CRAM_MD5" default:"false" desc:"use CRAM-MD5 auth as defined in RFC 2195 instead of simple authentication"`
	PoolSize   int    `split_words:"true" default:"2" desc:"the smtp connection pool size to use for concurrent email sending"`
}

// Configuration for sending emails using SendGrid.
type SendGridConfig struct {
	APIKey string `split_words:"true" required:"false" desc:"set the sendgrid api key to use sendgrid as the email backend"`
}

func (c Config) Validate() (err error) {
	// It is important that if we're in testing mode that we do not validate the
	// config because this creates dependencies for config validation in other modules.
	if c.Testing {
		return nil
	}

	// Check that a from email exists and that it is parseable
	// TODO: ensure it can be parsed
	if c.FromEmail == "" {
		return errors.New("invalid configuration: from email is required")
	}

	// At least one email mechanism must be enabled
	if !c.SMTP.Enabled() && !c.SendGrid.Enabled() {
		return errors.New("invalid configuration: specify either smtp or sendgrid configuration")
	}

	// Cannot specify both email mechanisms
	if c.SMTP.Enabled() && c.SendGrid.Enabled() {
		return errors.New("invalid configuration: cannot specify configuration for both smtp and sendgrid")
	}

	// Validate the SMTP configuration
	if c.SMTP.Enabled() {
		if err = c.SMTP.Validate(); err != nil {
			return err
		}
	}

	// Validate the SendGrid configuration
	if c.SendGrid.Enabled() {
		if err = c.SendGrid.Validate(); err != nil {
			return err
		}
	}

	return nil
}

func (c SMTPConfig) Enabled() bool {
	return c.Host != ""
}

func (c SMTPConfig) Validate() (err error) {
	// Do not validate if not enabled
	if !c.Enabled() {
		return nil
	}

	if c.Port == 0 {
		return errors.New("invalid configuration: smtp port is required")
	}

	if c.PoolSize < 1 {
		return errors.New("invalid configuration: smtp connections pool size must be greater than zero")
	}

	if c.UseCRAMMD5 {
		if c.Username == "" || c.Password == "" {
			return errors.New("invalid configuration: smtp cram-md5 requires username and password")
		}
	}

	return nil
}

func (c SMTPConfig) Pool() (*email.Pool, error) {
	return email.NewPool(c.Addr(), c.PoolSize, c.Auth())
}

func (c SMTPConfig) Auth() smtp.Auth {
	if c.UseCRAMMD5 {
		return smtp.CRAMMD5Auth(c.Username, c.Password)
	}
	return smtp.PlainAuth("", c.Username, c.Password, c.Host)
}

func (c SMTPConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

func (c SendGridConfig) Enabled() bool {
	return c.APIKey != ""
}

func (c SendGridConfig) Validate() (err error) {
	// Do not validate if not enabled
	if !c.Enabled() {
		return nil
	}
	return nil
}
