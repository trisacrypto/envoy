package api_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/web/api/v1"
)

func TestProfilePasswordValidate(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		pw := &api.ProfilePassword{
			Current:  "supersecretsquirrel",
			Password: "h4ck3rPr@@f!",
			Confirm:  "h4ck3rPr@@f!",
		}
		require.NoError(t, pw.Validate())
	})

	t.Run("Invalid", func(t *testing.T) {
		tests := []*api.ProfilePassword{
			{Password: "short", Confirm: "short", Current: "supersecretsquirrel"},
			{Password: "h4ck3rPr@@f!", Confirm: "h4ck3rPr@@f!", Current: ""},
			{Password: "", Confirm: "h4ck3rPr@@f!", Current: "supersecretsquirrel"},
			{Password: "h4ck3rPr@@f!", Confirm: "", Current: "supersecretsquirrel"},
			{Password: "", Confirm: "", Current: "supersecretsquirrel"},
			{Password: "h4ck3rPr@@f!", Confirm: "hAck3rPr@@f!", Current: "supersecretsquirrel"},
			{Password: "notsecureenough", Confirm: "notsecureenough", Current: "supersecretsquirrel"},
		}

		for i, tc := range tests {
			require.Error(t, tc.Validate(), "test case %d failed", i)
		}
	})
}
