package models_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/store/mock"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"go.rtnl.ai/ulid"
)

func TestComplianceAuditLogParams(t *testing.T) {
	// setup a model
	theModel := mock.GetComplianceAuditLog(false, false)

	// create the model public field name comparison list
	fields := GetPublicFieldNames(*theModel)

	// create the `Params()` comparison list
	// Exceptions: None
	exceptions := map[string]string{}
	params := GetParamsNames(theModel, exceptions)

	// test
	require.ElementsMatch(t, fields, params, "the model's public fields and Params() lists should have the same names")
}

func TestComplianceAuditLogScan(t *testing.T) {
	t.Run("SuccessMeta", func(t *testing.T) {
		//setup
		data := []any{
			uuid.New().String(),        // ID
			time.Now(),                 // Timestamp
			ulid.MakeSecure().Bytes(),  // ActorID
			"user",                     // ActorType
			ulid.MakeSecure().Bytes(),  // ResourceID
			"transaction",              // ResourceType
			"update",                   // Action
			ulid.MakeSecure().String(), // ResourceActionMeta
			ulid.MakeSecure().Bytes(),  // Signature (a fake)
		}
		mockScanner := &mock.MockScanner{}
		mockScanner.SetData(data)

		//test
		model := &models.ComplianceAuditLog{}
		err := model.Scan(mockScanner)
		require.NoError(t, err, "expected no errors when scanning")
		mockScanner.AssertScanned(t, len(data))

		// make sure scanned data matches the fields they were supposed to scan into
		require.Equal(t, data[0], model.ID.String(), "expected field ID to match data[0]")
		require.Equal(t, data[1], model.Timestamp, "expected field Timestamp to match data[1]")
		require.Equal(t, data[2], model.ActorID, "expected field ActorID to match data[2]")
		require.Equal(t, data[3], model.ActorType.String(), "expected field ActorType to match data[3]")
		require.Equal(t, data[4], model.ResourceID, "expected field ResourceID to match data[4]")
		require.Equal(t, data[5], model.ResourceType.String(), "expected field ResourceType to match data[5]")
		require.Equal(t, data[6], model.Action.String(), "expected field Action to match data[6]")
		require.Equal(t, data[7], model.ResourceActionMeta.String, "expected field ResourceActionMeta to match data[7]")
		require.Equal(t, data[8], model.Signature, "expected field Signature to match data[8]")
	})

	t.Run("SuccessNoMeta", func(t *testing.T) {
		//setup
		data := []any{
			uuid.New().String(),       // ID
			time.Now(),                // Timestamp
			ulid.MakeSecure().Bytes(), // ActorID
			"user",                    // ActorType
			ulid.MakeSecure().Bytes(), // ResourceID
			"transaction",             // ResourceType
			"update",                  // Action
			nil,                       // ResourceActionMeta
			ulid.MakeSecure().Bytes(), // Signature (a fake)
		}
		mockScanner := &mock.MockScanner{}
		mockScanner.SetData(data)

		//test
		model := &models.ComplianceAuditLog{}
		err := model.Scan(mockScanner)
		require.NoError(t, err, "expected no errors when scanning")
		mockScanner.AssertScanned(t, len(data))

		// make sure scanned data matches the fields they were supposed to scan into
		require.Equal(t, data[0], model.ID.String(), "expected field ID to match data[0]")
		require.Equal(t, data[1], model.Timestamp, "expected field Timestamp to match data[1]")
		require.Equal(t, data[2], model.ActorID, "expected field ActorID to match data[2]")
		require.Equal(t, data[3], model.ActorType.String(), "expected field ActorType to match data[3]")
		require.Equal(t, data[4], model.ResourceID, "expected field ResourceID to match data[4]")
		require.Equal(t, data[5], model.ResourceType.String(), "expected field ResourceType to match data[5]")
		require.Equal(t, data[6], model.Action.String(), "expected field Action to match data[6]")
		require.False(t, model.ResourceActionMeta.Valid, "expected field ResourceActionMeta.Valid to be false")
		require.Equal(t, data[8], model.Signature, "expected field Signature to match data[8]")
	})
}

// FIXME: tests for Sign()

// FIXME: tests for Verify()
