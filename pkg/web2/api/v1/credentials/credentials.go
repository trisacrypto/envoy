package credentials

import "errors"

type Credentials interface {
	AccessToken() (string, error)
}

var ErrInvalidCredentials = errors.New("missing, invalid or expired credentials")

type Token string

func (t Token) AccessToken() (string, error) {
	if string(t) == "" {
		return "", ErrInvalidCredentials
	}
	return string(t), nil
}
