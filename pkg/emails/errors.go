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
