package models_test

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/enum"
	"github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"go.rtnl.ai/ulid"
)

//==========================================================================
// Tests
//==========================================================================

func TestTransactionParams(t *testing.T) {
	// setup a model
	theModel := getSampleTransaction(true, false)

	// create the model public field name comparison list
	fields := GetPublicFieldNames(*theModel)

	// create the `Params()` comparison list
	// Exceptions: None
	exceptions := map[string]string{}
	params := GetParamsNames(theModel, exceptions)

	// test
	require.ElementsMatch(t, fields, params, "the model's public fields and Params() lists should have the same names")
}
func TestSecureEnvelopeParams(t *testing.T) {
	// setup a model
	theModel := getSampleSecureEnvelope(true, false)

	// create the model public field name comparison list
	fields := GetPublicFieldNames(*theModel)

	// create the `Params()` comparison list
	// Exceptions: None
	exceptions := map[string]string{}
	params := GetParamsNames(theModel, exceptions)

	// test
	require.ElementsMatch(t, fields, params, "the model's public fields and Params() lists should have the same names")
}

func TestTransactionSecureEnvelopes(t *testing.T) {
	// test 1: has envelopes
	envelopes, err := getSampleTransaction(true, true).SecureEnvelopes()
	require.NotNil(t, envelopes, "envelopes should not be nil")
	require.Nil(t, err, "error should be nil")

	//test 2: no envelopes
	envelopes, err = getSampleTransaction(true, false).SecureEnvelopes()
	require.Nil(t, envelopes, "envelopes should be nil")
	require.Error(t, err, "error should not be nil")
	require.Equal(t, errors.ErrMissingAssociation, err, "error should be ErrMissingAssociation")

}

func TestTransactionNumEnvelopes(t *testing.T) {
	// test 1: no envelopes
	number := getSampleTransaction(true, false).NumEnvelopes()
	require.Equal(t, int64(0), number, fmt.Sprintf("there should be 0 envelopes but there were %d", number))

	//test 2: has envelopes
	number = getSampleTransaction(true, true).NumEnvelopes()
	require.Equal(t, int64(1), number, fmt.Sprintf("there should be 1 envelopes but there were %d", number))

}

func TestSecureEnvelopeTransaction(t *testing.T) {
	// test 1: has transaction
	transaction, err := getSampleSecureEnvelope(true, true).Transaction()
	require.NotNil(t, transaction, "transaction should not be nil")
	require.Nil(t, err, "error should be nil")

	//test 2: no transaction
	transaction, err = getSampleSecureEnvelope(true, false).Transaction()
	require.Nil(t, transaction, "transaction should be nil")
	require.Error(t, err, "error should not be nil")
	require.Equal(t, errors.ErrMissingAssociation, err, "error should be ErrMissingAssociation")

}

//==========================================================================
// Helpers
//==========================================================================

// Returns a sample Transaction.
func getSampleTransaction(includeNulls bool, includeEnvelopes bool) (model *models.Transaction) {
	id := uuid.New()
	timeNow := time.Now()

	model = &models.Transaction{
		ID:                 id,
		Source:             enum.SourceDirectorySync,
		Status:             enum.StatusAccepted,
		Counterparty:       "Counterparty",
		CounterpartyID:     ulid.NullULID{},
		Originator:         sql.NullString{},
		OriginatorAddress:  sql.NullString{},
		Beneficiary:        sql.NullString{},
		BeneficiaryAddress: sql.NullString{},
		VirtualAsset:       "BTC",
		Amount:             0.123456,
		Archived:           false,
		ArchivedOn:         sql.NullTime{},
		LastUpdate:         sql.NullTime{},
		Created:            timeNow,
		Modified:           timeNow,
	}

	if includeNulls {
		model.CounterpartyID = ulid.NullULID{ULID: ulid.MakeSecure(), Valid: true}
		model.Originator = sql.NullString{String: "Originator", Valid: true}
		model.OriginatorAddress = sql.NullString{String: "OriginatorAddress", Valid: true}
		model.Beneficiary = sql.NullString{String: "Beneficiary", Valid: true}
		model.BeneficiaryAddress = sql.NullString{String: "BeneficiaryAddress", Valid: true}
		model.Archived = true
		model.ArchivedOn = sql.NullTime{Time: timeNow, Valid: true}
		model.LastUpdate = sql.NullTime{Time: timeNow, Valid: true}
	}

	if includeEnvelopes {
		model.SetSecureEnvelopes([]*models.SecureEnvelope{getSampleSecureEnvelope(true, false)})
	}

	return model
}

// Returns a sample SecureEnvelope.
func getSampleSecureEnvelope(includeNulls bool, includeTransaction bool) (model *models.SecureEnvelope) {
	id := ulid.MakeSecure()
	timeNow := time.Now()

	model = &models.SecureEnvelope{
		Model: models.Model{
			ID:       id,
			Created:  timeNow,
			Modified: timeNow,
		},
		EnvelopeID:    uuid.New(),
		Direction:     enum.DirectionOutgoing,
		Remote:        sql.NullString{},
		ReplyTo:       ulid.NullULID{},
		IsError:       false,
		EncryptionKey: nil,
		HMACSecret:    nil,
		ValidHMAC:     sql.NullBool{},
		Timestamp:     timeNow,
		PublicKey:     sql.NullString{},
		TransferState: 808,
		Envelope:      nil,
	}

	if includeNulls {
		model.Remote = sql.NullString{String: "Remote", Valid: true}
		model.ReplyTo = ulid.NullULID{ULID: ulid.MakeSecure(), Valid: true}
		model.ValidHMAC = sql.NullBool{Bool: false, Valid: true}
		model.PublicKey = sql.NullString{String: "PublicKey", Valid: true}
	}

	if includeTransaction {
		model.SetTransaction(getSampleTransaction(true, false))
	}

	return model
}
