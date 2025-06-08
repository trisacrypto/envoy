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
	"github.com/trisacrypto/envoy/pkg/store/mock"
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

func TestTransactionScan(t *testing.T) {
	t.Run("SuccessFilled", func(t *testing.T) {
		// setup
		data := []any{
			uuid.New().String(),        // ID
			"local",                    // Source
			"accepted",                 // Status
			"Counterparty",             // Counterparty
			ulid.MakeSecure().String(), // CounterpartyID
			"Originator",               // Originator
			"OriginatorAddress",        // OriginatorAddress
			"Beneficiary",              // Beneficiary
			"BeneficiaryAddress",       // BeneficiaryAddress
			"VirtualAsset",             // VirtualAsset
			float64(1.2345),            // Amount
			false,                      // Archived
			time.Now(),                 // ArchivedOn
			time.Now(),                 // LastUpdate
			time.Now(),                 // Created
			time.Now(),                 // Modified
		}
		mockScanner := &mock.MockScanner{}
		mockScanner.SetData(data)

		// test
		model := &models.Transaction{}
		err := model.Scan(mockScanner)
		require.NoError(t, err, "expected no errors from the scanner")
		mockScanner.AssertScanned(t, len(data))

		// make sure scanned data matches the fields they were supposed to scan into
		require.Equal(t, data[0], model.ID.String(), "expected field ID to match data[0]")
		require.Equal(t, data[1], model.Source.String(), "expected field Source to match data[1]")
		require.Equal(t, data[2], model.Status.String(), "expected field Status to match data[2]")
		require.Equal(t, data[3], model.Counterparty, "expected field Counterparty to match data[3]")
		require.Equal(t, data[4], model.CounterpartyID.ULID.String(), "expected field CounterpartyID to match data[4]")
		require.Equal(t, data[5], model.Originator.String, "expected field Originator to match data[5]")
		require.Equal(t, data[6], model.OriginatorAddress.String, "expected field OriginatorAddress to match data[6]")
		require.Equal(t, data[7], model.Beneficiary.String, "expected field Beneficiary to match data[7]")
		require.Equal(t, data[8], model.BeneficiaryAddress.String, "expected field BeneficiaryAddress to match data[8]")
		require.Equal(t, data[9], model.VirtualAsset, "expected field VirtualAsset to match data[9]")
		require.Equal(t, data[10], model.Amount, "expected field Amount to match data[10]")
		require.Equal(t, data[11], model.Archived, "expected field Archived to match data[11]")
		require.Equal(t, data[12], model.ArchivedOn.Time, "expected field ArchivedOn to match data[12]")
		require.Equal(t, data[13], model.LastUpdate.Time, "expected field LastUpdate to match data[13]")
		require.Equal(t, data[14], model.Created, "expected field Created to match data[14]")
		require.Equal(t, data[15], model.Modified, "expected field Modified to match data[15]")
	})

	t.Run("SuccessNulls", func(t *testing.T) {
		// setup
		data := []any{
			uuid.New().String(),        // ID
			"local",                    // Source
			"accepted",                 // Status
			"Counterparty",             // Counterparty
			ulid.MakeSecure().String(), // CounterpartyID
			nil,                        // Originator (testing null string)
			nil,                        // OriginatorAddress (testing null string)
			nil,                        // Beneficiary (testing null string)
			nil,                        // BeneficiaryAddress (testing null string)
			"VirtualAsset",             // VirtualAsset
			float64(1.2345),            // Amount
			false,                      // Archived
			nil,                        // ArchivedOn (testing null time)
			nil,                        // LastUpdate (testing null time)
			time.Now(),                 // Created
			time.Now(),                 // Modified
		}
		mockScanner := &mock.MockScanner{}
		mockScanner.SetData(data)

		// test
		model := &models.Transaction{}
		err := model.Scan(mockScanner)
		require.NoError(t, err, "expected no errors from the scanner")
		mockScanner.AssertScanned(t, len(data))
	})

	t.Run("InvalidSource", func(t *testing.T) {
		// setup
		data := []any{
			uuid.New().String(),        // ID
			"not_a_source_829134",      // Source
			"accepted",                 // Status
			"Counterparty",             // Counterparty
			ulid.MakeSecure().String(), // CounterpartyID
			nil,                        // Originator (testing null string)
			nil,                        // OriginatorAddress (testing null string)
			nil,                        // Beneficiary (testing null string)
			nil,                        // BeneficiaryAddress (testing null string)
			"VirtualAsset",             // VirtualAsset
			float64(1.2345),            // Amount
			false,                      // Archived
			nil,                        // ArchivedOn (testing null time)
			nil,                        // LastUpdate (testing null time)
			time.Now(),                 // Created
			time.Now(),                 // Modified
		}
		mockScanner := &mock.MockScanner{}
		mockScanner.SetData(data)

		// test
		model := &models.Transaction{}
		err := model.Scan(mockScanner)
		require.Error(t, err, "expected an error from the scanner")
		require.ErrorContains(t, err, "invalid source", "expected an 'invalid source' error from the scanner")
	})

	t.Run("InvalidStatus", func(t *testing.T) {
		// setup
		data := []any{
			uuid.New().String(),        // ID
			"local",                    // Source
			"not_a_status_987342",      // Status
			"Counterparty",             // Counterparty
			ulid.MakeSecure().String(), // CounterpartyID
			nil,                        // Originator (testing null string)
			nil,                        // OriginatorAddress (testing null string)
			nil,                        // Beneficiary (testing null string)
			nil,                        // BeneficiaryAddress (testing null string)
			"VirtualAsset",             // VirtualAsset
			float64(1.2345),            // Amount
			false,                      // Archived
			nil,                        // ArchivedOn (testing null time)
			nil,                        // LastUpdate (testing null time)
			time.Now(),                 // Created
			time.Now(),                 // Modified
		}
		mockScanner := &mock.MockScanner{}
		mockScanner.SetData(data)

		// test
		model := &models.Transaction{}
		err := model.Scan(mockScanner)
		require.Error(t, err, "expected an error from the scanner")
		require.ErrorContains(t, err, "invalid status", "expected an 'invalid status' error from the scanner")
	})
}

