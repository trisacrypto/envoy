package api_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/web/api/v1"
)

func TestUserPasswordStrength(t *testing.T) {
	t.Run("Invalid", func(t *testing.T) {
		testCases := []struct {
			password string
			err      string
		}{
			{"", "missing password: this field is required"},
			{"short", "invalid field password: password must be at least 8 characters"},
			{"   spacy   ", "invalid field password: password must not start or end with whitespace"},
			{"simplepassword", "invalid field password: password must contain uppercase letters, lowercase letters, numbers, and special characters"},
			{"simple3password", "invalid field password: password must contain uppercase letters, lowercase letters, numbers, and special characters"},
			{"SimplepassworD", "invalid field password: password must contain uppercase letters, lowercase letters, numbers, and special characters"},
			{"s!mplepassw?rd", "invalid field password: password must contain uppercase letters, lowercase letters, numbers, and special characters"},
		}

		for i, tc := range testCases {
			req := api.UserPassword{Password: tc.password}
			require.EqualError(t, req.Validate(), tc.err, "test case %d failed", i)
		}
	})

	t.Run("Valid", func(t *testing.T) {
		testCases := []string{
			"lowernum$symb0ls",
			"UpperNum31234122",
			"lowUP$sYUMbols",
			"UPPER0NLYNUM$MB0L",
		}

		for i, tc := range testCases {
			req := api.UserPassword{Password: tc}
			require.NoError(t, req.Validate(), "test case %d failed", i)
		}
	})
}
