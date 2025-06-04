package models_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/enum"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"go.rtnl.ai/ulid"
)

//==========================================================================
// Tests
//==========================================================================

func TestSunriseParams(t *testing.T) {
	// setup a model
	theModel := getSampleSunrise(true)

	// create the model public field name comparison list
	fields := GetPublicFieldNames(*theModel)

	// create the `Params()` comparison list
	// Exceptions: None
	exceptions := map[string]string{}
	params := GetParamsNames(theModel, exceptions)

	// test
	require.ElementsMatch(t, fields, params, "the model's public fields and Params() lists should have the same names")
}

//==========================================================================
// Helpers
//==========================================================================

// Returns a sample Sunrise.
func getSampleSunrise(includeNulls bool) (model *models.Sunrise) {
	id := ulid.MakeSecure()
	envId := uuid.New()
	timeNow := time.Now()

	model = &models.Sunrise{
		Model: models.Model{
			ID:       id,
			Created:  timeNow,
			Modified: timeNow,
		},
		EnvelopeID: envId,
		Email:      "email@example.com",
		Expiration: timeNow.Add(1 * time.Hour),
		Signature:  nil,
		Status:     enum.StatusDraft,
		SentOn:     sql.NullTime{},
		VerifiedOn: sql.NullTime{},
	}

	if includeNulls {
		model.SentOn = sql.NullTime{Time: timeNow, Valid: true}
		model.VerifiedOn = sql.NullTime{Time: timeNow, Valid: true}
	}

	return model
}
