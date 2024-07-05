package passwords

import "errors"

// Password Strength Errors
var (
	ErrPasswordEmpty      = errors.New("password must not be an empty string")
	ErrPasswordTooShort   = errors.New("password must be at least 8 characters")
	ErrPasswordWhitespace = errors.New("password must not start or end with whitespace")
	ErrPasswordStrength   = errors.New("password must contain uppercase letters, lowercase letters, numbers, and special characters")
)
