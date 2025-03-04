package scene

import "net/mail"

type Error struct {
	Scene
	Error        string
	SupportEmail string
	Support      string
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

	addr, _ := mail.ParseAddress(email)
	s.SupportEmail = addr.Address
	s.Support = addr.Name

	if s.Support == "" {
		s.Support = s.SupportEmail
	}

	return s
}
