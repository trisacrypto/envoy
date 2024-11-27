package api_test

import (
	"testing"

	"github.com/oklog/ulid/v2"
	. "github.com/trisacrypto/envoy/pkg/web/api/v1"

	"github.com/stretchr/testify/require"
)

const legalPersonEncoded = "CowBCiUKIUJpdENoYWluWCBGaW5hbmNpYWwgU2VydmljZXMsIExMQxABCg0KCUJpdENoYWluWBACGkAKPGLJqnTKp2XJqm7Jm2tzIGZhyarLiG7Dpm7Kg-G1imwgy4hzyZzLkHbJqnPJqnosIMmbbC3Jm2wtc2nLkBABGhIKDmLJqnTKp2XJqm7Jm2tzEAISLggCIg1FbGsgUmQgTGl0dGxlKgMzNzJSBTg1NzEwWgZUdXNjb25yAkFaggECVVMaJDRjNDIxNzk0LTNiMjUtNDg3NS1iNTFiLTYzN2RlMzcwNGY1NCIYChQ2NDhGMDAwRDlDMEFOV1JBTDMxNBAJKgJVUw=="

func TestCounterpartyValidate(t *testing.T) {
	t.Run("Source", func(t *testing.T) {
		counterparty := &Counterparty{
			Source:     "user",
			Protocol:   "trp",
			CommonName: "trisa.example.com",
			Endpoint:   "https://trisa.example.com",
			Name:       "Test VASP",
			Country:    "FR",
		}

		require.EqualError(t, counterparty.Validate(), "read-only field source: this field cannot be written by the user")
	})

	t.Run("DirectoryID", func(t *testing.T) {
		counterparty := &Counterparty{
			DirectoryID: "4120afd5-ed01-401e-be91-74359bb4c98f",
			Protocol:    "trp",
			CommonName:  "trisa.example.com",
			Endpoint:    "https://trisa.example.com",
			Name:        "Test VASP",
			Country:     "FR",
		}

		require.EqualError(t, counterparty.Validate(), "read-only field directory_id: this field cannot be written by the user")
	})

	t.Run("RegisteredDirectory", func(t *testing.T) {
		counterparty := &Counterparty{
			RegisteredDirectory: "trisatest.net",
			Protocol:            "trp",
			CommonName:          "trisa.example.com",
			Endpoint:            "https://trisa.example.com",
			Name:                "Test VASP",
			Country:             "FR",
		}

		require.EqualError(t, counterparty.Validate(), "read-only field registered_directory: this field cannot be written by the user")
	})

	t.Run("Protocol", func(t *testing.T) {
		t.Run("Missing", func(t *testing.T) {
			counterparty := &Counterparty{
				CommonName: "trisa.example.com",
				Endpoint:   "https://trisa.example.com",
				Name:       "Test VASP",
				Country:    "FR",
			}

			require.EqualError(t, counterparty.Validate(), "missing protocol: this field is required")
		})

		t.Run("Invalid", func(t *testing.T) {
			counterparty := &Counterparty{
				Protocol:   "foo",
				CommonName: "trisa.example.com",
				Endpoint:   "https://trisa.example.com",
				Name:       "Test VASP",
				Country:    "FR",
			}

			require.EqualError(t, counterparty.Validate(), "invalid field protocol: protocol must be either trisa or trp")
		})
	})

	t.Run("CommonName", func(t *testing.T) {
		t.Run("Missing", func(t *testing.T) {
			counterparty := &Counterparty{
				Protocol: "trp",
				Name:     "Test VASP",
				Country:  "FR",
			}

			require.EqualError(t, counterparty.Validate(), "2 validation errors occurred:\n  missing common_name: this field is required\n  missing endpoint: this field is required")
		})

		t.Run("FromEndpoint", func(t *testing.T) {
			counterparty := &Counterparty{
				Protocol: "trp",
				Endpoint: "https://example.com",
				Name:     "Test VASP",
				Country:  "FR",
			}

			require.NoError(t, counterparty.Validate(), "expected common name to not be required when endpoint is set")
			require.Equal(t, "example.com", counterparty.CommonName, "expected common name to be set by endpoint")
		})
	})

	t.Run("Endpoint", func(t *testing.T) {
		counterparty := &Counterparty{
			Protocol:   "trp",
			CommonName: "trisa.example.com",
			Name:       "Test VASP",
			Country:    "FR",
		}

		require.EqualError(t, counterparty.Validate(), "missing endpoint: this field is required")
	})

	t.Run("Name", func(t *testing.T) {
		counterparty := &Counterparty{
			Protocol:   "trp",
			CommonName: "trisa.example.com",
			Endpoint:   "https://trisa.example.com",
			Country:    "FR",
		}

		require.EqualError(t, counterparty.Validate(), "missing name: this field is required")
	})

	t.Run("Country", func(t *testing.T) {
		t.Run("Missing", func(t *testing.T) {
			counterparty := &Counterparty{
				Protocol:   "trp",
				CommonName: "trisa.example.com",
				Endpoint:   "https://trisa.example.com",
				Name:       "Test VASP",
			}

			require.EqualError(t, counterparty.Validate(), "missing country: this field is required")
		})

		t.Run("Invalid", func(t *testing.T) {
			counterparty := &Counterparty{
				Protocol:   "trp",
				CommonName: "trisa.example.com",
				Endpoint:   "https://trisa.example.com",
				Name:       "Test VASP",
				Country:    "France",
			}

			require.EqualError(t, counterparty.Validate(), "invalid field country: country must be the two character (alpha-2) country code")
		})
	})

	t.Run("NoIVMSRecord", func(t *testing.T) {
		counterparty := &Counterparty{
			Protocol:   "trp",
			CommonName: "trisa.example.com",
			Endpoint:   "https://trisa.example.com",
			Name:       "Test VASP",
			Country:    "FR",
			IVMSRecord: "",
		}

		require.NoError(t, counterparty.Validate(), "expected counterparty record to be valid without IVMS101")
	})

	t.Run("IVMSRecord", func(t *testing.T) {
		counterparty := &Counterparty{
			Protocol:   "trp",
			CommonName: "trisa.example.com",
			Endpoint:   "https://trisa.example.com",
			Name:       "Test VASP",
			Country:    "FR",
			IVMSRecord: legalPersonEncoded,
		}

		counterparty.SetEncoding(&EncodingQuery{Format: "pb"})
		require.NoError(t, counterparty.Validate(), "expected counterparty record to be valid with IVMS101")
	})

	t.Run("StackOfErrors", func(t *testing.T) {
		record, err := encodeIVMSFixture("testdata/invalid_person.json", "base64", "pb")
		require.NoError(t, err, "could not load testdata/invalid_person.json fixture")

		counterparty := &Counterparty{
			Source:              "gds",
			DirectoryID:         "123",
			RegisteredDirectory: "trisatest.dev",
			Protocol:            "foo",
			Country:             "France",
			IVMSRecord:          record,
		}

		err = counterparty.Validate()
		require.Error(t, err, "expected counterparty to be completely invalid")

		verr, ok := err.(ValidationErrors)
		require.True(t, ok, "expected error to be ValidationErrors")
		require.Len(t, verr, 8)

	})
}

