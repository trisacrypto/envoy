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
			nil,                           // IVMSRecord (will not scan)
			time.Now(),                    // Created
			time.Now().Add(1 * time.Hour), // Modified
		}
		mockScanner := &mock.MockScanner{}
		mockScanner.SetData(data)

		// test
		model := &models.Account{}
		err := model.Scan(mockScanner)
		require.NoError(t, err, "expected no errors from the scanner")
		mockScanner.AssertScanned(t, len(data)-1) // IVMSRecord will not scan

		// make sure scanned data matches the fields they were supposed to scan into

		// make sure scanned data matches the fields they were supposed to scan into
		require.Equal(t, data[0], model.ID.String(), "expected field ID to match data[0]")
		require.Equal(t, data[1], model.CustomerID.String, "expected field CustomerID to match data[1]")
		require.Equal(t, data[2], model.FirstName.String, "expected field FirstName to match data[2]")
		require.Equal(t, data[3], model.LastName.String, "expected field LastName to match data[3]")
		require.Equal(t, data[4], model.TravelAddress.String, "expected field TravelAddress to match data[4]")
		require.Equal(t, data[6], model.Created, "expected field Created to match data[6]")
		require.Equal(t, data[7], model.Modified, "expected field Modified to match data[7]")
	})
}

func TestCryptoAddressScan(t *testing.T) {
	t.Run("SuccessFilled", func(t *testing.T) {
		// setup
		data := []any{
			ulid.MakeSecure().String(), // ID
			ulid.MakeSecure().String(), // AccountID
			"CryptoAddress",            // CryptoAddress
			"Network",                  // Network
			"AssetType",                // AssetType
			"Tag",                      // Tag
			"TravelAddress",            // TravelAddress
			time.Now(),                 // Created
			time.Now(),                 // Modified
		}
		mockScanner := &mock.MockScanner{}
		mockScanner.SetData(data)

		// test
		model := &models.CryptoAddress{}
		err := model.Scan(mockScanner)
		require.NoError(t, err, "expected no errors from the scanner")
		mockScanner.AssertScanned(t, len(data))

		// make sure scanned data matches the fields they were supposed to scan into
		require.Equal(t, data[0], model.ID.String(), "expected field ID to match data[0]")
		require.Equal(t, data[1], model.AccountID.String(), "expected field AccountID to match data[1]")
		require.Equal(t, data[2], model.CryptoAddress, "expected field CryptoAddress to match data[2]")
		require.Equal(t, data[3], model.Network, "expected field Network to match data[3]")
		require.Equal(t, data[4], model.AssetType.String, "expected field AssetType to match data[4]")
		require.Equal(t, data[5], model.Tag.String, "expected field Tag to match data[5]")
		require.Equal(t, data[6], model.TravelAddress.String, "expected field TravelAddress to match data[6]")
		require.Equal(t, data[7], model.Created, "expected field Created to match data[7]")
		require.Equal(t, data[8], model.Modified, "expected field Modified to match data[8]")
	})

	t.Run("SuccessNulls", func(t *testing.T) {
		// setup
		data := []any{
			ulid.MakeSecure().String(), // ID
			ulid.Zero.String(),         // AccountID (testing a zero ULID)
			"CryptoAddress",            // CryptoAddress
			"Network",                  // Network
			nil,                        // AssetType (testing a null string)
			nil,                        // Tag (testing a null string)
			nil,                        // TravelAddress (testing a null string)
			time.Now(),                 // Created
			time.Time{},                // Modified (testing a zero value time)
		}
		mockScanner := &mock.MockScanner{}
		mockScanner.SetData(data)

		// test
		model := &models.CryptoAddress{}
		err := model.Scan(mockScanner)
		require.NoError(t, err, "expected no errors from the scanner")
		mockScanner.AssertScanned(t, len(data))
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
