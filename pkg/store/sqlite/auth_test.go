package sqlite_test

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/mock"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"go.rtnl.ai/ulid"
)

func (s *storeTestSuite) TestListUsers() {
	s.Run("Success", func() {
		//setup
		require := s.Require()
		ctx := context.Background()

		//test
		users, err := s.store.ListUsers(ctx, &models.UserPageInfo{})
		require.NoError(err, "expected no errors")
		require.NotNil(users.Users, "there were no users")
		require.Len(users.Users, 3, fmt.Sprintf("there should be 3 users, but there were %d", len(users.Users)))
	})

	s.Run("FailureNilPageInfo", func() {
		//setup
		require := s.Require()
		ctx := context.Background()

		//test
		require.Panics(func() { s.store.ListUsers(ctx, nil) }, "should panic with nil page info")
	})
}

func (s *storeTestSuite) TestCreateUser() {
	s.Run("Success", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		user := mock.GetSampleUser(true)
		user.ID = ulid.Zero

		//test
		err := s.store.CreateUser(ctx, user)
		require.NoError(err, "no error was expected")

		user2, err := s.store.RetrieveUser(ctx, user.ID)
		require.NoError(err, "expected no error")
		require.NotNil(user2, "user should not be nil")
		require.Equal(user.ID, user2.ID, fmt.Sprintf("user ID should be %s, found %s instead", user.ID, user2.ID))
	})

	s.Run("FailureNonZeroID", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		user := mock.GetSampleUser(true)

		//test
		err := s.store.CreateUser(ctx, user)
		require.Error(err, "an error was expected")
		require.Equal(errors.ErrNoIDOnCreate, err, "expected an ErrNoIDOnCreate error")
	})

	s.Run("FailureNotFoundRoleID", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		user := mock.GetSampleUser(true)
		user.ID = ulid.Zero

		user.RoleID = 808

		//test
		err := s.store.CreateUser(ctx, user)
		require.Error(err, "an error was expected")
		// TODO: (ticket sc-32339) this currently returns an ErrAlreadyExists
		// instead of an ErrNotFound as would be logical, because in the `dbe()`
		// function we return an ErrAlreadyExists for any SQLite constraint error
		require.Equal(errors.ErrAlreadyExists, err, "expected an ErrAlreadyExists error")
	})
}

func (s *storeTestSuite) TestRetrieveUser() {
	s.Run("SuccessID", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		userId := ulid.MustParse("01HWQE3N4S6PZGKNCH7E617N8T")

		//test
		user, err := s.store.RetrieveUser(ctx, userId)
		require.NoError(err, "expected no error")
		require.NotNil(user, "user should not be nil")
		require.Equal(userId, user.ID, fmt.Sprintf("user ID should be %s, found %s instead", userId, user.ID))
	})

	s.Run("SuccessEmail", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		email := "observer@example.com"
		userId := ulid.MustParse("01HWQE3N4S6PZGKNCH7E617N8T")

		//test
		user, err := s.store.RetrieveUser(ctx, email)
		require.NoError(err, "expected no error")
		require.NotNil(user, "user should not be nil")
		require.Equal(email, user.Email, fmt.Sprintf("user email should be %s, found %s instead", email, user.Email))
		require.Equal(userId, user.ID, fmt.Sprintf("user ID should be %s, found %s instead", userId, user.ID))
	})

	s.Run("FailureNotFoundID", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		userId := ulid.MakeSecure()

		//test
		user, err := s.store.RetrieveUser(ctx, userId)
		require.Error(err, "expected an error")
		require.Equal(errors.ErrNotFound, err, "expected ErrNotFound")
		require.Nil(user, "user should be nil")
	})

	s.Run("FailureNotFoundEmail", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		email := "this_user_does_not_exist@example.com"

		//test
		user, err := s.store.RetrieveUser(ctx, email)
		require.Error(err, "expected an error")
		require.Equal(errors.ErrNotFound, err, "expected ErrNotFound")
		require.Nil(user, "user should be nil")
	})

	s.Run("FailureUnknownType", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		wrongType := int64(808)

		//test
		user, err := s.store.RetrieveUser(ctx, wrongType)
		require.Error(err, "expected an error")
		require.Nil(user, "user should be nil")
		require.ErrorContains(err, "unknown type", "expected an error starting with 'unknown type...'")
	})

	s.Run("FailureNilType", func() {
		//setup
		require := s.Require()
		ctx := context.Background()

		//test
		user, err := s.store.RetrieveUser(ctx, nil)
		require.Error(err, "expected an error")
		require.Nil(user, "user should be nil")
		require.ErrorContains(err, "unknown type", "expected an error starting with 'unknown type...'")
	})
}

