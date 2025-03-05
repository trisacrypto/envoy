package api

import "github.com/trisacrypto/envoy/pkg/web/auth/passwords"

type ProfilePassword struct {
	Current  string `json:"current,omitempty"`
	Password string `json:"password,omitempty"`
	Confirm  string `json:"confirm,omitempty"`
}

// Validate ensures that the password change request is valid and meets the minimum
// password requirements. Note that this method does not check the current password
// against the user's actual password and must be performed by any handler that has
// access to the database store.
func (p *ProfilePassword) Validate() (err error) {
	if p.Current == "" {
		err = ValidationError(err, MissingField("current"))
	}

	if p.Password == "" {
		err = ValidationError(err, MissingField("password"))
	} else if len(p.Password) < 8 {
		err = ValidationError(err, IncorrectField("password", "must be at least 8 characters long"))
	} else {
		// Validate password strength
		if _, verr := passwords.Strength(p.Password); verr != nil {
			err = ValidationError(err, IncorrectField("password", verr.Error()))
		}
	}

	if p.Confirm == "" {
		err = ValidationError(err, MissingField("confirm"))
	} else if p.Password != p.Confirm {
		err = ValidationError(err, IncorrectField("confirm", "does not match the password"))
	}

	return err
}
