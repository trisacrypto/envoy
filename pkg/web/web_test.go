package web_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/trisacrypto/envoy/pkg/bufconn"
	"github.com/trisacrypto/envoy/pkg/config"
	"github.com/trisacrypto/envoy/pkg/store"
	"github.com/trisacrypto/envoy/pkg/store/mock"
	"github.com/trisacrypto/envoy/pkg/trisa/network"
	"github.com/trisacrypto/envoy/pkg/web"
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
	s       *web.Server
	store   *mock.Store
	httpsrv *httptest.Server
}

// Sets up a web test suite by creating a web.Server from mock components
func (w *webTestSuite) SetupSuite() {
	w.CreateServer()
}

// Shuts down the test servers gracefully
func (w *webTestSuite) TeardownSuite() {
	w.httpsrv.Close()
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

// Creates a web.Server from mock components
func (w *webTestSuite) CreateServer() {
	// Setup the http test server (handler doesn't matter, it will be replaced)
	w.httpsrv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

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

	// Setup the configuration
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
		},
	}

	// Create the web.Server
	w.s, err = web.Debug(conf, w.store, network, w.httpsrv.Config)
	if err != nil {
		panic(err)
	}
}

// Resets all of the suite components
func (w *webTestSuite) ResetAllComponents() {
	// Reset the mock store
	w.store.Reset()

	// Close all connections on the HTTP test server
	w.httpsrv.CloseClientConnections()

	// Close any client connections not in use (should be all of them)
	w.httpsrv.Client().CloseIdleConnections()
}

// Runs the tests that are part of the webTestSuite
func TestWeb(t *testing.T) {
	suite.Run(t, new(webTestSuite))
}

func (w *webTestSuite) TestWebTestSuiteServerStatus() {
	//setup
	require := w.Require()
	type statusResponse struct {
		Status  string
		Version string
		Uptime  string
	}

	//test
	resp, err := w.httpsrv.Client().Get(w.httpsrv.URL + "/v1/status")
	require.NoError(err, "couldn't make an HTTP request to the status endpoint")
	require.NotNil(resp, "expected a non-nil response")
	require.Equalf(200, resp.StatusCode, "non-200 status code: %d", resp.StatusCode)

	body := make([]byte, 76)
	n, err := resp.Body.Read(body)
	if err != nil {
		// This reader can return a nil or an "EOF" error but n should not be zero
		require.ErrorContains(err, "EOF", "expected only EOF errors")
		require.Greater(n, 0, "expected read byte count to be larger than zero")
	}

	stat := &statusResponse{}
	err = json.Unmarshal(body, stat)
	require.NoError(err, "couldn't unmarshal the response status JSON")
	//FIXME: get an "unhealthy" status response here:
	require.Equalf("ok", stat.Status, `expected status 'ok', got '%s'`, stat.Status)
}
