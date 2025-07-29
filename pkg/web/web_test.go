package web_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	dirapi "github.com/trisacrypto/directory/pkg/bff/api/v1"
	"github.com/trisacrypto/envoy/pkg/bufconn"
	"github.com/trisacrypto/envoy/pkg/config"
	"github.com/trisacrypto/envoy/pkg/store"
	"github.com/trisacrypto/envoy/pkg/store/mock"
	"github.com/trisacrypto/envoy/pkg/trisa/network"
	"github.com/trisacrypto/envoy/pkg/web"
	api "github.com/trisacrypto/envoy/pkg/web/api/v1"
	"github.com/trisacrypto/envoy/pkg/web/auth"
	"go.rtnl.ai/ulid"
)

func TestServerEnabled(t *testing.T) {
	t.Run("AtLeastOne", func(t *testing.T) {
		conf := config.Config{Web: config.WebConfig{
			Enabled:    true,
			APIEnabled: false,
			UIEnabled:  false,
		}}

		_, err := web.New(conf, nil, nil)
		require.EqualError(t, err, "invalid configuration: if enabled, either the api, ui, or both need to be enabled")
	})

	store, _ := store.Open("mock:///")
	network, _ := network.NewMocked(nil)

	routes := []struct {
		path   string
		method string
		isAPI  bool
		isUI   bool
	}{
		{"/", http.MethodGet, false, true},
		{"/transactions", http.MethodGet, false, true},
		{"/accounts", http.MethodGet, false, true},
		{"/counterparty", http.MethodGet, false, true},
		{"/send-envelope", http.MethodGet, false, true},
		{"/utilities/travel-address", http.MethodGet, false, true},

		{"/v1/docs", http.MethodGet, true, false},
		{"/v1/status", http.MethodGet, true, false},
		{"/v1/accounts", http.MethodGet, true, true},
		{"/v1/transactions", http.MethodGet, true, true},
		{"/v1/counterparties", http.MethodGet, true, true},
	}

	allValidRoutes := []string{
		"/healthz",
		"/livez",
		"/readyz",
		"/metrics",
	}

	t.Run("APIEnabled", func(t *testing.T) {
		conf := config.Config{Web: config.WebConfig{
			Enabled:    true,
			APIEnabled: true,
			UIEnabled:  false,
			BindAddr:   "127.0.0.1:57132",
			Origin:     "http://locahost:57132",
		}}

		srv, err := web.New(conf, store, network)
		require.NoError(t, err, "could not start web server")

		err = srv.Serve(nil)
		require.NoError(t, err, "could not serve web server")
		defer srv.Shutdown()

		statusForRequest := func(req *http.Request) (int, error) {
			rep, err := http.DefaultClient.Do(req)
			if err != nil {
				return 0, err
			}
			return rep.StatusCode, nil
		}

		statusForAPIRequest := func(endpoint, method string) (int, error) {
			uri := &url.URL{Scheme: "http", Host: "localhost:57132", Path: endpoint}
			req, err := http.NewRequest(method, uri.String(), nil)
			if err != nil {
				return 0, err
			}

			req.Header.Add("Accept", "application/json")
			return statusForRequest(req)
		}

		statusForUIRequest := func(endpoint, method string) (int, error) {
			uri := &url.URL{Scheme: "http", Host: "localhost:57132", Path: endpoint}
			req, err := http.NewRequest(method, uri.String(), nil)
			if err != nil {
				return 0, err
			}

			req.Header.Add("Accept", "text/html")
			return statusForRequest(req)
		}

		for _, route := range routes {
			if route.isAPI {
				// Expect this route to be enabled for a JSON request
				code, err := statusForAPIRequest(route.path, route.method)
				require.NoError(t, err)
				require.NotEqual(t, http.StatusServiceUnavailable, code)
			}

			if route.isUI {
				code, err := statusForUIRequest(route.path, route.method)
				require.NoError(t, err)
				require.Equal(t, http.StatusServiceUnavailable, code)
			}
		}

		// Test Valid for Both UI and API routes
		for _, path := range allValidRoutes {
			code, err := statusForAPIRequest(path, http.MethodGet)
			require.NoError(t, err)
			require.NotEqual(t, http.StatusServiceUnavailable, code)

			code, err = statusForUIRequest(path, http.MethodGet)
			require.NoError(t, err)
			require.NotEqual(t, http.StatusServiceUnavailable, code)
		}

		// Test Not Found route
		code, err := statusForAPIRequest("/v1/foo", http.MethodGet)
		require.NoError(t, err)
		require.Equal(t, http.StatusNotFound, code)

		// Test Not Found route when not enabled
		code, err = statusForUIRequest("/v1/foo", http.MethodGet)
		require.NoError(t, err)
		require.Equal(t, http.StatusServiceUnavailable, code)

		// Test Not Allowed route
		code, err = statusForAPIRequest("/v1/login", http.MethodDelete)
		require.NoError(t, err)
		require.Equal(t, http.StatusMethodNotAllowed, code)

		// Test Not Allowed route when not enabled
		code, err = statusForUIRequest("/v1/login", http.MethodDelete)
		require.NoError(t, err)
		require.Equal(t, http.StatusServiceUnavailable, code)
	})

	t.Run("UIEnabled", func(t *testing.T) {
		conf := config.Config{Web: config.WebConfig{
			Enabled:    true,
			APIEnabled: false,
			UIEnabled:  true,
			BindAddr:   "127.0.0.1:57132",
			Origin:     "http://locahost:57132",
		}}

		srv, err := web.New(conf, store, network)
		require.NoError(t, err, "could not start web server")

		err = srv.Serve(nil)
		require.NoError(t, err, "could not serve web server")
		defer srv.Shutdown()

		statusForRequest := func(req *http.Request) (int, error) {
			rep, err := http.DefaultClient.Do(req)
			if err != nil {
				return 0, err
			}
			return rep.StatusCode, nil
		}

		statusForAPIRequest := func(endpoint, method string) (int, error) {
			uri := &url.URL{Scheme: "http", Host: "localhost:57132", Path: endpoint}
			req, err := http.NewRequest(method, uri.String(), nil)
			if err != nil {
				return 0, err
			}

			req.Header.Add("Accept", "application/json")
			return statusForRequest(req)
		}

		statusForUIRequest := func(endpoint, method string) (int, error) {
			uri := &url.URL{Scheme: "http", Host: "localhost:57132", Path: endpoint}
			req, err := http.NewRequest(method, uri.String(), nil)
			if err != nil {
				return 0, err
			}

			req.Header.Add("Accept", "text/html")
			return statusForRequest(req)
		}

		for _, route := range routes {
			if route.isUI {
				code, err := statusForUIRequest(route.path, route.method)
				require.NoError(t, err)
				require.NotEqual(t, http.StatusServiceUnavailable, code)
			}

			if route.isAPI {
				// Expect this route to be enabled for a JSON request
				code, err := statusForAPIRequest(route.path, route.method)
				require.NoError(t, err)
				require.Equal(t, http.StatusServiceUnavailable, code)

			}
		}

		// Test Valid for Both UI and API routes
		for _, path := range allValidRoutes {
			code, err := statusForAPIRequest(path, http.MethodGet)
			require.NoError(t, err)
			require.NotEqual(t, http.StatusServiceUnavailable, code)

			code, err = statusForUIRequest(path, http.MethodGet)
			require.NoError(t, err)
			require.NotEqual(t, http.StatusServiceUnavailable, code)
		}

		// Test Not Found route
		code, err := statusForUIRequest("/v1/foo", http.MethodGet)
		require.NoError(t, err)
		require.Equal(t, http.StatusNotFound, code)

		// Test Not Found when not enabled
		code, err = statusForAPIRequest("/v1/foo", http.MethodGet)
		require.NoError(t, err)
		require.Equal(t, http.StatusServiceUnavailable, code)

		// Test Not Allowed route
		code, err = statusForUIRequest("/v1/login", http.MethodDelete)
		require.NoError(t, err)
		require.Equal(t, http.StatusMethodNotAllowed, code)

		// Test Not Allowed when not enabled
		code, err = statusForAPIRequest("/v1/login", http.MethodDelete)
		require.NoError(t, err)
		require.Equal(t, http.StatusServiceUnavailable, code)
	})

}