func (s *storeTestSuite) TestUpdateUser() {
	s.Run("Success", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		userId := ulid.MustParse("01HWQE3N4S6PZGKNCH7E617N8T")
		user, err := s.store.RetrieveUser(ctx, userId)
		require.NoError(err, "expected no error")
		require.NotNil(user, "user should not be nil")

		newEmail := "here_is_the_new_email@example.com"
		user.Email = newEmail

		//test
		err = s.store.UpdateUser(ctx, user)
		require.NoError(err, "expected no error")
		require.NotNil(user, "user should not be nil")

		user2, err := s.store.RetrieveUser(ctx, userId)
		require.NoError(err, "expected no error")
		require.NotNil(user2, "user should not be nil")
		require.Equal(newEmail, user2.Email, fmt.Sprintf("expected email %s, got email %s", newEmail, user2.Email))
	})

	s.Run("FailureNotFound", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		userId := ulid.MustParse("01HWQE3N4S6PZGKNCH7E617N8T")
		user, err := s.store.RetrieveUser(ctx, userId)
		require.NoError(err, "expected no error")
		require.NotNil(user, "user should not be nil")

		user.ID = ulid.MakeSecure()

		//test
		err = s.store.UpdateUser(ctx, user)
		require.Error(err, "expected an error")
		require.Equal(errors.ErrNotFound, err, "expected ErrNotFound")
	})
}

func (s *storeTestSuite) TestSetUserPassword() {
	s.Run("Success", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		userId := ulid.MustParse("01HWQE3N4S6PZGKNCH7E617N8T")
		newPassword := "password_1234"

		//test
		err := s.store.SetUserPassword(ctx, userId, newPassword)
		require.NoError(err, "expected no error")

		user, err := s.store.RetrieveUser(ctx, userId)
		require.NoError(err, "expected no error")
		require.NotNil(user, "user should not be nil")
		require.Equal(newPassword, user.Password, "expected the password to be the new one")
	})

	s.Run("FailureNotFoundRandomID", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		userId := ulid.MakeSecure()
		newPassword := "password_1234"

		//test
		err := s.store.SetUserPassword(ctx, userId, newPassword)
		require.Error(err, "expected an error")
		require.Equal(errors.ErrNotFound, err, "expected ErrNotFound")
	})

	s.Run("FailureNotFoundZeroID", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		userId := ulid.Zero
		newPassword := "password_1234"

		//test
		err := s.store.SetUserPassword(ctx, userId, newPassword)
		require.Error(err, "expected an error")
		require.Equal(errors.ErrNotFound, err, "expected ErrNotFound")
	})
}

func (s *storeTestSuite) TestSetUserLastLogin() {
	s.Run("SuccessNow", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		userId := ulid.MustParse("01HWQE3N4S6PZGKNCH7E617N8T")
		newTime := time.Now()

		//test
		err := s.store.SetUserLastLogin(ctx, userId, newTime)
		require.NoError(err, "expected no error")

		user, err := s.store.RetrieveUser(ctx, userId)
		require.NoError(err, "expected no error")
		require.NotNil(user, "user should not be nil")
		require.True(newTime.Equal(user.LastLogin.Time), "expected the last login time to be the new one")
		require.True(user.LastLogin.Valid, "last login time was invalid")
	})

	s.Run("SuccessZero", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		userId := ulid.MustParse("01HWQE3N4S6PZGKNCH7E617N8T")
		newTime := time.Time{}

		//test
		err := s.store.SetUserLastLogin(ctx, userId, newTime)
		require.NoError(err, "expected no error")

		user, err := s.store.RetrieveUser(ctx, userId)
		require.NoError(err, "expected no error")
		require.NotNil(user, "user should not be nil")
		require.True(newTime.Equal(user.LastLogin.Time), "expected the last login time to be the new one")
		require.False(user.LastLogin.Valid, "last login time was valid")
	})

	s.Run("FailureNotFoundRandomID", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		userId := ulid.MakeSecure()
		newTime := time.Now()

		//test
		err := s.store.SetUserLastLogin(ctx, userId, newTime)
		require.Error(err, "expected no error")
		require.Equal(errors.ErrNotFound, err, "expected ErrNotFound")
	})

	s.Run("FailureNotFoundZeroID", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		userId := ulid.Zero
		newTime := time.Now()

		//test
		err := s.store.SetUserLastLogin(ctx, userId, newTime)
		require.Error(err, "expected no error")
		require.Equal(errors.ErrNotFound, err, "expected ErrNotFound")
	})
}

