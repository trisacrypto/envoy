package webhook_test

import (
	"compress/gzip"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/config"
	"github.com/trisacrypto/envoy/pkg/webhook"
	trisa "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
)

func TestWebhook(t *testing.T) {
	// Prepare the request to send to the server
	ctx := context.Background()
	req, err := loadRequest("transaction_payload.json")
	require.NoError(t, err, "could not load request fixture with transaction payload")

	t.Run("PendingReply", func(t *testing.T) {
		srv := httptest.NewServer(makeWebhookHandler("pending_payload.json", ""))
		defer srv.Close()

		endpoint, _ := url.Parse(srv.URL)
		endpoint.Path = "/"

		cb := webhook.New(config.WebhookConfig{URL: endpoint.String()})
		require.IsType(t, &webhook.Webhook{}, cb, "unexpected webhook handler for real url")

		rep, err := cb.Callback(ctx, req)
		require.NoError(t, err, "could not execute callback request")
		require.NotNil(t, rep, "a reply was not returned by the server")
		require.Equal(t, req.TransactionID, rep.TransactionID, "response was not correctly parsed")

		require.Nil(t, rep.Error, "expected a non-nil error returned")
		require.NotNil(t, rep.Payload, "expoected a nil payload returned")
	})

	t.Run("EncodedReply", func(t *testing.T) {
		srv := httptest.NewServer(makeWebhookHandler("pending_payload.json", "gzip"))
		defer srv.Close()

		endpoint, _ := url.Parse(srv.URL)
		endpoint.Path = "/"

		cb := webhook.New(config.WebhookConfig{URL: endpoint.String()})
		require.IsType(t, &webhook.Webhook{}, cb, "unexpected webhook handler for real url")

		rep, err := cb.Callback(ctx, req)
		require.NoError(t, err, "could not execute callback request")
		require.NotNil(t, rep, "a reply was not returned by the server")
		require.Equal(t, req.TransactionID, rep.TransactionID, "response was not correctly parsed")

		require.Nil(t, rep.Error, "expected a non-nil error returned")
		require.NotNil(t, rep.Payload, "expoected a nil payload returned")
	})

	t.Run("ErrorReply", func(t *testing.T) {
		srv := httptest.NewServer(makeWebhookHandler("error.json", ""))
		defer srv.Close()

		endpoint, _ := url.Parse(srv.URL)
		endpoint.Path = "/"

		cb := webhook.New(config.WebhookConfig{URL: endpoint.String()})
		require.IsType(t, &webhook.Webhook{}, cb, "unexpected webhook handler for real url")

		rep, err := cb.Callback(ctx, req)
		require.NoError(t, err, "could not execute callback request")
		require.NotNil(t, rep, "a reply was not returned by the server")
		require.Equal(t, req.TransactionID, rep.TransactionID, "response was not correctly parsed")

		require.NotNil(t, rep.Error, "expected a non-nil error returned")
		require.Nil(t, rep.Payload, "expoected a nil payload returned")
	})

	t.Run("HTTPError", func(t *testing.T) {
		srv := httptest.NewServer(makeWebhookError("the server is currently in maintenance mode", http.StatusServiceUnavailable))
		defer srv.Close()

		endpoint, _ := url.Parse(srv.URL)
		endpoint.Path = "/"

		cb := webhook.New(config.WebhookConfig{URL: endpoint.String()})
		require.IsType(t, &webhook.Webhook{}, cb, "unexpected webhook handler for real url")

		rep, err := cb.Callback(ctx, req)
		require.EqualError(t, err, "could not make webhook callback: received status 503 Service Unavailable")
		require.Nil(t, rep)
	})

	t.Run("NoContentReply", func(t *testing.T) {
		srv := httptest.NewServer(makeWebhookNoContent())
		defer srv.Close()

		endpoint, _ := url.Parse(srv.URL)
		endpoint.Path = "/"

		cb := webhook.New(config.WebhookConfig{URL: endpoint.String()})
		require.IsType(t, &webhook.Webhook{}, cb, "unexpected webhook handler for real url")

		rep, err := cb.Callback(ctx, req)
		require.NoError(t, err, "could not execute callback request")
		require.NotNil(t, rep, "a reply was not returned by the callback")
		require.Equal(t, rep.TransferAction, webhook.DefaultTransferAction, "expected the default transfer action")

		require.Nil(t, rep.Error, "expected a nil error returned")
		require.Nil(t, rep.Payload, "expoected a nil payload returned")
	})

	t.Run("ClientAuth", func(t *testing.T) {
		srv := httptest.NewServer(makeWebhookAuthHandler())
		defer srv.Close()

		endpoint, _ := url.Parse(srv.URL)
		endpoint.Path = "/"

		cb := webhook.New(config.WebhookConfig{
			URL:               endpoint.String(),
			AuthKeyID:         "01JT4B3R5Z6AHJXV87QHPPKRBM",
			AuthKeySecret:     "cfbabc4715b4759d45ba26953dd2fc0bfc2344ef70a2005432e7f16b5081610d",
			RequireServerAuth: false,
		})

		rep, err := cb.Callback(ctx, req)
		require.NoError(t, err, "could not execute callback request")
		require.NotNil(t, rep, "a reply was not returned by the callback")
		require.Equal(t, rep.TransferAction, webhook.DefaultTransferAction, "expected the default transfer action")
	})

	t.Run("BadClientAuth", func(t *testing.T) {
		srv := httptest.NewServer(makeWebhookAuthHandler())
		defer srv.Close()

		endpoint, _ := url.Parse(srv.URL)
		endpoint.Path = "/"

		cb := webhook.New(config.WebhookConfig{
			URL:               endpoint.String(),
			AuthKeyID:         "01JT4JY0MS4BDJAT4BA9T4621Y",
			AuthKeySecret:     "9b33d18fe311b4a0155dde8ca61b94afbce14193232f2c69f5630c6e73818f22",
			RequireServerAuth: false,
		})

		_, err := cb.Callback(ctx, req)
		require.Error(t, err, "could not execute callback request")
		require.EqualError(t, err, "could not make webhook callback: received status 401 Unauthorized")
	})

	t.Run("ServerAuth", func(t *testing.T) {
		srv := httptest.NewServer(makeWebhookAuthHandler())
		defer srv.Close()

		endpoint, _ := url.Parse(srv.URL)
		endpoint.Path = "/"

		cb := webhook.New(config.WebhookConfig{
			URL:               endpoint.String(),
			AuthKeyID:         "01JT4B3R5Z6AHJXV87QHPPKRBM",
			AuthKeySecret:     "cfbabc4715b4759d45ba26953dd2fc0bfc2344ef70a2005432e7f16b5081610d",
			RequireServerAuth: true,
		})

		rep, err := cb.Callback(ctx, req)
		require.NoError(t, err, "could not execute callback request")
		require.NotNil(t, rep, "a reply was not returned by the callback")
		require.Equal(t, rep.TransferAction, webhook.DefaultTransferAction, "expected the default transfer action")
	})
}

