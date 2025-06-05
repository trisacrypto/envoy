package models_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"go.rtnl.ai/ulid"
)

//==========================================================================
// Tests
//==========================================================================

func TestUserParams(t *testing.T) {
	// setup a model
	theModel := getSampleUser(true)

	// create the model public field name comparison list
	fields := GetPublicFieldNames(*theModel)

	// create the `Params()` comparison list
	// Exceptions: None
	exceptions := map[string]string{}
	params := GetParamsNames(theModel, exceptions)

	// test
	require.ElementsMatch(t, fields, params, "the model's public fields and Params() lists should have the same names")
}

func TestAPIKeyParams(t *testing.T) {
	// setup a model
	theModel := getSampleAPIKey(true)

	// create the model public field name comparison list
	fields := GetPublicFieldNames(*theModel)

	// create the `Params()` comparison list
	// Exceptions: None
	exceptions := map[string]string{}
	params := GetParamsNames(theModel, exceptions)

	// test
	require.ElementsMatch(t, fields, params, "the model's public fields and Params() lists should have the same names")
}

func TestRoleParams(t *testing.T) {
	// setup a model
	theModel := getSampleRole(808, false)

	// create the model public field name comparison list
	fields := GetPublicFieldNames(*theModel)

	// create the `Params()` comparison list
	// Exceptions: None
	exceptions := map[string]string{}
	params := GetParamsNames(theModel, exceptions)

	// test
	require.ElementsMatch(t, fields, params, "the model's public fields and Params() lists should have the same names")
}

func TestRolePermissions(t *testing.T) {
	// test 1: has permissions
	permissions, err := getSampleRole(808, true).Permissions()
	require.NotNil(t, permissions, "permissions should not be nil")
	require.Nil(t, err, "error should be nil")

	//test 2: no permissions
	permissions, err = getSampleRole(808, false).Permissions()
	require.Nil(t, permissions, "permissions should be nil")
	require.Error(t, err, "error should not be nil")
	require.Equal(t, errors.ErrMissingAssociation, err, "error should be ErrMissingAssociation")
}

func TestPermissionParams(t *testing.T) {
	// setup a model
	theModel := getSamplePermission(808)

	// create the model public field name comparison list
	fields := GetPublicFieldNames(*theModel)

	// create the `Params()` comparison list
	// Exceptions: None
	exceptions := map[string]string{}
	params := GetParamsNames(theModel, exceptions)

	// test
	require.ElementsMatch(t, fields, params, "the model's public fields and Params() lists should have the same names")
}

func TestResetPasswordLinkParams(t *testing.T) {
	// setup a model
	theModel := getSampleResetPasswordLink(true)

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

// Returns a sample User. Can include or exclude any `NullType` types.
func getSampleUser(includeNulls bool) (model *models.User) {
	id := ulid.MakeSecure()
	timeNow := time.Now()

	model = &models.User{
		Model: models.Model{
			ID:       id,
			Created:  timeNow,
			Modified: timeNow,
		},
		Email:     "email@example.com",
		Password:  "Password",
		RoleID:    808,
		LastLogin: sql.NullTime{},
	}

	if includeNulls {
		model.LastLogin = sql.NullTime{Time: timeNow, Valid: true}
	}

	return model
}

// Returns a sample APIKey. Can include or exclude any `NullType` types.
func getSampleAPIKey(includeNulls bool) (model *models.APIKey) {
	id := ulid.MakeSecure()
	timeNow := time.Now()

	model = &models.APIKey{
		Model: models.Model{
			ID:       id,
			Created:  timeNow,
			Modified: timeNow,
		},
		Description: sql.NullString{},
		ClientID:    "ClientID",
		Secret:      "Secret",
		LastSeen:    sql.NullTime{},
	}

	if includeNulls {
		model.Description = sql.NullString{String: "Description", Valid: true}
		model.LastSeen = sql.NullTime{Time: timeNow, Valid: true}
	}

	return model
}

// Returns a sample Role. Can include sample Permissions with it.
func getSampleRole(id int64, includePermissions bool) (model *models.Role) {
	timeNow := time.Now()

	model = &models.Role{
		ID:          id,
		Created:     timeNow,
		Modified:    timeNow,
		Title:       "Title",
		Description: "Description",
		IsDefault:   true,
	}

	if includePermissions {
		model.SetPermissions([]*models.Permission{getSamplePermission(1), getSamplePermission(2)})
	}

	return model
}

// Returns a sample Permission.
func getSamplePermission(id int64) (model *models.Permission) {
	timeNow := time.Now()

	model = &models.Permission{
		ID:          id,
		Created:     timeNow,
		Modified:    timeNow,
		Title:       "Title",
		Description: "Description",
	}

	return model
}

// Returns a sample ResetPasswordLink. Can include or exclude any `NullType` types.
func getSampleResetPasswordLink(includeNulls bool) (model *models.ResetPasswordLink) {
	id := ulid.MakeSecure()
	userid := ulid.MakeSecure()
	timeNow := time.Now()

	model = &models.ResetPasswordLink{
		Model: models.Model{
			ID:       id,
			Created:  timeNow,
			Modified: timeNow,
		},
		UserID:     userid,
		Email:      "email@example.com",
		Expiration: timeNow.Add(1 * time.Hour),
		Signature:  nil, // these tokens are tested in their package
		SentOn:     sql.NullTime{},
	}

	if includeNulls {
		model.SentOn = sql.NullTime{Time: timeNow, Valid: true}
	}

	return model
}
