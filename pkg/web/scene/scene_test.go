package scene_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg"
	"github.com/trisacrypto/envoy/pkg/config"
	"github.com/trisacrypto/envoy/pkg/web/api/v1"
	"github.com/trisacrypto/envoy/pkg/web/auth"
	"github.com/trisacrypto/envoy/pkg/web/scene"
)

func TestNew(t *testing.T) {
	t.Run("NilContext", func(t *testing.T) {
		data := scene.New(nil)
		require.NotEmpty(t, data, "expected scene to be returned")
		CheckVersion(t, data)
	})

	t.Run("ContextNoClaims", func(t *testing.T) {
		data := scene.New(CreateContext())
		require.NotEmpty(t, data, "expected scene to be returned")
		CheckVersion(t, data)

		require.Contains(t, data, scene.Page, "expected the page to be in the scene")
		require.Equal(t, "/dashboard", data[scene.Page], "unexpected page value")

		require.Contains(t, data, scene.IsAuthenticated, "expected is authenticated to be in the scene")
		require.False(t, data[scene.IsAuthenticated].(bool), "expected is authenticated to be false")

		require.Contains(t, data, scene.User, "expected user to be in the scene")
		require.Nil(t, data[scene.User], "expected user to be nil")
	})

	t.Run("ContextClaims", func(t *testing.T) {
		data := scene.New(CreateUserContext(nil))
		require.NotEmpty(t, data, "expected scene to be returned")
		CheckVersion(t, data)

		require.Contains(t, data, scene.Page, "expected the page to be in the scene")
		require.Equal(t, "/dashboard", data[scene.Page], "unexpected page value")

		require.Contains(t, data, scene.IsAuthenticated, "expected is authenticated to be in the scene")
		require.True(t, data[scene.IsAuthenticated].(bool), "expected is authenticated to be true")

		require.Contains(t, data, scene.User, "expected user to be in the scene")
		require.NotNil(t, data[scene.User], "expected user to be in the scene")
	})

	t.Run("Configuration", func(t *testing.T) {
		t.Run("None", func(t *testing.T) {
			data := scene.New(CreateContext())
			require.NotContains(t, data, scene.SunriseEnabled, "expected no config info without it being explicitly set")
			require.NotContains(t, data, scene.DaybreakEnabled, "expected no config info without it being explicitly set")
		})

		t.Run("WithConf", func(t *testing.T) {
			conf, err := CreateConf(t)
			require.NoError(t, err, "could not create configuration")

			scene.WithConf(&conf)

			data := scene.New(CreateContext())

			require.Contains(t, data, scene.SunriseEnabled, "expected sunrise enabled to be set with conf")
			require.False(t, data[scene.SunriseEnabled].(bool), "expected sunrise enabled to default to false because email is unavailable by default")

			require.Contains(t, data, scene.DaybreakEnabled, "expected daybreak enabled to be set with conf")
			require.False(t, data[scene.DaybreakEnabled].(bool), "expected daybreak enabled to default to false")
		})

	})
}

func TestUpdate(t *testing.T) {
	alpha := scene.New(nil)
	alpha["Fruit"] = "Orange"
	alpha["Age"] = 42

	bravo := scene.Scene{
		scene.Version:      "0.0.1-alpha.4",
		scene.ShortVersion: "0.0.1",
		"Fruit":            "Orange",
		"Name":             "Roger",
	}

	// Assert original
	require.Len(t, alpha, 4)
	require.Len(t, bravo, 4)

	// Update alpha from bravo
	alpha.Update(bravo)
	require.Len(t, alpha, 5)
	require.Len(t, bravo, 4)

	// Check the update happened correctly
	expected := scene.Scene{
		scene.Version:      "0.0.1-alpha.4",
		scene.ShortVersion: "0.0.1",
		"Fruit":            "Orange",
		"Name":             "Roger",
		"Age":              42,
	}
	require.Equal(t, expected, alpha)
}

func TestIsAuthenticated(t *testing.T) {
	testCases := []struct {
		c      *gin.Context
		assert require.BoolAssertionFunc
	}{
		{nil, require.False},
		{CreateContext(), require.False},
		{CreateUserContext(nil), require.True},
	}

	for i, tc := range testCases {
		data := scene.New(tc.c)
		tc.assert(t, data.IsAuthenticated(), "test case %d failed", i)
	}
}

