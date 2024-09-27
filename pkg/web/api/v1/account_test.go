package api_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/ulids"
	. "github.com/trisacrypto/envoy/pkg/web/api/v1"
)

func TestAccountValidate(t *testing.T) {
	t.Run("ID", func(t *testing.T) {
		account := &Account{
			ID:         ulids.New(),
			CustomerID: "6661182",
			FirstName:  "James",
			LastName:   "Bond",
		}

		require.EqualError(t, account.Validate(true), "read-only field id: this field cannot be written by the user", "expected error on create validate with ID")
		require.NoError(t, account.Validate(false), "", "expected no error on update validate with ID")
	})

	t.Run("LastName", func(t *testing.T) {
		account := &Account{
			CustomerID: "6661182",
			FirstName:  "James",
		}

		require.EqualError(t, account.Validate(true), "missing last_name: this field is required", "expected last name to be required on create")
		require.EqualError(t, account.Validate(false), "missing last_name: this field is required", "expected last name to be required on update")
	})

	t.Run("TravelAddress", func(t *testing.T) {
		account := &Account{
			CustomerID:    "6661182",
			FirstName:     "James",
			LastName:      "Bond",
			TravelAddress: "taLg4sBFp3cWhB9wN7qqPwDzq32bWwhibhFvADbiYp5fMR7asAxbqNrPuUyT4VzZa98oPk6dHcdKhov9jiraNrcZ7yQdikXcwbv",
		}

		require.EqualError(t, account.Validate(true), "read-only field travel_address: this field cannot be written by the user", "expected travel address to be read-only on create")
		require.EqualError(t, account.Validate(false), "read-only field travel_address: this field cannot be written by the user", "expected travel address to be read-only update")
	})

	t.Run("NoIVMSRecord", func(t *testing.T) {
		account := &Account{
			CustomerID: "6661182",
			FirstName:  "James",
			LastName:   "Bond",
			IVMSRecord: "",
		}

		require.NoError(t, account.Validate(true), "expected account to be valid with no IVMS101 record on create")

		account.ID = ulids.New()
		require.NoError(t, account.Validate(false), "expected account to be valid with no IVMS101 record on update")
	})

	t.Run("IVMSRecord", func(t *testing.T) {
		account := &Account{
			CustomerID: "6661182",
			FirstName:  "James",
			LastName:   "Bond",
			IVMSRecord: "CjwKEQoPCgRCb25kEgVKYW1lcxgCKiMKCjE5MjAtMTEtMTESFVdhdHRlbnNjaGVpZCwgR2VybWFueTICR0I=",
		}

		account.SetEncoding(&EncodingQuery{Format: "pb"})
		require.NoError(t, account.Validate(true), "expected account to be valid with IVMS101 record on create")

		account.ID = ulids.New()
		require.NoError(t, account.Validate(false), "expected account to be valid with IVMS101 record on update")
	})
}
