package web_test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/config"
	"github.com/trisacrypto/envoy/pkg/store"
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
			BindAddr:   "127.0.0.1:9100",
			Origin:     "http://locahost:9100",
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
			uri := &url.URL{Scheme: "http", Host: "localhost:9100", Path: endpoint}
			req, err := http.NewRequest(method, uri.String(), nil)
			if err != nil {
				return 0, err
			}

			req.Header.Add("Accept", "application/json")
			return statusForRequest(req)
		}

		statusForUIRequest := func(endpoint, method string) (int, error) {
			uri := &url.URL{Scheme: "http", Host: "localhost:9100", Path: endpoint}
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
			BindAddr:   "127.0.0.1:9100",
			Origin:     "http://locahost:9100",
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
			uri := &url.URL{Scheme: "http", Host: "localhost:9100", Path: endpoint}
			req, err := http.NewRequest(method, uri.String(), nil)
			if err != nil {
				return 0, err
			}

			req.Header.Add("Accept", "application/json")
			return statusForRequest(req)
		}

		statusForUIRequest := func(endpoint, method string) (int, error) {
			uri := &url.URL{Scheme: "http", Host: "localhost:9100", Path: endpoint}
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
