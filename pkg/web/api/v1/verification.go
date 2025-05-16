package api

import (
	"strings"

	"github.com/trisacrypto/envoy/pkg/verification"
)

//===========================================================================
// URL Verification
//===========================================================================

// Allows the user to pass a verification token via the URL.
type URLVerification struct {
	Token string `json:"token,omitempty" url:"token,omitempty" form:"token"`
	token verification.VerificationToken
}

func (s *URLVerification) Validate() (err error) {
	s.Token = strings.TrimSpace(s.Token)
	if s.Token == "" {
		err = ValidationError(err, MissingField("token"))
	} else {
		var perr error
		if s.token, perr = verification.ParseVerification(s.Token); perr != nil {
			err = ValidationError(err, IncorrectField("token", perr.Error()))
		}
	}

	return err
}

// Returns the underlying verification token if it has already been parsed. It parses
// the token if not, but does not return the error (only) nil. Callers should ensure
// that Validate() is called first to ensure there will be no parse errors.
func (s *URLVerification) VerificationToken() verification.VerificationToken {
	if len(s.token) == 0 {
		var err error
		if s.token, err = verification.ParseVerification(s.Token); err != nil {
			return nil
		}
	}
	return s.token
}
