package mock

import (
	"database/sql"
	"time"

	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/trisa/pkg/ivms101"
	"go.rtnl.ai/ulid"
)

// ============================================================================
// This file contains functions that allow a test-creator to get a "sample"
// of each of the models in `store/models`. These "sample" models are filled
// with dummy data of the correct types and can be used for most test
// operations.
// ============================================================================

// Returns a sample Account. Can add the IVMS101 and CryptoAddresses and include
// or exclude `NullType` values.
func GetSampleAccount(includeNulls bool, addIvms101 bool, addCrypto bool, zeroID bool) (account *models.Account) {
	timeNow := time.Now()

	var id ulid.ULID
	if zeroID {
		id = ulid.Zero
	} else {
		id = ulid.MakeSecure()
	}

	account = &models.Account{
		Model: models.Model{
			ID:       id,
			Created:  timeNow,
			Modified: timeNow.Add(1 * time.Hour)},
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
				AccountID:     id,
				CryptoAddress: "CryptoAddress1",
				Network:       "BTC",
			},
			{
				AccountID:     id,
				CryptoAddress: "CryptoAddress2",
				Network:       "BTC",
			},
		}
		account.SetCryptoAddresses(addresses)
	}

	return account
}

// Returns a sample CryptoAddress for the account ID provided. The crypto and
// travel addresses will be a random ULID strings and the network is "BTC".
func GetSampleCryptoAddress(accountId ulid.ULID) (account *models.CryptoAddress) {
	return &models.CryptoAddress{
		AccountID:     accountId,
		CryptoAddress: ulid.MakeSecure().String(),
		Network:       "BTC",
		TravelAddress: sql.NullString{String: ulid.MakeSecure().String(), Valid: true},
	}
}
