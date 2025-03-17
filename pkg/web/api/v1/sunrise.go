package api

import (
	"strings"

	"github.com/trisacrypto/envoy/pkg/sunrise"
)

//===========================================================================
// Sunrise Verification
//===========================================================================

// Allows the user to pass a verification token via the URL.
type SunriseVerification struct {
	Token string `json:"token,omitempty" url:"token,omitempty" form:"token"`
	token sunrise.VerificationToken
}

func (s *SunriseVerification) Validate() (err error) {
	s.Token = strings.TrimSpace(s.Token)
	if s.Token == "" {
		err = ValidationError(err, MissingField("token"))
	} else {
		var perr error
		if s.token, perr = sunrise.ParseVerification(s.Token); perr != nil {
			err = ValidationError(err, IncorrectField("token", perr.Error()))
		}
	}

	return err
}

// Returns the underlying verification token if it has already been parsed. It parses
// the token if not, but does not return the error (only) nil. Callers should ensure
// that Validate() is called first to ensure there will be no parse errors.
func (s *SunriseVerification) VerificationToken() sunrise.VerificationToken {
	if len(s.token) == 0 {
		var err error
		if s.token, err = sunrise.ParseVerification(s.Token); err != nil {
			return nil
		}
	}
	return s.token
}