func TestContactValidate(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		testCases := []struct {
			contact *Contact
			create  bool
		}{
			{
				&Contact{Email: "bdenison@example.com"},
				true,
			},
			{
				&Contact{Name: "Barry Denison", Email: "bdenison@example.co.uk", Role: "Compliance Officer"},
				true,
			},
			{
				&Contact{ID: ulid.MustParse("01JDQCF64H7F3M408V3ADFWB9R"), Name: "Barry Denison", Email: "barry.denison@example.co.uk", Role: "Compliance Officer"},
				false,
			},
		}

		for i, tc := range testCases {
			err := tc.contact.Validate(tc.create)
			require.NoError(t, err, "test case %d failed", i)
		}
	})

	t.Run("Invalid", func(t *testing.T) {
		testCases := []struct {
			contact *Contact
			create  bool
			err     string
		}{
			{
				&Contact{ID: ulid.MustParse("01JDQCF64H7F3M408V3ADFWB9R"), Email: "bdenison@example.com"},
				true,
				"read-only field id: this field cannot be written by the user",
			},
			{
				&Contact{ID: ulid.MustParse("01JDQCF64H7F3M408V3ADFWB9R"), Email: ""},
				false,
				"missing email: this field is required",
			},
			{
				&Contact{Email: "bob"},
				true,
				"invalid field email: not an email address",
			},
			{
				&Contact{Email: "Bob Dylan <bob@foo.com>"},
				true,
				"invalid field email: not an email address",
			},
		}

		for i, tc := range testCases {
			err := tc.contact.Validate(tc.create)
			require.EqualError(t, err, tc.err, "test case %d failed", i)
		}
	})
}
