package api_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/web/api/v1"
	"github.com/trisacrypto/trisa/pkg/ivms101"
)

func TestParseNationalIdentifierType(t *testing.T) {
	testCases := []struct {
		input    string
		expected ivms101.NationalIdentifierTypeCode
	}{
		// Standard list of national identifiers
		{"MISC", ivms101.NationalIdentifierMISC},
		{"ARNU", ivms101.NationalIdentifierARNU},
		{"CCPT", ivms101.NationalIdentifierCCPT},
		{"RAID", ivms101.NationalIdentifierRAID}, // only for legal persons
		{"DRLC", ivms101.NationalIdentifierDRLC},
		{"FIIN", ivms101.NationalIdentifierFIIN},
		{"TXID", ivms101.NationalIdentifierTXID}, // only for legal persons
		{"SOCS", ivms101.NationalIdentifierSOCS},
		{"IDCD", ivms101.NationalIdentifierIDCD},
		{"LEIX", ivms101.NationalIdentifierLEIX}, // only for legal persons

		// Bad national identifiers return MISC
		{"BADTC", ivms101.NationalIdentifierMISC},
		{"", ivms101.NationalIdentifierMISC},

		// Case insensitive parsing
		{"misc", ivms101.NationalIdentifierMISC},
		{"arnu", ivms101.NationalIdentifierARNU},
		{"ccpt", ivms101.NationalIdentifierCCPT},
		{"raid", ivms101.NationalIdentifierRAID}, // only for legal persons
		{"drlc", ivms101.NationalIdentifierDRLC},
		{"fiin", ivms101.NationalIdentifierFIIN},
		{"txid", ivms101.NationalIdentifierTXID}, // only for legal persons
		{"socs", ivms101.NationalIdentifierSOCS},
		{"idcd", ivms101.NationalIdentifierIDCD},
		{"leix", ivms101.NationalIdentifierLEIX}, // only for legal persons

		// Whitespace trimming
		{"  \n    CCPT   \t", ivms101.NationalIdentifierCCPT},

		// Allow Prefix
		{"NATIONAL_IDENTIFIER_TYPE_CODE_MISC", ivms101.NationalIdentifierMISC},
		{"NATIONAL_IDENTIFIER_TYPE_CODE_ARNU", ivms101.NationalIdentifierARNU},
		{"NATIONAL_IDENTIFIER_TYPE_CODE_CCPT", ivms101.NationalIdentifierCCPT},
		{"NATIONAL_IDENTIFIER_TYPE_CODE_RAID", ivms101.NationalIdentifierRAID}, // only for legal persons
		{"NATIONAL_IDENTIFIER_TYPE_CODE_DRLC", ivms101.NationalIdentifierDRLC},
		{"NATIONAL_IDENTIFIER_TYPE_CODE_FIIN", ivms101.NationalIdentifierFIIN},
		{"NATIONAL_IDENTIFIER_TYPE_CODE_TXID", ivms101.NationalIdentifierTXID}, // only for legal persons
		{"NATIONAL_IDENTIFIER_TYPE_CODE_SOCS", ivms101.NationalIdentifierSOCS},
		{"NATIONAL_IDENTIFIER_TYPE_CODE_IDCD", ivms101.NationalIdentifierIDCD},
		{"NATIONAL_IDENTIFIER_TYPE_CODE_LEIX", ivms101.NationalIdentifierLEIX}, // only for legal persons
	}

	for i, tc := range testCases {
		identification := &api.Identification{TypeCode: tc.input}
		require.Equal(t, tc.expected, identification.NationalIdentifierTypeCode(), "test case %d failed", i)
	}
}
