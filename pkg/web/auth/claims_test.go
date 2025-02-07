package auth_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/trisacrypto/envoy/pkg/store/models"
	. "github.com/trisacrypto/envoy/pkg/web/auth"

	jwt "github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/require"
	"go.rtnl.ai/ulid"
)

const (
	accessToken  = "eyJhbGciOiJSUzI1NiIsImtpZCI6IjAxR1g2NDdTOFBDVkJDUEpIWEdKUjI2UE42IiwidHlwIjoiSldUIn0.eyJpc3MiOiJodHRwOi8vMTI3LjAuMC4xIiwiYXVkIjpbImh0dHA6Ly8xMjcuMC4wLjEiXSwiZXhwIjoxNjgwNjE1MzMwLCJuYmYiOjE2ODA2MTE3MzAsImlhdCI6MTY4MDYxMTczMCwianRpIjoiMDFneDY0N3M4cGN2YmNwamh4Z2pzcG04N3AiLCJuYW1lIjoiSm9obiBEb2UiLCJlbWFpbCI6Impkb2VAZXhhbXBsZS5jb20iLCJvcmciOiIxMjMiLCJwcm9qZWN0IjoiYWJjIiwicGVybWlzc2lvbnMiOlsicmVhZDpkYXRhIiwid3JpdGU6ZGF0YSJdfQ.LLb6c2RdACJmoT3IFgJEwfu2_YJMcKgM2bF3ISF41A37gKTOkBaOe-UuTmjgZ7WEcuQ-cVkht0KI_4zqYYctB_WB9481XoNwff5VgFf3xrPdOYxS00YXQnl09RRqt6Fmca8nvd4mXfdO7uvpyNVuCIqNxBPXdSnRhreSoFB1GtFm42sBPAD7vF-MQUmU0c4PTsbiCfhR1_buH0NYEE1QFp3vYcgoiXOJHh9VStmRscqvLB12AQrcs26G9opdTCCORmvR2W3JLJ_hliHyp-d9lhXmCDFyiGkDEhTAUglqwBjqz5SO1UfAThWJO18PvZl4QPhb724oNT82VPh0DMDwfw"
	refreshToken = "eyJhbGciOiJSUzI1NiIsImtpZCI6IjAxR1g2NDdTOFBDVkJDUEpIWEdKUjI2UE42IiwidHlwIjoiSldUIn0.eyJpc3MiOiJodHRwOi8vMTI3LjAuMC4xIiwiYXVkIjpbImh0dHA6Ly8xMjcuMC4wLjEiLCJodHRwOi8vMTI3LjAuMC4xL3YxL3JlZnJlc2giXSwiZXhwIjoxNjgwNjE4OTMwLCJuYmYiOjE2ODA2MTQ0MzAsImlhdCI6MTY4MDYxMTczMCwianRpIjoiMDFneDY0N3M4cGN2YmNwamh4Z2pzcG04N3AifQ.CLHmtZwSPFCPoMBX06D_C3h3WuEonUbvbfWLvtmrMmIwnTwQ4hxsaRJo_a4qI-emp1HNg-yu_7c3VNwjkti-d0c7CAGApTaf5eRdGJ5HGUkI8RDHbbMFaOK86nAFnzdPJ2JLmGtLzvpF9eFXFllDhRiAB-2t0uKcOdN7cFghdwyWXIVJIJNjngF_WUFklmLKnqORtj_tA6UJ6NJnZln34eMGftAHbuH8x-xUiRePHnro4ydS43CKNOgRP8biMHiRR2broBz0apIt30TeQShaBSbmGx__LYdm7RKPJNVHAn_3h_PwwKQG567-Aqabg6TSmpwhXCk_RfUyQVGv2b997w"
)