//===========================================================================
// Web Test Suite
//===========================================================================

type webTestSuite struct {
	suite.Suite
	s     *web.Server
	store *mock.Store
	tsrv  *httptest.Server
}

// Sets up a web test suite by creating a web.Server from mock components
func (w *webTestSuite) SetupSuite() {
	w.CreateAPIServer()
}

// Shuts down the test servers gracefully
func (w *webTestSuite) TeardownSuite() {
	w.tsrv.Close()
	w.s.Shutdown()
}

// Resets the state of the web test suite before tests
func (w *webTestSuite) SetupTest() {
	w.ResetAllComponents()
}

// Resets the state of the web test suite before sub-tests
func (w *webTestSuite) SetupSubTest() {
	w.ResetAllComponents()
}

// Creates a web.Server (with no UI, just API) from mock components
func (w *webTestSuite) CreateAPIServer() {
	// Setup the http test server (handler doesn't matter, it will be replaced)
	w.tsrv = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	// Setup the mock store
	sto, err := store.Open("mock:///")
	if err != nil {
		panic(err)
	}
	w.store = sto.(*mock.Store)

	// Setup the mock trisa network
	network, err := network.NewMocked(&config.TRISAConfig{
		Maintenance: false,
		Enabled:     true,
		BindAddr:    "bufnet",
		MTLSConfig: config.MTLSConfig{
			Certs: "../trisa/testdata/certs/alice.vaspbot.com.pem",
			Pool:  "../trisa/testdata/certs/trisatest.dev.pem",
		},
		KeyExchangeCacheTTL: 60 * time.Second,
		Directory: config.DirectoryConfig{
			Insecure:        true,
			Endpoint:        bufconn.Endpoint,
			MembersEndpoint: bufconn.Endpoint,
		},
	})
	if err != nil {
		panic(err)
	}

	// Setup the configuration with only the necessary parts
	// NOTE: we may need to add to this if a new test requires it later!
	conf := config.Config{
		Maintenance:  false,
		Organization: "Envoy Testing",
		Mode:         "testing",
		ConsoleLog:   true,
		Web: config.WebConfig{
			Enabled:    true,
			APIEnabled: true,
			UIEnabled:  false,
			BindAddr:   ":4000",
			Origin:     "http://localhost:4000",
			Auth: config.AuthConfig{
				AccessTokenTTL: 1 * time.Hour,
				Audience:       "http://localhost:4000",
				Issuer:         "http://localhost:4000",
			},
		},
	}

	// Create the web.Server
	w.s, err = web.Debug(conf, w.store, network, w.tsrv.Config)
	if err != nil {
		panic(err)
	}
}

