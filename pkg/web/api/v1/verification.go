package api

import (
	"strings"

	"github.com/google/uuid"
	"go.rtnl.ai/ulid"
	"go.rtnl.ai/x/vero"
)

//===========================================================================
// URL Verification
//===========================================================================

// Allows the user to pass a verification token via the URL.
type URLVerification struct {
	Token string `json:"token,omitempty" url:"token,omitempty" form:"token"`
	token vero.VerificationToken
}

func (s *URLVerification) Validate() (err error) {
	s.Token = strings.TrimSpace(s.Token)
	if s.Token == "" {
		return ValidationError(err, MissingField("token"))
	}

	var perr error
	if s.token, perr = vero.Parse(s.Token); perr != nil {
		return ValidationError(err, IncorrectField("token", perr.Error()))
	}

	// Check that the record ID is either a UUID or a ULID
	var (
		uuidID uuid.UUID
		ulidID ulid.ULID
	)

	uuiderr := uuidID.UnmarshalBinary(s.token.RecordID())
	uliderr := ulidID.UnmarshalBinary(s.token.RecordID())

	if uuiderr != nil && uliderr != nil {
		err = ValidationError(err, IncorrectField("token", "record ID must be a valid UUID or ULID"))
	}

	return err
}

// Returns the underlying verification token if it has already been parsed. It parses
// the token if not, but does not return the error (only) nil. Callers should ensure
// that Validate() is called first to ensure there will be no parse errors.
func (s *URLVerification) VerificationToken() vero.VerificationToken {
	if len(s.token) == 0 {
		var err error
		if s.token, err = vero.Parse(s.Token); err != nil {
			return nil
		}
	}
	return s.token
}

// Parses the underlying record ID as a UUID. Does not return an error if the record
// ID is not a valid UUID, but will return uuid.Nil. Callers should ensure that
// Validate() is called first to ensure there will be no parse errors.
func (s *URLVerification) RecordUUID() (recordID uuid.UUID) {
	var err error
	if len(s.token) == 0 {
		if s.token, err = vero.Parse(s.Token); err != nil {
			return uuid.Nil
		}
	}

	if err = recordID.UnmarshalBinary(s.token.RecordID()); err != nil {
		return uuid.Nil
	}

	return recordID
}

// Parses the underlying record ID as a ULID. Does not return an error if the record
// ID is not a valid UUID, but will return uuid.Nil. Callers should ensure that
// Validate() is called first to ensure there will be no parse errors.
func (s *URLVerification) RecordULID() (recordID ulid.ULID) {
	var err error
	if len(s.token) == 0 {
		if s.token, err = vero.Parse(s.Token); err != nil {
			return ulid.Zero
		}
	}

	if err = recordID.UnmarshalBinary(s.token.RecordID()); err != nil {
		return ulid.Zero
	}

	return recordID
}