func (s *storeTestSuite) TestDeleteUser() {
	s.Run("Success", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		userId := ulid.MustParse("01HWQE3N4S6PZGKNCH7E617N8T")

		//test
		user, err := s.store.RetrieveUser(ctx, userId)
		require.NoError(err, "expected no error")
		require.NotNil(user, "user should not be nil")

		err = s.store.DeleteUser(ctx, userId)
		require.NoError(err, "expected no error")

		user, err = s.store.RetrieveUser(ctx, userId)
		require.Error(err, "expected an error")
		require.Equal(errors.ErrNotFound, err, "expected user to be missing")
		require.Nil(user, "user should be nil")
	})

	s.Run("FailureNotFound", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		userId := ulid.MakeSecure()

		//test
		err := s.store.DeleteUser(ctx, userId)
		require.Error(err, "expected no error")
		require.Equal(errors.ErrNotFound, err, "expected user to be missing")
	})
}

func (s *storeTestSuite) TestLookupRole() {
	s.Run("SuccessCases", func() {
		type roleSuccessTestCase struct {
			RoleName string
			RoleId   int64
		}
		cases := []roleSuccessTestCase{
			{
				RoleName: "Admin",
				RoleId:   int64(1),
			},
			{
				RoleName: "Compliance",
				RoleId:   int64(2),
			},
			{
				RoleName: "Observer",
				RoleId:   int64(3),
			},
		}

		for i := range cases {
			s.Run("Success"+cases[i].RoleName, func() {
				//setup
				require := s.Require()
				ctx := context.Background()

				//test
				role, err := s.store.LookupRole(ctx, cases[i].RoleName)
				require.NoError(err, "expected no error")
				require.NotNil(role, "role should not be nil")
				require.Equal(cases[i].RoleId, role.ID, fmt.Sprintf("role ID should be %d, found %d instead", cases[i].RoleId, role.ID))
			})

		}
	})

	s.Run("FailureNotFound", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		roleName := "not_a_role_name_for_sure"

		//test
		role, err := s.store.LookupRole(ctx, roleName)
		require.Error(err, "expected an error")
		require.Equal(errors.ErrNotFound, err, "expected ErrNotFound")
		require.Nil(role, "role should be nil")
	})
}

func (s *storeTestSuite) TestListAPIKeys() {
	s.Run("Success", func() {
		//setup
		require := s.Require()
		ctx := context.Background()

		//test
		apiKeys, err := s.store.ListAPIKeys(ctx, &models.PageInfo{})
		require.NoError(err, "expected no errors")
		require.NotNil(apiKeys.APIKeys, "there were no api keys")
		require.Len(apiKeys.APIKeys, 2, fmt.Sprintf("there should be 2 api keys, but there were %d", len(apiKeys.APIKeys)))
	})

	s.Run("SuccessNilPageInfo", func() {
		//setup
		require := s.Require()
		ctx := context.Background()

		//test
		apiKeys, err := s.store.ListAPIKeys(ctx, nil)
		require.NoError(err, "expected no errors")
		require.NotNil(apiKeys.APIKeys, "there were no api keys")
		require.Len(apiKeys.APIKeys, 2, fmt.Sprintf("there should be 2 api keys, but there were %d", len(apiKeys.APIKeys)))
	})
}

