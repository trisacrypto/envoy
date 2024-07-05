package passwords_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	. "github.com/trisacrypto/envoy/pkg/web/auth/passwords"
)

func TestStrength(t *testing.T) {
	testCases := []struct {
		password         string
		expectedStrength uint8
		expectedErr      error
	}{
		{"", 0, ErrPasswordEmpty},                              // cannot be empty
		{"a", 0, ErrPasswordTooShort},                          // cannot be a single character
		{"foo", 0, ErrPasswordTooShort},                        // cannot be too short
		{"a3O1#Db", 0, ErrPasswordTooShort},                    // even complex, cannot be too short
		{"  a301#Db", 0, ErrPasswordWhitespace},                // can't start with space
		{"a301#Db ", 0, ErrPasswordWhitespace},                 // can't end with space
		{"onlylowercase", 1, ErrPasswordStrength},              // only lowercase
		{"ONLYUPPERCASE", 1, ErrPasswordStrength},              // only uppercase
		{"1234567890", 1, ErrPasswordStrength},                 // only numbers
		{"#!@%@$^!@#$!", 1, ErrPasswordStrength},               // only symbols
		{"abcdef1234", 2, ErrPasswordStrength},                 // lowercase and numbers
		{"abcdefABCDE", 2, ErrPasswordStrength},                // lowercase and uppercase
		{"abcdef@!@#$", 2, ErrPasswordStrength},                // lowercase and symbols
		{"ABCDDEF12345", 2, ErrPasswordStrength},               // uppercase and numbers
		{"ABCDDEF@!@#$", 2, ErrPasswordStrength},               // uppercase and symbols
		{"123456@!@#$", 2, ErrPasswordStrength},                // numbers and symbols
		{"thisisanextralongpassword", 2, ErrPasswordStrength},  // extra long only lowercase
		{"THISISANEXTRALONGPASSWORD", 2, ErrPasswordStrength},  // extra long only uppercase
		{"1235512031234123412341234", 2, ErrPasswordStrength},  // extra long only numbers
		{"!@#$!$#%!@^$%&#%!$%!@@#$^%", 2, ErrPasswordStrength}, // extra long only symbols
		{"s3cr4tMissION", 3, nil},                              // weak but valid password: lc, up, num
		{"s#cr!tMissION", 3, nil},                              // weak but valid password: lc, up, sym
		{"secretMissIONwithlength", 3, nil},                    // weak but valid password: lc, up, long
		{"secret4iss213withlength", 3, nil},                    // weak but valid password: lc, num, long
		{"Sup3rS3@ret", 4, nil},                                // valid password: lc, uc, num, sym
		{"Sup3rS3@ret!r0nM4n", 5, nil},                         // valid password: lc, uc, num, sym, long
		{"This is a valid 4ever password!", 5, nil},            // passwords can contain spaces in the middle
	}

	for i, tc := range testCases {
		strength, err := Strength(tc.password)
		require.Equal(t, tc.expectedStrength, strength, "expected strength did not match actual strength for test case %d", i)

		if tc.expectedErr == nil {
			require.NoError(t, err, "unexpected error on test case %d", i)
		} else {
			require.ErrorIs(t, err, tc.expectedErr, "error mismatch on test case %d", i)
		}
	}
}
