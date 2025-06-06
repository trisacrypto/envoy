package models_test

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/mock"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/trisa/pkg/ivms101"
	"go.rtnl.ai/ulid"
)

//==========================================================================
// Tests
//==========================================================================

func TestAccountParams(t *testing.T) {
	// setup a model
	theModel := getSampleAccount(true, true, true)

	// create the model public field name comparison list
	fields := GetPublicFieldNames(*theModel)

	// create the `Params()` comparison list
	// Exception 1) replace "ivms101" as "ivmsrecord"
	exceptions := map[string]string{ConvertNameForComparison("ivms101"): ConvertNameForComparison("IVMSRecord")}
	params := GetParamsNames(theModel, exceptions)

	// test
	require.ElementsMatch(t, fields, params, "the model's public fields and Params() lists should have the same names")
}

func TestAccountCryptoAddresses(t *testing.T) {
	// test 1: has addresses
	addresses, err := getSampleAccount(true, true, true).CryptoAddresses()
	require.NotNil(t, addresses, "addresses should not be nil")
	require.Nil(t, err, "error should be nil")

	//test 2: no addresses
	addresses, err = getSampleAccount(false, false, false).CryptoAddresses()
	require.Nil(t, addresses, "addresses should be nil")
	require.Error(t, err, "error should not be nil")
	require.Equal(t, errors.ErrMissingAssociation, err, "error should be ErrMissingAssociation")

}

func TestAccountNumAddresses(t *testing.T) {
	// test 1: has addresses
	number := getSampleAccount(true, true, true).NumAddresses()
	require.Equal(t, int64(2), number, fmt.Sprintf("should have 2 addresses: %d", number))

	//test 2: no addresses
	number = getSampleAccount(false, false, false).NumAddresses()
	require.Equal(t, int64(0), number, fmt.Sprintf("should have 0 addresses: %d", number))

}

func TestAccountHasIVMSRecord(t *testing.T) {
	// test 1: has IVMSRecord
	ok := getSampleAccount(true, true, true).HasIVMSRecord()
	require.True(t, ok, "should have an IVMSRecord")

	//test 2: no IVMSRecord
	ok = getSampleAccount(false, false, false).HasIVMSRecord()
	require.False(t, ok, "should not have an IVMSRecord")

}

func TestAccountScan(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// setup
		data := []any{
			ulid.MakeSecure().String(),    // ID
			"CustomerID",                  // CustomerID
			"FirstName",                   // FirstName
			"LastName",                    // LastName
			"TravelAddress",               // TravelAddress
			nil,                           // IVMSRecord
			time.Now(),                    // Created
			time.Now().Add(1 * time.Hour), // Modified
		}
		mockScanner := &mock.MockScanner{}
		mockScanner.SetData(data)

		// test
		account := &models.Account{}
		err := account.Scan(mockScanner)
		require.NoError(t, err, "expected no errors from the scanner")
		mockScanner.AssertScanned(t, len(data)-1) // FIXME: 5 are scanned, but we expect all but IVMSRecord
	})
}

//==========================================================================
// Helpers
//==========================================================================

// Returns a sample Account. Can add the IVMS101 and CryptoAddresses and include
// or exclude `NullType` values.
func getSampleAccount(includeNulls bool, addIvms101 bool, addCrypto bool) (account *models.Account) {
	id := ulid.MakeSecure()
	timeNow := time.Now()

	account = &models.Account{
		Model: models.Model{
			ID:       id,
			Created:  timeNow,
			Modified: timeNow},
		CustomerID:    sql.NullString{},
		FirstName:     sql.NullString{},
		LastName:      sql.NullString{},
		TravelAddress: sql.NullString{},
		IVMSRecord:    nil,
	}

	if includeNulls {
		account.CustomerID = sql.NullString{String: "CustomerID", Valid: true}
		account.FirstName = sql.NullString{String: "FirstName", Valid: true}
		account.LastName = sql.NullString{String: "LastName", Valid: true}
		account.TravelAddress = sql.NullString{String: "TravelAddress", Valid: true}
	}

	if addIvms101 {
		account.IVMSRecord = &ivms101.Person{
			Person: &ivms101.Person_NaturalPerson{
				NaturalPerson: &ivms101.NaturalPerson{
					Name: &ivms101.NaturalPersonName{
						NameIdentifiers: []*ivms101.NaturalPersonNameId{
							{
								PrimaryIdentifier:   "FirstName",
								SecondaryIdentifier: "LastName",
								NameIdentifierType:  ivms101.NaturalPersonNameTypeCode_NATURAL_PERSON_NAME_TYPE_CODE_LEGL,
							},
						},
					},
				},
			},
		}
	}

	if addCrypto {
		addresses := []*models.CryptoAddress{
			{
				AccountID:     ulid.MakeSecure(),
				CryptoAddress: "CryptoAddress",
				Network:       "BTC",
			},
			{
				AccountID:     ulid.MakeSecure(),
				CryptoAddress: "CryptoAddress",
				Network:       "BTC",
			},
		}
		account.SetCryptoAddresses(addresses)
	}

	return account
}