func (s *storeTestSuite) TestCreateAPIKey() {
	s.Run("SuccessNoRoles", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		apiKey := mock.GetSampleAPIKey(true)
		apiKey.ID = ulid.Zero

		//test
		err := s.store.CreateAPIKey(ctx, apiKey)
		require.NoError(err, "expected no errors")

		apiKey2, err := s.store.RetrieveAPIKey(ctx, apiKey.ClientID)
		require.NoError(err, "expected no errors")
		require.NotNil(apiKey2, "api key should not be nil")
		require.Equal(apiKey.ClientID, apiKey2.ClientID, fmt.Sprintf("client id should be %s, but found %s", apiKey.ClientID, apiKey2.ClientID))
	})

	s.Run("SuccessAllRoles", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		apiKey := mock.GetSampleAPIKey(true)
		apiKey.ID = ulid.Zero
		apiKey.SetPermissions([]string{
			"users:manage",
			"users:view",
			"apikeys:manage",
			"apikeys:view",
			"apikeys:revoke",
			"counterparties:manage",
			"counterparties:view",
			"accounts:manage",
			"accounts:view",
			"travelrule:manage",
			"travelrule:delete",
			"travelrule:view",
			"config:manage",
			"config:view",
			"pki:manage",
			"pki:delete",
			"pki:view",
		})

		//test
		err := s.store.CreateAPIKey(ctx, apiKey)
		require.NoError(err, "expected no errors")

		apiKey2, err := s.store.RetrieveAPIKey(ctx, apiKey.ClientID)
		require.NoError(err, "expected no errors")
		require.NotNil(apiKey2, "api key should not be nil")
		require.Equal(apiKey.ClientID, apiKey2.ClientID, fmt.Sprintf("client id should be %s, but found %s", apiKey.ClientID, apiKey2.ClientID))

	})

	s.Run("FailureNotZeroID", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		apiKey := mock.GetSampleAPIKey(true)

		//test
		err := s.store.CreateAPIKey(ctx, apiKey)
		require.Error(err, "expected an error")
		require.Equal(errors.ErrNoIDOnCreate, err, "expected ErrNoIDOnCreate")
	})

	s.Run("FailureBadPermissionID", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		apiKey := mock.GetSampleAPIKey(true)
		apiKey.ID = ulid.Zero
		apiKey.SetPermissions([]string{"permission_that_doesn't_exist"})

		//test
		err := s.store.CreateAPIKey(ctx, apiKey)
		require.Error(err, "expected an error")
		// TODO: (ticket sc-32339) this currently returns an ErrAlreadyExists
		// instead of an ErrNotFound as would be logical, because in the `dbe()`
		// function we return an ErrAlreadyExists for any SQLite constraint error
		require.Equal(errors.ErrAlreadyExists, err, "expected ErrAlreadyExists")
	})
}

func (s *storeTestSuite) TestRetrieveAPIKey() {
	s.Run("SuccessKeyID", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		keyId := ulid.MustParse("01HWQEJJDMS5EKNARHPJEDMHA4")

		//test
		apiKey, err := s.store.RetrieveAPIKey(ctx, keyId)
		require.NoError(err, "expected no errors")
		require.NotNil(apiKey, "api key should not be nil")
		require.Equal(keyId, apiKey.ID, fmt.Sprintf("id should be %s, but found %s", keyId, apiKey.ID))
		require.Equal("Full permissions keys", apiKey.Description.String, "expected a different api key description")

		permissions := apiKey.Permissions()
		require.NotNil(permissions, "permissions should not be nil")
		require.Len(permissions, 17, fmt.Sprintf("expected 17 permissions, got %d", len(permissions)))
	})

	s.Run("SuccessClientID", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		clientId := "ISoIuDiGkpVpAyCrLGYrKU"

		//test
		apiKey, err := s.store.RetrieveAPIKey(ctx, clientId)
		require.NoError(err, "expected no errors")
		require.NotNil(apiKey, "api key should not be nil")
		require.Equal(clientId, apiKey.ClientID, fmt.Sprintf("client id should be %s, but found %s", clientId, apiKey.ClientID))
		require.Equal("Full permissions keys", apiKey.Description.String, "expected a different api key description")

		permissions := apiKey.Permissions()
		require.NotNil(permissions, "permissions should not be nil")
		require.Len(permissions, 17, fmt.Sprintf("expected 17 permissions, got %d", len(permissions)))
	})

	s.Run("FailureNotFound", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		clientId := "this_is_not_a_client_id"

		//test
		apiKey, err := s.store.RetrieveAPIKey(ctx, clientId)
		require.Error(err, "expected an error")
		require.Equal(errors.ErrNotFound, err, "expected ErrNotFound")
		require.Nil(apiKey, "api key should be nil")
	})

	s.Run("FailureInvalidType", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		invalidType := int64(808)

		//test
		apiKey, err := s.store.RetrieveAPIKey(ctx, invalidType)
		require.Error(err, "expected an error")
		require.ErrorContains(err, "unkown type", "expected 'unknown type' error")
		require.Nil(apiKey, "api key should be nil")
	})
}

