package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/rs/zerolog/log"
)

const Timeout = 30 * time.Second

var (
	ErrIdentityRequired    = errors.New("identity payload is required in webhook response")
	ErrTransactionRequired = errors.New("pending or transaction payload is required in webhook response")
)

// New returns a webhook handler that will POST callbacks to the webhook specified by
// the given URL. If the "mock" scheme is specified for the URL, then a MockCallback
// handler will be returned for external testing purposes.
func New(webhook *url.URL) Handler {
	if webhook.Scheme == mockScheme {
		return &Mock{}
	}

	return &Webhook{
		url: webhook.String(),
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
	client *http.Client
	url    string
}

const (
	userAgent    = "Envoy Webhook Client/v1"
	contentType  = "application/json; charset=utf-8"
	accept       = "application/json"
	acceptLang   = "en-US,en"
	acceptEncode = "gzip;q=1.0, deflate;q=0.5, br;q=0.5, *;q=0.1"
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

	// Debug logging for the webhook POST request
	log.Debug().
		Str("url", req.URL.String()).
		Str("method", req.Method).
		Int64("content_length", req.ContentLength).
		Msg("preparing to send webhook callback")

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

	// Deserialize reply to the webhook POST call
	in = &Reply{}
	if err = json.NewDecoder(rep.Body).Decode(in); err != nil {
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
