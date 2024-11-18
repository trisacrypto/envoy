package emails

import "errors"

var (
	ErrMissingSubject   = errors.New("missing email subject")
	ErrMissingSender    = errors.New("missing email sender")
	ErrMissingRecipient = errors.New("missing email recipient(s)")
	ErrMissingTemplate  = errors.New("missing email template name")
	ErrIncorrectEmail   = errors.New("could not parse email address")
	ErrNotInitialized   = errors.New("email sending method has not been configured")
)

var (
	ErrConfigMissingSender = errors.New("invalid configuration: sender email is required")
	ErrConfigInvalidSender = errors.New("invalid configuration: could not parse sender email address")
	ErrConfigConflict      = errors.New("invalid configuration: cannot specify configuration for both smtp and sendgrid")
	ErrConfigMissingPort   = errors.New("invalid configuration: smtp port is required")
	ErrConfigPoolSize      = errors.New("invalid configuration: smtp connections pool size must be greater than zero")
	ErrConfigCRAMMD5Auth   = errors.New("invalid configuration: smtp cram-md5 requires username and password")
)
