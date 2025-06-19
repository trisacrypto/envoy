package models_test

import (
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
	theModel := mock.GetSampleUser(true)

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
	theModel := mock.GetSampleAPIKey(true)

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
	theModel := mock.GetSampleRole(808, false)

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
	permissions, err := mock.GetSampleRole(808, true).Permissions()
	require.NotNil(t, permissions, "permissions should not be nil")
	require.Nil(t, err, "error should be nil")

	//test 2: no permissions
	permissions, err = mock.GetSampleRole(808, false).Permissions()
	require.Nil(t, permissions, "permissions should be nil")
	require.Error(t, err, "error should not be nil")
	require.Equal(t, errors.ErrMissingAssociation, err, "error should be ErrMissingAssociation")
}

func TestPermissionParams(t *testing.T) {
	// setup a model
	theModel := mock.GetSamplePermission(808)

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
	theModel := mock.GetSampleResetPasswordLink(true)

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

		// make sure scanned data matches the fields they were supposed to scan into
		require.Equal(t, data[0], model.ID.String(), "expected field ID to match data[0]")
		require.Equal(t, data[1], model.Name.String, "expected field Name to match data[1]")
		require.Equal(t, data[2], model.Email, "expected field Email to match data[2]")
		require.Equal(t, data[3], model.Password, "expected field Password to match data[3]")
		require.Equal(t, data[4], model.RoleID, "expected field RoleID to match data[4]")
		require.Equal(t, data[5], model.LastLogin.Time, "expected field LastLogin to match data[5]")
		require.Equal(t, data[6], model.Created, "expected field Created to match data[6]")
		require.Equal(t, data[7], model.Modified, "expected field Modified to match data[7]")
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

		// make sure scanned data matches the fields they were supposed to scan into
		require.Equal(t, data[0], model.ID.String(), "expected field ID to match data[0]")
		require.Equal(t, data[1], model.Description.String, "expected field Description to match data[1]")
		require.Equal(t, data[2], model.ClientID, "expected field ClientID to match data[2]")
		require.Equal(t, data[3], model.Secret, "expected field Secret to match data[3]")
		require.Equal(t, data[4], model.LastSeen.Time, "expected field LastSeen to match data[4]")
		require.Equal(t, data[5], model.Created, "expected field Created to match data[5]")
		require.Equal(t, data[6], model.Modified, "expected field Modified to match data[6]")
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

		// make sure scanned data matches the fields they were supposed to scan into
		require.Equal(t, data[0], model.ID, "expected field ID to match data[0]")
		require.Equal(t, data[1], model.Title, "expected field Title to match data[1]")
		require.Equal(t, data[2], model.Description, "expected field Description to match data[2]")
		require.Equal(t, data[3], model.IsDefault, "expected field IsDefault to match data[3]")
		require.Equal(t, data[4], model.Created, "expected field Created to match data[4]")
		require.Equal(t, data[5], model.Modified, "expected field Modified to match data[5]")
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

		// make sure scanned data matches the fields they were supposed to scan into
		require.Equal(t, data[0], model.ID, "expected field ID to match data[0]")
		require.Equal(t, data[1], model.Title, "expected field Title to match data[1]")
		require.Equal(t, data[2], model.Description, "expected field Description to match data[2]")
		require.Equal(t, data[3], model.Created, "expected field Created to match data[3]")
		require.Equal(t, data[4], model.Modified, "expected field Modified to match dat4[5]")
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

		// make sure scanned data matches the fields they were supposed to scan into
		require.Equal(t, data[0], model.ID.String(), "expected field ID to match data[0]")
		require.Equal(t, data[1], model.UserID.String(), "expected field UserID to match data[1]")
		require.Equal(t, data[2], model.Email, "expected field Email to match data[2]")
		require.Equal(t, data[3], model.Expiration, "expected field Expiration to match data[3]")
		require.Equal(t, data[5], model.SentOn.Time, "expected field SentOn to match data[5]")
		require.Equal(t, data[6], model.Created, "expected field Created to match data[6]")
		require.Equal(t, data[7], model.Modified, "expected field Modified to match data[7]")
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
