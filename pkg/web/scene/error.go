package scene

import "net/mail"

type Error struct {
	Scene
	Error        string
	SupportEmail string
	Support      string
}

type SunriseError struct {
	Scene
	Error           string
	SupportEmail    string
	Support         string
	ComplianceEmail string
	Compliance      string
}

// Return the simplified/flattened IVMS101 identity representation if an Envelope has
// been set as the APIData in the Scene.
func (s Scene) Error(err error) *Error {
	e := &Error{
		Scene:        s,
		SupportEmail: "",
		Support:      "",
	}

	if err != nil {
		e.Error = err.Error()
	}

	return e
}

func (s *Error) WithEmail(email string) *Error {
	if email == "" {
		return s
	}

	if addr, _ := mail.ParseAddress(email); addr != nil {
		s.SupportEmail = addr.Address
		s.Support = addr.Name

		if s.Support == "" {
			s.Support = s.SupportEmail
		}
	}

	return s
}

func (s Scene) SunriseError(err error) *SunriseError {
	e := &SunriseError{
		Scene:           s,
		SupportEmail:    "",
		Support:         "",
		ComplianceEmail: "",
		Compliance:      "",
	}

	if err != nil {
		e.Error = err.Error()
	}

	return e
}

func (s *SunriseError) WithEmail(support, compliance string) *SunriseError {
	if addr, _ := mail.ParseAddress(support); addr != nil {
		s.SupportEmail = addr.Address
		s.Support = addr.Name

		if s.Support == "" {
			s.Support = s.SupportEmail
		}
	}

	if addr, _ := mail.ParseAddress(compliance); addr != nil {
		s.ComplianceEmail = addr.Address
		s.Compliance = addr.Name

		if s.Compliance == "" {
			s.Compliance = s.ComplianceEmail
		}
	}

	return s
}