func TestNewClaims(t *testing.T) {
	ctx := context.Background()

	t.Run("User", func(t *testing.T) {
		model := &models.User{
			Model: models.Model{
				ID: ulid.MustParse("01HVEH4E88XMYDXFAE4Y48CE9F"),
			},
			Name:   sql.NullString{Valid: true, String: "Carlos Hanger"},
			Email:  "carlos@example.com",
			RoleID: 2,
		}
		model.SetRole(&models.Role{ID: 2, Title: "editor"})
		model.SetPermissions([]string{"foo:manage", "foo:view", "foo:delete", "bar:view"})

		expected := &Claims{
			RegisteredClaims: jwt.RegisteredClaims{
				Subject: "u01HVEH4E88XMYDXFAE4Y48CE9F",
			},
			Name:         "Carlos Hanger",
			Email:        "carlos@example.com",
			Gravatar:     "https://www.gravatar.com/avatar/9f57a0b8bcfd77585fa2fd171455054e?d=mp&r=pg&s=80",
			Organization: "",
			Role:         "editor",
			Permissions:  []string{"foo:manage", "foo:view", "foo:delete", "bar:view"},
		}

		actual, err := NewClaims(ctx, model)
		require.NoError(t, err, "could not create claims for user")
		require.Equal(t, expected, actual, "created claims did not match expectation")
	})

	t.Run("APIKey", func(t *testing.T) {
		model := &models.APIKey{
			Model: models.Model{
				ID: ulid.MustParse("01HVEH4E88XMYDXFAE4Y48CE9F"),
			},
			ClientID: "abc1234",
		}
		model.SetPermissions([]string{"foo:manage", "foo:view", "foo:delete", "bar:view"})

		expected := &Claims{
			RegisteredClaims: jwt.RegisteredClaims{
				Subject: "k01HVEH4E88XMYDXFAE4Y48CE9F",
			},
			ClientID:    "abc1234",
			Permissions: []string{"foo:manage", "foo:view", "foo:delete", "bar:view"},
		}

		actual, err := NewClaims(ctx, model)
		require.NoError(t, err, "could not create claims for api key")
		require.Equal(t, expected, actual, "created claims did not match expectation")
	})

	t.Run("Invalid", func(t *testing.T) {
		claims, err := NewClaims(ctx, "foo")
		require.EqualError(t, err, "unknown model type string: cannot create claims")
		require.Nil(t, claims, "expected no claims returned on error")
	})
}

func TestNewClaimsWithOrganization(t *testing.T) {
	ctx := context.Background()
	SetOrganization("Rotational")
	defer SetOrganization("")

	t.Run("User", func(t *testing.T) {
		model := &models.User{
			Model: models.Model{
				ID: ulid.MustParse("01HVEH4E88XMYDXFAE4Y48CE9F"),
			},
			Name:   sql.NullString{Valid: true, String: "Laura Dewilder"},
			Email:  "laura@example.com",
			RoleID: 3,
		}
		model.SetRole(&models.Role{ID: 3, Title: "observer"})
		model.SetPermissions([]string{"foo:view", "bar:view"})

		expected := &Claims{
			RegisteredClaims: jwt.RegisteredClaims{
				Subject: "u01HVEH4E88XMYDXFAE4Y48CE9F",
			},
			Name:         "Laura Dewilder",
			Email:        "laura@example.com",
			Gravatar:     "https://www.gravatar.com/avatar/1d9fdac54add801f681bea83c05b13fb?d=mp&r=pg&s=80",
			Organization: "Rotational",
			Role:         "observer",
			Permissions:  []string{"foo:view", "bar:view"},
		}

		actual, err := NewClaims(ctx, model)
		require.NoError(t, err, "could not create claims for user")
		require.Equal(t, expected, actual, "created claims did not match expectation")
	})

	t.Run("APIKey", func(t *testing.T) {
		model := &models.APIKey{
			Model: models.Model{
				ID: ulid.MustParse("01HVEH4E88XMYDXFAE4Y48CE9F"),
			},
			ClientID: "abc1234",
		}

		actual, err := NewClaims(ctx, model)
		require.NoError(t, err, "could not create claims for api key")
		require.Empty(t, actual.Organization, "api keys should not have organization claims")
	})

	t.Run("GetOrganization", func(t *testing.T) {
		require.Equal(t, "Rotational", GetOrganization())
	})
}