func TestUser(t *testing.T) {
	t.Run("NoUser", func(t *testing.T) {
		data := scene.New(nil)
		require.Nil(t, data.GetUser(), "expected user to be nil")
		require.False(t, data.HasRole("Compliance"), "expected user to have no role")
		require.False(t, data.IsAdmin(), "expected no user to not be admin")
		require.True(t, data.IsViewOnly(), "expected no user to have no view only")
	})

	t.Run("User", func(t *testing.T) {
		data := scene.New(CreateUserContext(nil))
		require.NotNil(t, data.GetUser(), "expected user to not be nil")
		require.True(t, data.HasRole("Compliance"), "expected user to have no role")
		require.False(t, data.IsAdmin(), "expected no user to not be admin")
		require.False(t, data.IsViewOnly(), "expected no user to have no view only")
	})

	t.Run("Admin", func(t *testing.T) {
		data := scene.New(CreateUserContext(&auth.Claims{Role: scene.RoleAdmin}))
		require.NotNil(t, data.GetUser(), "expected user to not be nil")
		require.True(t, data.HasRole("Admin"), "expected user to have no role")
		require.True(t, data.IsAdmin(), "expected no user to not be admin")
		require.False(t, data.IsViewOnly(), "expected no user to have no view only")
	})

	t.Run("Observer", func(t *testing.T) {
		data := scene.New(CreateUserContext(&auth.Claims{Role: scene.RoleObserver}))
		require.NotNil(t, data.GetUser(), "expected user to not be nil")
		require.True(t, data.HasRole("Observer"), "expected user to have no role")
		require.False(t, data.IsAdmin(), "expected no user to not be admin")
		require.True(t, data.IsViewOnly(), "expected no user to have no view only")
	})
}

func TestAPIData(t *testing.T) {
	base := scene.New(CreateUserContext(nil))
	require.Nil(t, base.AccountsList(), "expected accounts list to be nil on base")

	out := base.WithAPIData(&api.AccountsList{})
	require.NotNil(t, out.AccountsList(), "expected accounts list to be returned")
	require.Nil(t, out.UserList(), "expected account detail to be nil")

}

func CheckVersion(t *testing.T, data scene.Scene) {
	require.Contains(t, data, scene.Version, "scene did not contain version")
	require.Equal(t, pkg.Version(false), data[scene.Version], "unexpected version mismatch")

	require.Contains(t, data, scene.ShortVersion, "scene did not contain short version")
	require.Equal(t, pkg.Version(true), data[scene.ShortVersion], "unexpected short version mismatch")
}

func CreateContext() *gin.Context {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request, _ = http.NewRequest(http.MethodGet, "http://localhost/dashboard", nil)
	return c
}

func CreateUserContext(claims *auth.Claims) *gin.Context {
	if claims == nil {
		claims = &auth.Claims{
			RegisteredClaims: jwt.RegisteredClaims{
				Subject: "u01HVEH4E88XMYDXFAE4Y48CE9F",
			},
			Name:        "Carlos Hanger",
			Email:       "carlos@example.com",
			Gravatar:    "",
			Role:        "Compliance",
			Permissions: []string{"foo:manage", "foo:view", "foo:delete", "bar:view"},
		}
	}

	c := CreateContext()
	c.Set(auth.ContextUserClaims, claims)
	return c
}

func CreateConf(t *testing.T) (config.Config, error) {
	// Set required environment variables and cleanup after the test is complete.
	t.Cleanup(cleanupEnv())
	setEnv()

	return config.New()
}

var testEnv = map[string]string{
	"TRISA_MAINTENANCE":            "true",
	"TRISA_ORGANIZATION":           "Testing Organization",
	"TRISA_MODE":                   "test",
	"TRISA_LOG_LEVEL":              "debug",
	"TRISA_CONSOLE_LOG":            "false",
	"TRISA_ENDPOINT":               "testing.tr-envoy.com:443",
	"TRISA_WEB_ENABLED":            "false",
	"TRISA_NODE_ENABLED":           "false",
	"TRISA_NODE_POOL":              "fixtures/certs/pool.gz",
	"TRISA_NODE_CERTS":             "fixtures/certs/certs.gz",
	"TRISA_DIRECTORY_SYNC_ENABLED": "false",
	"TRISA_TRP_ENABLED":            "false",
	"TRISA_EMAIL_TESTING":          "true",
	"TRISA_EMAIL_SMTP_HOST":        "email.example.com",
}

// Returns the current environment for the specified keys, or if no keys are specified
// then it returns the current environment for all keys in the testEnv variable.
func curEnv(keys ...string) map[string]string {
	env := make(map[string]string)
	if len(keys) > 0 {
		for _, key := range keys {
			if val, ok := os.LookupEnv(key); ok {
				env[key] = val
			}
		}
	} else {
		for key := range testEnv {
			env[key] = os.Getenv(key)
		}
	}

	return env
}

// Sets the environment variables from the testEnv variable. If no keys are specified,
// then this function sets all environment variables from the testEnv.
func setEnv(keys ...string) {
	if len(keys) > 0 {
		for _, key := range keys {
			if val, ok := testEnv[key]; ok {
				os.Setenv(key, val)
			}
		}
	} else {
		for key, val := range testEnv {
			os.Setenv(key, val)
		}
	}
}

// Cleanup helper function that can be run when the tests are complete to reset the
// environment back to its previous state before the test was run.
func cleanupEnv(keys ...string) func() {
	prevEnv := curEnv(keys...)
	return func() {
		for key, val := range prevEnv {
			if val != "" {
				os.Setenv(key, val)
			} else {
				os.Unsetenv(key)
			}
		}
	}
}
