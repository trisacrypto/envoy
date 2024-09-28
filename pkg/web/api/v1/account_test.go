package api_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/ulids"
	. "github.com/trisacrypto/envoy/pkg/web/api/v1"
	"github.com/trisacrypto/trisa/pkg/ivms101"
)

const naturalPersonEncoded = "CtwBCi8KFAoGTWlsbGVyEghDeXJpbCBFLhgEGhcKCMuIbcmqbMmZEgnLiHPJqnLJmWwYBBI4CAEiD0RlbGF3YXJlIEF2ZW51ZSoENDIwNFIFOTQxMTVaDVNhbiBGcmFuY2lzY29yAkNBggECVVMaEwoLNDExLTQxLTQxMTQQBxoCVVMiIm1pOWY0NkFCUmZKTHZTbmpBcThFVXFlZTloMTF2Q1BXTHAqMgoKMTk4OS0wNC0zMBIkQmFrZXJzZmllbGQsIFZpcmdpbmlhLCBVbml0ZWQgU3RhdGVzMgJVUw=="

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
			IVMSRecord: naturalPersonEncoded,
		}

		account.SetEncoding(&EncodingQuery{Format: "pb"})
		require.NoError(t, account.Validate(true), "expected account to be valid with IVMS101 record on create")

		account.ID = ulids.New()
		require.NoError(t, account.Validate(false), "expected account to be valid with IVMS101 record on update")
	})

	t.Run("InvalidIVMSRecord", func(t *testing.T) {
		record, err := encodeIVMSFixture("testdata/invalid_person.json", "base64", "pb")
		require.NoError(t, err, "could not load testdata/invalid_person.json fixture")

		account := &Account{
			CustomerID: "6661182",
			FirstName:  "James",
			LastName:   "Bond",
			IVMSRecord: record,
		}

		account.SetEncoding(&EncodingQuery{Format: "pb"})
		require.Error(t, account.Validate(true), "expected account to be invalid with invalid IVMS101 record on create")

		account.ID = ulids.New()
		require.Error(t, account.Validate(false), "expected account to be invalid valid with invalid IVMS101 record on update")
	})

	t.Run("InvalidCryptoAddress", func(t *testing.T) {
		account := &Account{
			CustomerID: "6661182",
			FirstName:  "James",
			LastName:   "Bond",
			CryptoAddresses: []*CryptoAddress{
				{
					CryptoAddress: "n2S7RuM2Y6PXMyaM2BDDTojpgsgqHxmDPx",
				},
			},
		}

		require.EqualError(t, account.Validate(true), "missing crypto_addresses[0].network: this field is required", "expected crypto address to be valid on create")
		require.EqualError(t, account.Validate(false), "missing crypto_addresses[0].network: this field is required", "expected crypto address to be valid update")
	})

	t.Run("StackOfErrors", func(t *testing.T) {
		record, err := encodeIVMSFixture("testdata/invalid_person.json", "base64", "pb")
		require.NoError(t, err, "could not load testdata/invalid_person.json fixture")

		account := &Account{
			ID:            ulids.New(),
			IVMSRecord:    record,
			TravelAddress: "taLg4sBFp3cWhB9wN7qqPwDzq32bWwhibhFvADbiYp5fMR7asAxbqNrPuUyT4VzZa98oPk6dHcdKhov9jiraNrcZ7yQdikXcwbv",
			CryptoAddresses: []*CryptoAddress{
				{
					ID:            ulids.New(),
					TravelAddress: "taLg4sBFp3cWhB9wN7qqPwDzq32bWwhibhFvADbiYp5fMR7asAxbqNrPuUyT4VzZa98oPk6dHcdKhov9jiraNrcZ7yQdikXcwbv",
				},
				{
					ID:            ulids.New(),
					Network:       "FOOOOO",
					TravelAddress: "taLg4sBFp3cWhB9wN7qqPwDzq32bWwhibhFvADbiYp5fMR7asAxbqNrPuUyT4VzZa98oPk6dHcdKhov9jiraNrcZ7yQdikXcwbv",
				},
				{
					ID:            ulids.New(),
					TravelAddress: "taLg4sBFp3cWhB9wN7qqPwDzq32bWwhibhFvADbiYp5fMR7asAxbqNrPuUyT4VzZa98oPk6dHcdKhov9jiraNrcZ7yQdikXcwbv",
				},
				{
					ID:            ulids.New(),
					TravelAddress: "taLg4sBFp3cWhB9wN7qqPwDzq32bWwhibhFvADbiYp5fMR7asAxbqNrPuUyT4VzZa98oPk6dHcdKhov9jiraNrcZ7yQdikXcwbv",
				},
			},
		}

		err = account.Validate(true)
		require.Error(t, err, "expected account to be completely invalid")

		verr, ok := err.(ValidationErrors)
		require.True(t, ok, "expected error to be ValidationErrors")
		require.Len(t, verr, 21)

	})
}

func encodeIVMSFixture(path, encoding, format string) (out string, err error) {
	person := &ivms101.Person{}
	if err = loadFixture(path, person); err != nil {
		return "", err
	}

	encoder := &EncodingQuery{Format: format, Encoding: encoding}
	return encoder.Marshal(person)
}