func TestSubjectType(t *testing.T) {
	id := ulid.MustParse("01HVEH4E88XMYDXFAE4Y48CE9F")

	t.Run("User", func(t *testing.T) {
		claims := &Claims{}
		claims.SetSubjectID(SubjectUser, id)
		require.Equal(t, "u01HVEH4E88XMYDXFAE4Y48CE9F", claims.Subject)

		sub, pid, err := claims.SubjectID()
		require.NoError(t, err)
		require.Equal(t, pid, id)
		require.Equal(t, SubjectUser, sub)
	})

	t.Run("APIKey", func(t *testing.T) {
		claims := &Claims{}
		claims.SetSubjectID(SubjectAPIKey, id)
		require.Equal(t, "k01HVEH4E88XMYDXFAE4Y48CE9F", claims.Subject)

		sub, pid, err := claims.SubjectID()
		require.NoError(t, err)
		require.Equal(t, pid, id)
		require.Equal(t, SubjectAPIKey, sub)
	})

	t.Run("Unkown", func(t *testing.T) {
		claims := &Claims{}
		claims.SetSubjectID(SubjectType('b'), id)
		require.Equal(t, "b01HVEH4E88XMYDXFAE4Y48CE9F", claims.Subject)

		sub, pid, err := claims.SubjectID()
		require.NoError(t, err)
		require.Equal(t, pid, id)
		require.Equal(t, SubjectType('b'), sub)
	})
}

func TestClaimsHasPermission(t *testing.T) {
	claims := &Claims{
		Permissions: []string{"foo:manage", "foo:view", "foo:delete", "bar:view"},
	}

	for _, permission := range []string{"foo:manage", "foo:view", "foo:delete", "bar:view"} {
		require.True(t, claims.HasPermission(permission), "expected claims to have permission %q", permission)
	}

	for _, permission := range []string{"", "bar:manage", "bar:delete", "FOO:VIEW"} {
		require.False(t, claims.HasPermission(permission), "expected claims to not have permisison %q", permission)
	}
}

func TestClaimsHasAllPermissions(t *testing.T) {
	claims := &Claims{
		Permissions: []string{"foo:manage", "foo:view", "foo:delete", "bar:view"},
	}

	tests := []struct {
		required []string
		assert   require.BoolAssertionFunc
	}{
		{
			[]string{},
			require.False,
		},
		{
			[]string{"foo:view", "bar:manage"},
			require.False,
		},
		{
			[]string{"foo:manage", "foo:view", "foo:delete", "bar:manage"},
			require.False,
		},
		{
			[]string{"foo:view"},
			require.True,
		},
		{
			[]string{"bar:view"},
			require.True,
		},
		{
			[]string{"foo:manage", "foo:view", "foo:delete", "bar:view"},
			require.True,
		},
		{
			[]string{"foo:view", "foo:delete"},
			require.True,
		},
	}

	for i, tc := range tests {
		tc.assert(t, claims.HasAllPermissions(tc.required...), "test case %d failed", i)
	}
}

func TestParse(t *testing.T) {
	accessClaims, err := ParseUnverified(accessToken)
	require.NoError(t, err, "could not parse access token")

	refreshClaims, err := ParseUnverified(refreshToken)
	require.NoError(t, err, "could not parse refresh token")

	// We expect the claims and refresh tokens to have the same ID
	require.Equal(t, accessClaims.ID, refreshClaims.ID, "access and refresh token had different IDs or the parse was unsuccessful")

	// Check that an error is returned when parsing a bad token
	_, err = ParseUnverified("notarealtoken")
	require.Error(t, err, "should not be able to parse a bad token")
}

func TestExpiresAt(t *testing.T) {
	expiration, err := ExpiresAt(accessToken)
	require.NoError(t, err, "could not parse access token")

	// Expect the time to be fetched correctly from the token
	expected := time.Date(2023, 4, 4, 13, 35, 30, 0, time.UTC)
	require.True(t, expected.Equal(expiration))

	// Check that an error is returned when parsing a bad token
	_, err = ExpiresAt("notarealtoken")
	require.Error(t, err, "should not be able to parse a bad token")
}

func TestNotBefore(t *testing.T) {
	expiration, err := NotBefore(refreshToken)
	require.NoError(t, err, "could not parse access token")

	// Expect the time to be fetched correctly from the token
	expected := time.Date(2023, 4, 4, 13, 20, 30, 0, time.UTC)
	require.True(t, expected.Equal(expiration))

	// Check that an error is returned when parsing a bad token
	_, err = NotBefore("notarealtoken")
	require.Error(t, err, "should not be able to parse a bad token")
}
