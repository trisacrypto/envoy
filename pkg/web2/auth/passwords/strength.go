package passwords

import "unicode"

// Returns an error if the password is not strong enough as well as a password strength
// score to compare the strengths of different passwords.
func Strength(password string) (strength uint8, err error) {
	// Password cannot be empty
	if password == "" {
		return 0, ErrPasswordEmpty
	}

	// Password must be at least 8 characters
	if len(password) < 8 {
		return 0, ErrPasswordTooShort
	}

	// Password must not start or end with whitespace
	if unicode.IsSpace(rune(password[0])) || unicode.IsSpace(rune(password[len(password)-1])) {
		return 0, ErrPasswordWhitespace
	}

	// Flags are used to compute strength scores the positions of the flags are:
	// numbers, uppercase, lowercase, special chars, very long
	var flags = []uint8{0, 0, 0, 0, 0}
	for _, c := range password {
		switch {
		case unicode.IsNumber(c):
			flags[0] = 1
		case unicode.IsUpper(c):
			flags[1] = 1
		case unicode.IsLower(c):
			flags[2] = 1
		case unicode.IsPunct(c) || unicode.IsSymbol(c):
			flags[3] = 1
		}
	}

	// Bonus points for a really long password
	if len(password) > 16 {
		flags[4] = 1
	}

	// Compute the total strength score
	for _, flag := range flags {
		strength = strength + flag
	}

	if strength < 3 {
		return strength, ErrPasswordStrength
	}

	return strength, nil
}
