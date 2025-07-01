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
		actorId, err := uuid.New().MarshalBinary()
		require.NoError(t, err, "expected no error getting UUID bytes")
		data := []any{
			ulid.MakeSecure().Bytes(),  // ID
			actorId,                    // ActorID
			"user",                     // ActorType
			ulid.MakeSecure().Bytes(),  // ResourceID
			"transaction",              // ResourceType
			time.Now(),                 // ResourceModified
			"update",                   // Action
			ulid.MakeSecure().String(), // ResourceActionMeta
			ulid.MakeSecure().Bytes(),  // Signature (a fake)
			ulid.MakeSecure().String(), // KeyID (a fake)
		}
		mockScanner := &mock.MockScanner{}
		mockScanner.SetData(data)

		//test
		model := &models.ComplianceAuditLog{}
		err = model.Scan(mockScanner)
		require.NoError(t, err, "expected no errors when scanning")
		mockScanner.AssertScanned(t, len(data))

		// make sure scanned data matches the fields they were supposed to scan into
		require.Equal(t, data[0], model.ID.Bytes(), "expected field ID to match data[0]")
		require.Equal(t, data[1], model.ActorID, "expected field ActorID to match data[1]")
		require.Equal(t, data[2], model.ActorType.String(), "expected field ActorType to match data[2]")
		require.Equal(t, data[3], model.ResourceID, "expected field ResourceID to match data[3]")
		require.Equal(t, data[4], model.ResourceType.String(), "expected field ResourceType to match data[4]")
		require.Equal(t, data[5], model.ResourceModified, "expected field ResourceModified to match data[5]")
		require.Equal(t, data[6], model.Action.String(), "expected field Action to match data[6]")
		require.Equal(t, data[7], model.ResourceActionMeta.String, "expected field ResourceActionMeta to match data[7]")
		require.Equal(t, data[8], model.Signature, "expected Signature to match data[8]")
		require.Equal(t, data[9], model.KeyID, "expected KeyID to match data[9]")
	})

	t.Run("SuccessNoMeta", func(t *testing.T) {
		//setup
		resourceId, err := uuid.New().MarshalBinary()
		require.NoError(t, err, "expected no error getting UUID bytes")
		data := []any{
			ulid.MakeSecure().Bytes(),  // ID
			ulid.MakeSecure().Bytes(),  // ActorID
			"user",                     // ActorType
			resourceId,                 // ResourceID
			"transaction",              // ResourceType
			time.Now(),                 // ResourceModified
			"update",                   // Action
			nil,                        // ResourceActionMeta
			ulid.MakeSecure().Bytes(),  // Signature (a fake)
			ulid.MakeSecure().String(), // KeyID (a fake)
		}
		mockScanner := &mock.MockScanner{}
		mockScanner.SetData(data)

		//test
		model := &models.ComplianceAuditLog{}
		err = model.Scan(mockScanner)
		require.NoError(t, err, "expected no errors when scanning")
		mockScanner.AssertScanned(t, len(data))

		// make sure scanned data matches the fields they were supposed to scan into
		require.Equal(t, data[0], model.ID.Bytes(), "expected field ID to match data[0]")
		require.Equal(t, data[1], model.ActorID, "expected field ActorID to match data[1]")
		require.Equal(t, data[2], model.ActorType.String(), "expected field ActorType to match data[2]")
		require.Equal(t, data[3], model.ResourceID, "expected field ResourceID to match data[3]")
		require.Equal(t, data[4], model.ResourceType.String(), "expected field ResourceType to match data[4]")
		require.Equal(t, data[5], model.ResourceModified, "expected field ResourceModified to match data[5]")
		require.Equal(t, data[6], model.Action.String(), "expected field Action to match data[6]")
		require.False(t, model.ResourceActionMeta.Valid, "expected field ResourceActionMeta.Valid to be false")
		require.Equal(t, data[8], model.Signature, "expected field Signature to match data[8]")
		require.Equal(t, data[9], model.KeyID, "expected field KeyID to match data[9]")
	})
}

// TODO (sc-32721): tests for ComplianceAuditLog.Sign()

// TODO (sc-32721): tests for ComplianceAuditLog.Verify()