// Resets all of the test suite components
func (w *webTestSuite) ResetAllComponents() {
	// Reset the mock store
	w.store.Reset()

	// Close all connections on the HTTP test server
	w.tsrv.CloseClientConnections()

	// Close any client connections not in use (should be all of them)
	w.tsrv.Client().CloseIdleConnections()
}

// Contains the strings for every permission available
var AllPermissions = []string{
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
}

// Returns an authenticated api.Client configured to communicate with the test
// server with the provided list of permissions. Use the variable AllPermissions
// if you want all of the permissions available.
func (w *webTestSuite) ClientWithPermissions(permissions []string) api.Client {
	// Generate an access token with the permissions given
	access, _, err := w.s.Issuer().CreateTokens(&auth.Claims{
		ClientID:    "webTestSuite",
		Permissions: permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			// API key subject; will be the actor metadata for audit logging
			Subject: "k" + ulid.MakeSecure().String(),
		},
	})
	if err != nil {
		panic(err)
	}

	// Create the client
	client, err := api.New(w.tsrv.URL, api.WithClient(w.tsrv.Client()), api.WithCreds(dirapi.Token(access)))
	if err != nil {
		panic(err)
	}
	return client
}

// Returns an unauthenticated api.Client configured to communicate with the test
// server
func (w *webTestSuite) ClientNoAuth() api.Client {
	client, err := api.New(w.tsrv.URL, api.WithClient(w.tsrv.Client()))
	if err != nil {
		panic(err)
	}
	return client
}

// Runs the tests that are part of the webTestSuite
func TestWeb(t *testing.T) {
	suite.Run(t, new(webTestSuite))
}
