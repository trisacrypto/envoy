package webhook

import (
	"bytes"
	"compress/gzip"
	"compress/lzw"
	"compress/zlib"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/trisacrypto/envoy/pkg/config"
)

const Timeout = 30 * time.Second

var (
	ErrIdentityRequired    = errors.New("identity payload is required in webhook response")
	ErrTransactionRequired = errors.New("pending or transaction payload is required in webhook response")
)

// New returns a webhook handler that will POST callbacks to the webhook specified by
// the given URL. If the "mock" scheme is specified for the URL, then a MockCallback
// handler will be returned for external testing purposes.
func New(conf config.WebhookConfig) Handler {
	if conf.Endpoint().Scheme == mockScheme {
		return &Mock{}
	}

	return &Webhook{
		url:     conf.URL,
		conf:    conf,
		authKey: conf.DecodeAuthKey(),
		client: &http.Client{
			Timeout: Timeout,
		},
	}
}

type Handler interface {
	Callback(context.Context, *Request) (*Reply, error)
}

// Webhook implements the Handler to make POST requests to the webhook URL.
type Webhook struct {
	client  *http.Client
	url     string
	conf    config.WebhookConfig
	authKey []byte
}

const (
	userAgent      = "Envoy Webhook Client/v1"
	contentType    = "application/json; charset=utf-8"
	accept         = "application/json"
	acceptLang     = "en-US,en"
	acceptEncode   = "gzip;q=1.0, deflate;q=0.8, identity;q=0.5, compress;q=0.1, *;q=0"
	gzipEncode     = "gzip"
	zlibEncode     = "deflate"
	lzwEncode      = "compress"
	identityEncode = "identity"
)

func (h *Webhook) Callback(ctx context.Context, out *Request) (in *Reply, err error) {
	var (
		req  *http.Request
		rep  *http.Response
		data *bytes.Buffer
	)

	data = new(bytes.Buffer)
	if err = json.NewEncoder(data).Encode(out); err != nil {
		return nil, fmt.Errorf("could not marshal request: %s", err)
	}

	if req, err = http.NewRequestWithContext(ctx, http.MethodPost, h.url, data); err != nil {
		return nil, err
	}

	// Add header information to the request
	req.Header.Add("User-Agent", userAgent)
	req.Header.Add("Accept", accept)
	req.Header.Add("Accept-Language", acceptLang)
	req.Header.Add("Accept-Encoding", acceptEncode)
	req.Header.Add("Content-Type", contentType)
	req.Header.Add("X-Transfer-ID", out.TransactionID.String())
	req.Header.Add("X-Transfer-Timestamp", out.Timestamp)

	if h.conf.RequireClientAuth() {
		// Create HMAC authorization token and add it to the request
		mac := NewHMAC(h.conf.AuthKeyID, h.authKey)
		mac.Append("X-Transfer-ID", req.Header.Get("X-Transfer-ID"))
		mac.Append("X-Transfer-Timestamp", req.Header.Get("X-Transfer-Timestamp"))

		var auth string
		if auth, err = mac.Authorization(); err != nil {
			return nil, fmt.Errorf("could not create authorization header: %s", err)
		}

		req.Header.Add("Authorization", auth)
	}

	// Debug logging for the webhook POST request
	log.Debug().
		Str("url", req.URL.String()).
		Str("method", req.Method).
		Str("auth_key_id", h.conf.AuthKeyID).
		Bool("client_auth_required", h.conf.RequireClientAuth()).
		Bool("server_auth_required", h.conf.RequireServerAuth).
		Int64("content_length", req.ContentLength).
		Msg("preparing to send webhook callback")

	// Execute the request
	if rep, err = h.client.Do(req); err != nil {
		return nil, err
	}
	defer rep.Body.Close()

	// Debug logging for the webhook reply
	log.Debug().
		Str("status", rep.Status).
		Int("status_code", rep.StatusCode).
		Int64("content_length", rep.ContentLength).
		Msg("webhook request complete")

	if rep.StatusCode < 200 || rep.StatusCode >= 300 {
		return nil, fmt.Errorf("could not make webhook callback: received status %s", rep.Status)
	}

	// Check server authentication if required
	if h.conf.RequireServerAuth {
		var token *HMACToken
		if token, err = ParseHMAC(rep.Header.Get("Server-Authorization")); err != nil {
			return nil, fmt.Errorf("could not parse server authorization header: %s", err)
		}

		token.Collect(rep.Header)

		if valid, err := token.Verify(h.authKey); err != nil {
			return nil, fmt.Errorf("could not verify server authorization header: %s", err)
		} else if !valid {
			return nil, fmt.Errorf("could not authorize webhook server")
		}
	}

	// Check for non-content 204 response for default handling.
	if rep.StatusCode == http.StatusNoContent {
		return &Reply{TransferAction: DefaultTransferAction}, nil
	}

	// Handle encoding of the response body
	var body io.Reader
	switch rep.Header.Get("Content-Encoding") {
	case gzipEncode:
		if body, err = gzip.NewReader(rep.Body); err != nil {
			return nil, fmt.Errorf("could not create gzip reader: %s", err)
		}
	case zlibEncode:
		if body, err = zlib.NewReader(rep.Body); err != nil {
			return nil, fmt.Errorf("could not create zlib reader: %s", err)
		}
	case "", identityEncode:
		body = rep.Body
	case lzwEncode:
		body = lzw.NewReader(rep.Body, lzw.MSB, 8)
	default:
		return nil, fmt.Errorf("unsupported content encoding %q", rep.Header.Get("Content-Encoding"))
	}

	// Deserialize reply to the webhook POST call
	in = &Reply{}
	if err = json.NewDecoder(body).Decode(in); err != nil {
		return nil, fmt.Errorf("could not unmarshal reply: %s", err)
	}

	// Nilify any zero-valued structs on the reply
	if in.Error != nil && in.Error.IsZero() {
		in.Error = nil
	}

	if in.Payload != nil && in.Payload.IsZero() {
		in.Payload = nil
	}

	return in, nil
}
