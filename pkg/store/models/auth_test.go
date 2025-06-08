package models_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/mock"
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

func TestUserScan(t *testing.T) {
	t.Run("SuccessFilled", func(t *testing.T) {
		//setup
		data := []any{
			ulid.MakeSecure().String(),    // ID
			"First Last",                  // Name
			"email@example.com",           // Email
			"Password",                    // Password
			int64(808),                    // RoleID
			time.Now(),                    // LastLogin
			time.Now(),                    // Created
			time.Now().Add(1 * time.Hour), // Modified
		}
		mockScanner := &mock.MockScanner{}
		mockScanner.SetData(data)

		//test
		model := &models.User{}
		err := model.Scan(mockScanner)
		require.NoError(t, err, "expected no errors when scanning")
		mockScanner.AssertScanned(t, len(data))
	})

	t.Run("SuccessNulls", func(t *testing.T) {
		//setup
		data := []any{
			ulid.MakeSecure().String(), // ID
			nil,                        // Name (testing null string)
			"email@example.com",        // Email
			"Password",                 // Password
			int64(808),                 // RoleID
			nil,                        // LastLogin (testing null time)
			time.Now(),                 // Created
			time.Time{},                // Modified (testing zero time)
		}
		mockScanner := &mock.MockScanner{}
		mockScanner.SetData(data)

		//test
		model := &models.User{}
		err := model.Scan(mockScanner)
		require.NoError(t, err, "expected no errors when scanning")
		mockScanner.AssertScanned(t, len(data))
	})
}

func TestAPIKeyScan(t *testing.T) {
	t.Run("SuccessFilled", func(t *testing.T) {
		//setup
		data := []any{
			ulid.MakeSecure().String(), // ID
			"Description",              // Description
			"ClientID",                 // ClientID
			"Secret",                   // Secret
			time.Now(),                 // LastSeen
			time.Now(),                 // Created
			time.Now(),                 // Modified
		}
		mockScanner := &mock.MockScanner{}
		mockScanner.SetData(data)

		//test
		model := &models.APIKey{}
		err := model.Scan(mockScanner)
		require.NoError(t, err, "expected no errors when scanning")
		mockScanner.AssertScanned(t, len(data))
	})

	t.Run("SuccessNulls", func(t *testing.T) {
		//setup
		data := []any{
			ulid.MakeSecure().String(), // ID
			nil,                        // Description (testing null string)
			"ClientID",                 // ClientID
			"Secret",                   // Secret
			nil,                        // LastSeen (testing null time)
			time.Now(),                 // Created
			time.Time{},                // Modified (testing a zero value time)
		}
		mockScanner := &mock.MockScanner{}
		mockScanner.SetData(data)

		//test
		model := &models.APIKey{}
		err := model.Scan(mockScanner)
		require.NoError(t, err, "expected no errors when scanning")
		mockScanner.AssertScanned(t, len(data))
	})
}

func TestRoleScan(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		//setup
		data := []any{
			int64(808),    // ID
			"Title",       // Title
			"Description", // Description
			true,          // IsDefault
			time.Now(),    // Created
			time.Now(),    // Modified
		}
		mockScanner := &mock.MockScanner{}
		mockScanner.SetData(data)

		//test
		model := &models.Role{}
		err := model.Scan(mockScanner)
		require.NoError(t, err, "expected no errors when scanning")
		mockScanner.AssertScanned(t, len(data))
	})
}

func TestPermissionScan(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		//setup
		data := []any{
			int64(808),    // ID
			"Title",       // Title
			"Description", // Description
			time.Now(),    // Created
			time.Now(),    // Modified
		}
		mockScanner := &mock.MockScanner{}
		mockScanner.SetData(data)

		//test
		model := &models.Permission{}
		err := model.Scan(mockScanner)
		require.NoError(t, err, "expected no errors when scanning")
		mockScanner.AssertScanned(t, len(data))
	})
}

func TestResetPasswordLinkScan(t *testing.T) {
	t.Run("SuccessFilled", func(t *testing.T) {
		//setup
		data := []any{
			ulid.MakeSecure().String(), // ID
			ulid.MakeSecure().String(), // UserID
			"email@example.com",        // Email
			time.Now(),                 // Expiration
			nil,                        // Signature (vero token; ignored for now)
			time.Now(),                 // SentOn
			time.Now(),                 // Created
			time.Now(),                 // Modified
		}
		mockScanner := &mock.MockScanner{}
		mockScanner.SetData(data)

		//test
		model := &models.ResetPasswordLink{}
		err := model.Scan(mockScanner)
		require.NoError(t, err, "expected no errors when scanning")
		mockScanner.AssertScanned(t, len(data)-1) // will not scan Signature
	})

	t.Run("SuccessNulls", func(t *testing.T) {
		//setup
		data := []any{
			ulid.MakeSecure().String(), // ID
			ulid.MakeSecure().String(), // UserID
			"email@example.com",        // Email
			time.Now(),                 // Expiration
			nil,                        // Signature (vero token; ignored for now)
			nil,                        // SentOn (testing null time)
			time.Now(),                 // Created
			time.Now(),                 // Modified
		}
		mockScanner := &mock.MockScanner{}
		mockScanner.SetData(data)

		//test
		model := &models.ResetPasswordLink{}
		err := model.Scan(mockScanner)
		require.NoError(t, err, "expected no errors when scanning")
		mockScanner.AssertScanned(t, len(data)-1) // will not scan Signature
	})
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
