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

//==========================================================================
// Tests
//==========================================================================

func TestSunriseParams(t *testing.T) {
	// setup a model
	theModel := mock.GetSampleSunrise(true)

	// create the model public field name comparison list
	fields := GetPublicFieldNames(*theModel)

	// create the `Params()` comparison list
	// Exceptions: None
	exceptions := map[string]string{}
	params := GetParamsNames(theModel, exceptions)

	// test
	require.ElementsMatch(t, fields, params, "the model's public fields and Params() lists should have the same names")
}

func TestSunriseScan(t *testing.T) {
	t.Run("SuccessFilled", func(t *testing.T) {
		// setup
		data := []any{
			ulid.MakeSecure().String(), // ID
			uuid.New().String(),        // EnvelopeID
			"email@example.com",        // Email
			time.Now(),                 // Expiration
			nil,                        // Signature (vero token ignored)
			"accepted",                 // Status
			time.Now(),                 // SentOn
			time.Now(),                 // VerifiedOn
			time.Now(),                 // Created
			time.Now(),                 // Modified
		}
		mockScanner := &mock.MockScanner{}
		mockScanner.SetData(data)

		// test
		model := &models.Sunrise{}
		err := model.Scan(mockScanner)
		require.NoError(t, err, "expected no errors from the scanner")
		mockScanner.AssertScanned(t, len(data)-1) // Signature won't scan

		// make sure scanned data matches the fields they were supposed to scan into
		require.Equal(t, data[0], model.ID.String(), "expected field ID to match data[0]")
		require.Equal(t, data[1], model.EnvelopeID.String(), "expected field EnvelopeID to match data[1]")
		require.Equal(t, data[2], model.Email, "expected field Email to match data[2]")
		require.Equal(t, data[3], model.Expiration, "expected field Expiration to match data[3]")
		require.Equal(t, data[5], model.Status.String(), "expected field Status to match data[5]")
		require.Equal(t, data[6], model.SentOn.Time, "expected field SentOn to match data[6]")
		require.Equal(t, data[7], model.VerifiedOn.Time, "expected field VerifiedOn to match data[7]")
		require.Equal(t, data[8], model.Created, "expected field Created to match data[8]")
		require.Equal(t, data[9], model.Modified, "expected field Modified to match data[9]")
	})

	t.Run("SuccessNulls", func(t *testing.T) {
		// setup
		data := []any{
			ulid.MakeSecure().String(), // ID
			uuid.New().String(),        // EnvelopeID
			"email@example.com",        // Email
			time.Now(),                 // Expiration
			nil,                        // Signature (vero token ignored)
			"accepted",                 // Status
			nil,                        // SentOn (testing null time)
			nil,                        // VerifiedOn (testing null time)
			time.Now(),                 // Created
			time.Now(),                 // Modified
		}
		mockScanner := &mock.MockScanner{}
		mockScanner.SetData(data)

		// test
		model := &models.Sunrise{}
		err := model.Scan(mockScanner)
		require.NoError(t, err, "expected no errors from the scanner")
		mockScanner.AssertScanned(t, len(data)-1) // Signature won't scan
	})
}