//===========================================================================
// Test Server Helpers
//===========================================================================

func makeWebhookHandler(reply, encoding string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var (
			err error
			req *webhook.Request
			rep *webhook.Reply
			out io.Writer
		)

		// Ensure the request is valid and can be decoded.
		req = &webhook.Request{}
		if err = json.NewDecoder(r.Body).Decode(req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Create the reply to send back to the test handler.
		if rep, err = loadReply(reply); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		switch encoding {
		case "gzip":
			w.Header().Set("Content-Encoding", "gzip")
			cw := gzip.NewWriter(w)
			out = cw
			defer cw.Close()
		default:
			out = w
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(out).Encode(rep)
	}
}

func makeWebhookError(msg string, code int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, msg, code)
	}
}

func makeWebhookNoContent() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}
}

func makeWebhookAuthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var (
			err   error
			token *webhook.HMACToken
			key   []byte
		)

		key, _ = hex.DecodeString("cfbabc4715b4759d45ba26953dd2fc0bfc2344ef70a2005432e7f16b5081610d")

		// Require authentication header
		if token, err = webhook.ParseHMAC(r.Header.Get("Authorization")); err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		token.Collect(r.Header)
		if valid, err := token.Verify(key); err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		} else if !valid {
			http.Error(w, "could not verify auth token", http.StatusUnauthorized)
			return
		}

		// Echo a Server-Authorization header back to the client
		mac := webhook.NewHMAC("01JT4B3R5Z6AHJXV87QHPPKRBM", key)
		for _, header := range token.Headers() {
			mac.Append(header, r.Header.Get(header))
			w.Header().Set(header, r.Header.Get(header))
		}

		if auth, err := mac.Authorization(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		} else {
			w.Header().Set("Server-Authorization", auth)
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

//===========================================================================
// Fixture Helpers
//===========================================================================

func loadRequest(name string) (req *webhook.Request, err error) {
	if name != "" && !strings.HasPrefix(name, "request_") {
		name = "request_" + name
	}

	req = &webhook.Request{}
	if err = loadFixture("testdata/request.json", req); err != nil {
		return nil, err
	}

	path := filepath.Join("testdata", name)
	switch {
	case name == "":
		return req, nil
	case strings.HasSuffix(name, "error.json"):
		// Load the error into the request
		req.Error = &trisa.Error{}
		if err = loadFixture(path, req.Error); err != nil {
			return nil, err
		}

		// Modify the request to be error-like
		req.HMAC = ""
		req.PKS = ""
		req.TransferState = "REJECTED"
		req.Payload = nil

	case strings.HasSuffix(name, "payload.json"):
		req.Payload = &webhook.Payload{}
		if err = loadFixture(path, req.Payload); err != nil {
			return nil, err
		}

	default:
		return nil, fmt.Errorf("cannot load request data of type %q", name)
	}

	return req, nil
}

func loadReply(name string) (rep *webhook.Reply, err error) {
	if name != "" && !strings.HasPrefix(name, "reply_") {
		name = "reply_" + name
	}

	rep = &webhook.Reply{}
	if err = loadFixture("testdata/reply.json", rep); err != nil {
		return nil, err
	}

	path := filepath.Join("testdata", name)
	switch {
	case name == "":
		return rep, nil
	case strings.HasSuffix(name, "error.json"):
		// Load the error into the reply
		rep.Error = &trisa.Error{}
		if err = loadFixture(path, rep.Error); err != nil {
			return nil, err
		}

	case strings.HasSuffix(name, "payload.json"):
		rep.Payload = &webhook.Payload{}
		if err = loadFixture(path, rep.Payload); err != nil {
			return nil, err
		}

	default:
		return nil, fmt.Errorf("cannot load reply data of type %q", name)
	}

	return rep, nil
}

func loadFixture(path string, v interface{}) (err error) {
	var f *os.File
	if f, err = os.Open(path); err != nil {
		return err
	}
	defer f.Close()

	if err = json.NewDecoder(f).Decode(v); err != nil {
		return err
	}

	return nil
}