func (s *storeTestSuite) TestUpdateAPIKey() {
	s.Run("Success", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		keyId := ulid.MustParse("01HWQEJJDMS5EKNARHPJEDMHA4")
		apiKey, err := s.store.RetrieveAPIKey(ctx, keyId)
		require.NoError(err, "expected no errors")
		require.NotNil(apiKey, "api key should not be nil")

		newDescription := sql.NullString{String: "New Description 83r29123e89", Valid: true}
		apiKey.Description = newDescription

		//test
		err = s.store.UpdateAPIKey(ctx, apiKey)
		require.NoError(err, "expected no errors")

		apiKey, err = s.store.RetrieveAPIKey(ctx, keyId)
		require.NoError(err, "expected no errors")
		require.NotNil(apiKey, "api key should not be nil")
		require.Equal(newDescription, apiKey.Description, "expected the new description")

		permissions := apiKey.Permissions()
		require.NotNil(permissions, "permissions should not be nil")
		require.Len(permissions, 17, fmt.Sprintf("expected 17 permissions, got %d", len(permissions)))

	})

	s.Run("FailureNotFoundRandomID", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		apiKey := mock.GetSampleAPIKey(true)

		//test
		err := s.store.UpdateAPIKey(ctx, apiKey)
		require.Error(err, "expected an error")
		require.Equal(errors.ErrNotFound, err, "expected ErrNotFound")
	})

	s.Run("FailureNotFoundZeroID", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		apiKey := mock.GetSampleAPIKey(true)
		apiKey.ID = ulid.Zero

		//test
		err := s.store.UpdateAPIKey(ctx, apiKey)
		require.Error(err, "expected an error")
		require.Equal(errors.ErrNotFound, err, "expected ErrNotFound")
	})
}

func (s *storeTestSuite) TestDeleteAPIKey() {
	s.Run("Success", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		keyId := ulid.MustParse("01HWQEJJDMS5EKNARHPJEDMHA4")

		//test
		err := s.store.DeleteAPIKey(ctx, keyId)
		require.NoError(err, "expected no error")

		apiKey, err := s.store.RetrieveAPIKey(ctx, keyId)
		require.Error(err, "expected an error")
		require.Equal(errors.ErrNotFound, err, "expected ErrNotFound")
		require.Nil(apiKey, "api key should be nil")
	})

	s.Run("FailureNotFound", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		keyId := ulid.MakeSecure()

		//test
		err := s.store.DeleteAPIKey(ctx, keyId)
		require.Error(err, "expected an error")
		require.Equal(errors.ErrNotFound, err, "expected ErrNotFound")
	})
}

func (s *storeTestSuite) TestListResetPasswordLinks() {
	s.Run("Success", func() {
		//setup
		require := s.Require()
		ctx := context.Background()

		//test
		links, err := s.store.ListResetPasswordLinks(ctx, &models.PageInfo{})
		require.NoError(err, "expected no errors")
		require.NotNil(links.Links, "there were no links")
		require.Len(links.Links, 1, fmt.Sprintf("there should be 1 link, but there was %d", len(links.Links)))
	})

	s.Run("SuccessNilPageInfo", func() {
		//setup
		require := s.Require()
		ctx := context.Background()

		//test
		links, err := s.store.ListResetPasswordLinks(ctx, nil)
		require.NoError(err, "expected no errors")
		require.NotNil(links.Links, "there were no links")
		require.Len(links.Links, 1, fmt.Sprintf("there should be 1 link, but there was %d", len(links.Links)))
	})
}

func (s *storeTestSuite) TestCreateResetPasswordLink() {
	s.Run("SuccessAddLink", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		link := mock.GetSampleResetPasswordLink(true)
		link.ID = ulid.Zero
		link.UserID = ulid.MustParse("01HWQE29RW1S1D8ZN58M528A1M")

		//test
		links, err := s.store.ListResetPasswordLinks(ctx, &models.PageInfo{})
		require.NoError(err, "expected no errors")
		require.NotNil(links.Links, "there were no links")
		beforeLength := len(links.Links) // there should be 1 more after

		err = s.store.CreateResetPasswordLink(ctx, link)
		require.NoError(err, "expected no errors")

		links, err = s.store.ListResetPasswordLinks(ctx, &models.PageInfo{})
		require.NoError(err, "expected no errors")
		require.NotNil(links.Links, "there were no links")
		require.Len(links.Links, beforeLength+1, fmt.Sprintf("there should be %d links, but there was %d", beforeLength+1, len(links.Links)))
	})

	s.Run("SuccessOverwriteLink", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		link := mock.GetSampleResetPasswordLink(true)
		link.ID = ulid.Zero
		link.UserID = ulid.MustParse("01HWQE3N4S6PZGKNCH7E617N8T")

		//test
		links, err := s.store.ListResetPasswordLinks(ctx, &models.PageInfo{})
		require.NoError(err, "expected no errors")
		require.NotNil(links.Links, "there were no links")
		beforeLength := len(links.Links) // there should be the same number before as after

		err = s.store.CreateResetPasswordLink(ctx, link)
		require.NoError(err, "expected no errors")

		links, err = s.store.ListResetPasswordLinks(ctx, &models.PageInfo{})
		require.NoError(err, "expected no errors")
		require.NotNil(links.Links, "there were no links")
		require.Len(links.Links, beforeLength, fmt.Sprintf("there should be %d link, but there was %d", beforeLength, len(links.Links)))
	})

	s.Run("FailureNotFound", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		link := mock.GetSampleResetPasswordLink(true)
		link.ID = ulid.Zero

		//test
		err := s.store.CreateResetPasswordLink(ctx, link)
		require.Error(err, "expected an error")
		// TODO: (ticket sc-32339) this currently returns an ErrAlreadyExists
		// instead of an ErrNotFound as would be logical, because in the `dbe()`
		// function we return an ErrAlreadyExists for any SQLite constraint error
		require.Equal(errors.ErrAlreadyExists, err, "expected ErrAlreadyExists")
	})
}