func TestSecureEnvelopeScan(t *testing.T) {
	t.Run("SuccessFilled", func(t *testing.T) {
		// setup
		data := []any{
			ulid.MakeSecure().String(), // ID
			uuid.New().String(),        // EnvelopeID
			"in",                       // Direction
			false,                      // IsError
			[]byte("EncryptionKey"),    // EncryptionKey
			[]byte("HMACSecret"),       // HMACSecret
			false,                      // ValidHMAC
			time.Now(),                 // Timestamp
			"PublicKey",                // PublicKey
			nil,                        // Envelope
			time.Now(),                 // Created
			time.Now(),                 // Modified
			"Remote",                   // Remote
			ulid.MakeSecure().String(), // ReplyTo
			int32(808),                 // TransferState
		}
		mockScanner := &mock.MockScanner{}
		mockScanner.SetData(data)

		// test
		model := &models.SecureEnvelope{}
		err := model.Scan(mockScanner)
		require.NoError(t, err, "expected no errors from the scanner")
		mockScanner.AssertScanned(t, len(data)-1) // Envelope will not scan

		// make sure scanned data matches the fields they were supposed to scan into
		require.Equal(t, data[0], model.ID.String(), "expected field ID to match data[0]")
		require.Equal(t, data[1], model.EnvelopeID.String(), "expected field EnvelopeID to match data[1]")
		require.Equal(t, data[2], model.Direction.String(), "expected field Direction to match data[2]")
		require.Equal(t, data[3], model.IsError, "expected field IsError to match data[3]")
		require.Equal(t, data[4], model.EncryptionKey, "expected field EncryptionKey to match data[4]")
		require.Equal(t, data[5], model.HMACSecret, "expected field HMACSecret to match data[5]")
		require.Equal(t, data[6], model.ValidHMAC.Bool, "expected field ValidHMAC to match data[6]")
		require.Equal(t, data[7], model.Timestamp, "expected field Timestamp to match data[7]")
		require.Equal(t, data[8], model.PublicKey.String, "expected field PublicKey to match data[8]")
		require.Equal(t, data[10], model.Created, "expected field Created to match data[10]")
		require.Equal(t, data[11], model.Modified, "expected field Modified to match data[11]")
		require.Equal(t, data[12], model.Remote.String, "expected field Remote to match data[12]")
		require.Equal(t, data[13], model.ReplyTo.ULID.String(), "expected field ReplyTo to match data[13]")
		require.Equal(t, data[14], model.TransferState, "expected field TransferState to match data[14]")
	})

	t.Run("SuccessNulls", func(t *testing.T) {
		// setup
		data := []any{
			ulid.MakeSecure().String(), // ID
			uuid.New().String(),        // EnvelopeID
			"incoming",                 // Direction
			false,                      // IsError
			[]byte("EncryptionKey"),    // EncryptionKey
			[]byte("HMACSecret"),       // HMACSecret
			nil,                        // ValidHMAC (testing null)
			time.Now(),                 // Timestamp
			nil,                        // PublicKey (testing null)
			nil,                        // Envelope
			time.Now(),                 // Created
			time.Now(),                 // Modified
			nil,                        // Remote (testing null)
			nil,                        // ReplyTo (testing null)
			int32(808),                 // TransferState
		}
		mockScanner := &mock.MockScanner{}
		mockScanner.SetData(data)

		// test
		model := &models.SecureEnvelope{}
		err := model.Scan(mockScanner)
		require.NoError(t, err, "expected no errors from the scanner")
		mockScanner.AssertScanned(t, len(data)-1) // Envelope will not scan
	})

	t.Run("InvalidDirection", func(t *testing.T) {
		// setup
		data := []any{
			ulid.MakeSecure().String(), // ID
			uuid.New().String(),        // EnvelopeID
			"not_a_direction_23847",    // Direction
			false,                      // IsError
			[]byte("EncryptionKey"),    // EncryptionKey
			[]byte("HMACSecret"),       // HMACSecret
			nil,                        // ValidHMAC (testing null)
			time.Now(),                 // Timestamp
			nil,                        // PublicKey (testing null)
			nil,                        // Envelope
			time.Now(),                 // Created
			time.Now(),                 // Modified
			nil,                        // Remote (testing null)
			nil,                        // ReplyTo (testing null)
			int32(808),                 // TransferState
		}
		mockScanner := &mock.MockScanner{}
		mockScanner.SetData(data)

		// test
		model := &models.SecureEnvelope{}
		err := model.Scan(mockScanner)
		require.Error(t, err, "expected an error from the scanner")
		require.ErrorContains(t, err, "invalid direction", "expected an 'invalid direction' error from the scanner")
	})
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