func (s *storeTestSuite) TestRetrieveResetPasswordLink() {
	s.Run("Success", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		linkId := ulid.MustParse("01JXTGSFRC88HAY8V173976Z9D")

		//test
		link, err := s.store.RetrieveResetPasswordLink(ctx, linkId)
		require.NoError(err, "expected no errors")
		require.NotNil(link, "there was no link")
		require.Equal("observer@example.com", link.Email, fmt.Sprintf("expected 'observer@example.com' email, got '%s'", link.Email))
	})

	s.Run("FailureNotFound", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		linkId := ulid.MakeSecure()

		//test
		link, err := s.store.RetrieveResetPasswordLink(ctx, linkId)
		require.Error(err, "expected an error")
		require.Error(errors.ErrNotFound, err, "expected ErrNotFound")
		require.Nil(link, "expected a nil link")
	})
}

func (s *storeTestSuite) TestUpdateResetPasswordLink() {
	s.Run("Success", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		linkId := ulid.MustParse("01JXTGSFRC88HAY8V173976Z9D")
		link, err := s.store.RetrieveResetPasswordLink(ctx, linkId)
		require.NoError(err, "expected no errors")
		require.NotNil(link, "there was no link")

		oldEmail := link.Email
		link.Email = "new_email_addy@example.com"
		newTime := sql.NullTime{Time: time.Now(), Valid: true}
		link.SentOn = newTime

		//test
		err = s.store.UpdateResetPasswordLink(ctx, link)
		require.NoError(err, "expected no error")

		link, err = s.store.RetrieveResetPasswordLink(ctx, linkId)
		require.NoError(err, "expected no errors")
		require.NotNil(link, "there was no link")
		require.True(newTime.Valid, "expected valid new time")
		require.True(newTime.Time.Equal(link.SentOn.Time), "expected the new sent on time")
		require.Equal(oldEmail, link.Email, "expected the email to remain the same")
	})

	s.Run("FailureNotFound", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		linkId := ulid.MustParse("01JXTGSFRC88HAY8V173976Z9D")
		link, err := s.store.RetrieveResetPasswordLink(ctx, linkId)
		require.NoError(err, "expected no errors")
		require.NotNil(link, "there was no link")

		link.ID = ulid.MakeSecure()

		//test
		err = s.store.UpdateResetPasswordLink(ctx, link)
		require.Error(err, "expected an error")
		require.Equal(errors.ErrNotFound, err, "expected ErrNotFound")
	})
}

func (s *storeTestSuite) TestDeleteResetPasswordLink() {
	s.Run("Success", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		linkId := ulid.MustParse("01JXTGSFRC88HAY8V173976Z9D")

		//test
		err := s.store.DeleteResetPasswordLink(ctx, linkId)
		require.NoError(err, "expected no error")

		link, err := s.store.RetrieveResetPasswordLink(ctx, linkId)
		require.Error(err, "expected an error")
		require.Equal(errors.ErrNotFound, err, "expected ErrNotFound")
		require.Nil(link, "expected a nil link")
	})

	s.Run("FailureNotFound", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		linkId := ulid.MakeSecure()

		//test
		err := s.store.DeleteResetPasswordLink(ctx, linkId)
		require.Error(err, "expected an error")
		require.Equal(errors.ErrNotFound, err, "expected ErrNotFound")
	})
}
