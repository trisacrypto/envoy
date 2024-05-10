package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"

	"github.com/trisacrypto/envoy/pkg/web/api/v1/credentials"

	"github.com/cenkalti/backoff/v4"
	"github.com/google/go-querystring/query"
	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
	"github.com/rs/zerolog/log"
)

// New creates a new APIv1 client that implements the Client interface.
func New(endpoint string, opts ...ClientOption) (_ Client, err error) {
	c := &APIv1{}
	if c.endpoint, err = url.Parse(endpoint); err != nil {
		return nil, fmt.Errorf("could not parse endpoint: %s", err)
	}

	// Apply our options
	for _, opt := range opts {
		if err = opt(c); err != nil {
			return nil, err
		}
	}

	// If an http client isn't specified, create a default client.
	if c.client == nil {
		c.client = &http.Client{
			Transport:     nil,
			CheckRedirect: nil,
			Timeout:       30 * time.Second,
		}

		// Create cookie jar for CSRF
		if c.client.Jar, err = cookiejar.New(nil); err != nil {
			return nil, fmt.Errorf("could not create cookiejar: %w", err)
		}
	}

	return c, nil
}

// APIv1 implements the v1 Client interface for making requests to the TRISA SHN.
type APIv1 struct {
	endpoint *url.URL                // the base url for all requests
	client   *http.Client            // used to make http requests to the server
	creds    credentials.Credentials // default credentials used to authorize requests
}

// Ensure the APIv1 implements the Client interface
var _ Client = &APIv1{}

//===========================================================================
// Client Methods
//===========================================================================

const statusEP = "/v1/status"

func (s *APIv1) Status(ctx context.Context) (out *StatusReply, err error) {
	// Make the HTTP request
	var req *http.Request
	if req, err = s.NewRequest(ctx, http.MethodGet, statusEP, nil, nil); err != nil {
		return nil, err
	}

	// NOTE: we cannot use s.Do because we want to parse 503 Unavailable errors
	var rep *http.Response
	if rep, err = s.client.Do(req); err != nil {
		return nil, err
	}
	defer rep.Body.Close()

	// Detect other errors
	if rep.StatusCode != http.StatusOK && rep.StatusCode != http.StatusServiceUnavailable {
		return nil, fmt.Errorf("%s", rep.Status)
	}

	// Deserialize the JSON data from the response
	out = &StatusReply{}
	if err = json.NewDecoder(rep.Body).Decode(out); err != nil {
		return nil, fmt.Errorf("could not deserialize status reply: %s", err)
	}
	return out, nil
}

const loginEP = "/v1/login"

func (s *APIv1) Login(ctx context.Context, in *LoginRequest) (out *LoginReply, err error) {
	return s.authenticate(ctx, loginEP, in)
}

const authenticateEP = "/v1/authenticate"

func (s *APIv1) Authenticate(ctx context.Context, in *APIAuthentication) (out *LoginReply, err error) {
	return s.authenticate(ctx, authenticateEP, in)
}

const refreshEP = "/v1/reauthenticate"

func (s *APIv1) Reauthenticate(ctx context.Context, in *ReauthenticateRequest) (out *LoginReply, err error) {
	return s.authenticate(ctx, refreshEP, in)
}

func (s *APIv1) authenticate(ctx context.Context, endpoint string, in any) (out *LoginReply, err error) {
	// Authenticate requests are posts with the given data (e.g. user credentials, api key credentials, or refresh token)
	var req *http.Request
	if req, err = s.NewRequest(ctx, http.MethodPost, endpoint, in, nil); err != nil {
		return nil, err
	}

	// The response will always be a login reply
	if _, err = s.Do(req, &out, true); err != nil {
		return nil, err
	}

	// Set the returned credentials on the client for future requests
	// TODO: handle refresh tokens for reauthentication
	s.creds = credentials.Token(out.AccessToken)
	return out, err
}

//===========================================================================
// Transactions Resource
//===========================================================================

const transactionsEP = "/v1/transactions"

func (s *APIv1) ListTransactions(ctx context.Context, in *PageQuery) (out *TransactionsList, err error) {
	if err = s.List(ctx, transactionsEP, in, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *APIv1) CreateTransaction(ctx context.Context, in *Transaction) (out *Transaction, err error) {
	if err = s.Create(ctx, transactionsEP, in, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *APIv1) TransactionDetail(ctx context.Context, id uuid.UUID) (out *Transaction, err error) {
	endpoint, _ := url.JoinPath(transactionsEP, id.String())
	if err = s.Detail(ctx, endpoint, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *APIv1) UpdateTransaction(ctx context.Context, in *Transaction) (out *Transaction, err error) {
	endpoint, _ := url.JoinPath(transactionsEP, in.ID.String())
	if err = s.Update(ctx, endpoint, in, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *APIv1) DeleteTransaction(ctx context.Context, id uuid.UUID) (err error) {
	endpoint, _ := url.JoinPath(transactionsEP, id.String())
	return s.Delete(ctx, endpoint)
}

//===========================================================================
// Secure and Decrypted Envelopes Resource
//===========================================================================

const secureEnvelopesEP = "secure-envelopes"

func (s *APIv1) ListSecureEnvelopes(ctx context.Context, transactionID uuid.UUID, in *EnvelopeListQuery) (out *EnvelopesList, err error) {
	var params url.Values
	if params, err = query.Values(in); err != nil {
		return nil, fmt.Errorf("could not encode envelope page query: %w", err)
	}

	endpoint, _ := url.JoinPath(transactionsEP, transactionID.String(), secureEnvelopesEP)

	var req *http.Request
	if req, err = s.NewRequest(ctx, http.MethodGet, endpoint, nil, &params); err != nil {
		return nil, err
	}

	if _, err = s.Do(req, &out, true); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *APIv1) SecureEnvelopeDetail(ctx context.Context, transactionID uuid.UUID, envID ulid.ULID) (out *SecureEnvelope, err error) {
	var params url.Values
	if params, err = query.Values(&EnvelopeQuery{Decrypt: false}); err != nil {
		return nil, fmt.Errorf("could not encode envelope query: %w", err)
	}

	endpoint, _ := url.JoinPath(transactionsEP, transactionID.String(), secureEnvelopesEP, envID.String())

	var req *http.Request
	if req, err = s.NewRequest(ctx, http.MethodGet, endpoint, nil, &params); err != nil {
		return nil, err
	}

	if _, err = s.Do(req, &out, true); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *APIv1) DecryptedEnvelopeDetail(ctx context.Context, transactionID uuid.UUID, envID ulid.ULID) (out *DecryptedEnvelope, err error) {
	var params url.Values
	if params, err = query.Values(&EnvelopeQuery{Decrypt: true}); err != nil {
		return nil, fmt.Errorf("could not encode envelope query: %w", err)
	}

	endpoint, _ := url.JoinPath(transactionsEP, transactionID.String(), secureEnvelopesEP, envID.String())

	var req *http.Request
	if req, err = s.NewRequest(ctx, http.MethodGet, endpoint, nil, &params); err != nil {
		return nil, err
	}

	if _, err = s.Do(req, &out, true); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *APIv1) DeleteSecureEnvelope(ctx context.Context, transactionID uuid.UUID, envID ulid.ULID) error {
	endpoint, _ := url.JoinPath(transactionsEP, transactionID.String(), secureEnvelopesEP, envID.String())
	return s.Delete(ctx, endpoint)
}

//===========================================================================
// Accounts Resource
//===========================================================================

const accountsEP = "/v1/accounts"

func (s *APIv1) ListAccounts(ctx context.Context, in *PageQuery) (out *AccountsList, err error) {
	if err = s.List(ctx, accountsEP, in, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *APIv1) CreateAccount(ctx context.Context, in *Account) (out *Account, err error) {
	if err = s.Create(ctx, accountsEP, in, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *APIv1) AccountDetail(ctx context.Context, id ulid.ULID) (out *Account, err error) {
	endpoint, _ := url.JoinPath(accountsEP, id.String())
	if err = s.Detail(ctx, endpoint, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *APIv1) UpdateAccount(ctx context.Context, in *Account) (out *Account, err error) {
	endpoint, _ := url.JoinPath(accountsEP, in.ID.String())
	if err = s.Update(ctx, endpoint, in, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *APIv1) DeleteAccount(ctx context.Context, id ulid.ULID) (err error) {
	endpoint, _ := url.JoinPath(accountsEP, id.String())
	return s.Delete(ctx, endpoint)
}

//===========================================================================
// CryptoAddress Resource
//===========================================================================

const cryptoAddressesEP = "crypto-addresses"

func (s *APIv1) ListCryptoAddresses(ctx context.Context, accountID ulid.ULID, in *PageQuery) (out *CryptoAddressList, err error) {
	endpoint, _ := url.JoinPath(accountsEP, accountID.String(), cryptoAddressesEP)
	if err = s.List(ctx, endpoint, in, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *APIv1) CreateCryptoAddress(ctx context.Context, accountID ulid.ULID, in *CryptoAddress) (out *CryptoAddress, err error) {
	endpoint, _ := url.JoinPath(accountsEP, accountID.String(), cryptoAddressesEP)
	if err = s.Create(ctx, endpoint, in, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *APIv1) CryptoAddressDetail(ctx context.Context, accountID, cryptoAddressID ulid.ULID) (out *CryptoAddress, err error) {
	endpoint, _ := url.JoinPath(accountsEP, accountID.String(), cryptoAddressesEP, cryptoAddressID.String())
	if err = s.Detail(ctx, endpoint, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *APIv1) UpdateCryptoAddress(ctx context.Context, accountID ulid.ULID, in *CryptoAddress) (out *CryptoAddress, err error) {
	endpoint, _ := url.JoinPath(accountsEP, accountID.String(), cryptoAddressesEP, in.ID.String())
	if err = s.Update(ctx, endpoint, in, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *APIv1) DeleteCryptoAddress(ctx context.Context, accountID, cryptoAddressID ulid.ULID) (err error) {
	endpoint, _ := url.JoinPath(accountsEP, accountID.String(), cryptoAddressesEP, cryptoAddressID.String())
	return s.Delete(ctx, endpoint)
}

//===========================================================================
// Counterparty Resource
//===========================================================================

const counterpartiesEP = "/v1/counterparties"

func (s *APIv1) ListCounterparties(ctx context.Context, in *PageQuery) (out *CounterpartyList, err error) {
	if err = s.List(ctx, counterpartiesEP, in, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *APIv1) CreateCounterparty(ctx context.Context, in *Counterparty) (out *Counterparty, err error) {
	if err = s.Create(ctx, counterpartiesEP, in, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *APIv1) CounterpartyDetail(ctx context.Context, id ulid.ULID) (out *Counterparty, err error) {
	endpoint, _ := url.JoinPath(counterpartiesEP, id.String())
	if err = s.Detail(ctx, endpoint, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *APIv1) UpdateCounterparty(ctx context.Context, in *Counterparty) (out *Counterparty, err error) {
	endpoint, _ := url.JoinPath(counterpartiesEP, in.ID.String())
	if err = s.Update(ctx, endpoint, in, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *APIv1) DeleteCounterparty(ctx context.Context, id ulid.ULID) (err error) {
	endpoint, _ := url.JoinPath(counterpartiesEP, id.String())
	return s.Delete(ctx, endpoint)
}

//===========================================================================
// Utilities Resource
//===========================================================================

const utilitiesEndpoint = "/v1/utilities"

func (s *APIv1) EncodeTravelAddress(ctx context.Context, in *TravelAddress) (out *TravelAddress, err error) {
	endpoint, _ := url.JoinPath(utilitiesEndpoint, "encode")
	if err = s.Create(ctx, endpoint, in, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *APIv1) DecodeTravelAddress(ctx context.Context, in *TravelAddress) (out *TravelAddress, err error) {
	endpoint, _ := url.JoinPath(utilitiesEndpoint, "decode")
	if err = s.Create(ctx, endpoint, in, &out); err != nil {
		return nil, err
	}
	return out, nil
}

//===========================================================================
// Client Utility Methods
//===========================================================================

// Wait for ready polls the node's status endpoint until it responds with an 200
// response, retrying with exponential backoff or until the context deadline is expired.
// If the user does not supply a context with a deadline, then a default deadline of
// 5 minutes is used so that this method does not block indefinitely. If the node API
// service is ready (e.g. responds to a status request) then no error is returned,
// otherwise an error is returned if the node never responds.
//
// NOTE: if the node returns a 503 Service Unavailable because it is in maintenance
// mode, this method will continue to wait until the deadline for the node to exit
// from maintenance mode and be ready again.
func (s *APIv1) WaitForReady(ctx context.Context) (err error) {
	// If context does not have a deadline, create a context with a default deadline.
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()
	}

	// Create the status request to send until ready
	var req *http.Request
	if req, err = s.NewRequest(ctx, http.MethodGet, "/v1/status", nil, nil); err != nil {
		return err
	}

	// Create a closure to repeatedly call the status endpoint
	checkReady := func() (err error) {
		var rep *http.Response
		if rep, err = s.client.Do(req); err != nil {
			return err
		}
		defer rep.Body.Close()

		if rep.StatusCode < 200 || rep.StatusCode >= 300 {
			return &StatusError{StatusCode: rep.StatusCode, Reply: Reply{Success: false, Error: http.StatusText(rep.StatusCode)}}
		}
		return nil
	}

	// Create exponential backoff ticker for retries
	ticker := backoff.NewExponentialBackOff()

	// Keep checking if the node is ready until it is ready or until the context expires.
	for {
		// Execute the status request
		if err = checkReady(); err == nil {
			// Success - node is ready for requests!
			return nil
		}

		// Log the error warning that we're still waiting to connect to the node
		log.Warn().Err(err).Str("endpoint", s.endpoint.String()).Msg("waiting to connect to TRISA node")
		wait := time.After(ticker.NextBackOff())

		// Wait for the context to be done or for the ticker to move to the next backoff.
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-wait:
		}
	}
}

//===========================================================================
// REST Resource Methods
//===========================================================================

func (s *APIv1) List(ctx context.Context, endpoint string, in *PageQuery, out interface{}) (err error) {
	var params url.Values
	if params, err = query.Values(in); err != nil {
		return fmt.Errorf("could not encode page query: %w", err)
	}

	var req *http.Request
	if req, err = s.NewRequest(ctx, http.MethodGet, endpoint, nil, &params); err != nil {
		return err
	}

	if _, err = s.Do(req, &out, true); err != nil {
		return err
	}

	return nil
}

func (s *APIv1) Create(ctx context.Context, endpoint string, in, out interface{}) (err error) {
	var req *http.Request
	if req, err = s.NewRequest(ctx, http.MethodPost, endpoint, in, nil); err != nil {
		return err
	}

	if _, err = s.Do(req, &out, true); err != nil {
		return err
	}
	return nil
}

func (s *APIv1) Detail(ctx context.Context, endpoint string, out interface{}) (err error) {
	var req *http.Request
	if req, err = s.NewRequest(ctx, http.MethodGet, endpoint, nil, nil); err != nil {
		return err
	}

	if _, err = s.Do(req, &out, true); err != nil {
		return err
	}
	return nil
}

func (s *APIv1) Update(ctx context.Context, endpoint string, in, out interface{}) (err error) {
	var req *http.Request
	if req, err = s.NewRequest(ctx, http.MethodPut, endpoint, in, nil); err != nil {
		return err
	}

	if _, err = s.Do(req, &out, true); err != nil {
		return err
	}
	return nil
}

func (s *APIv1) Delete(ctx context.Context, endpoint string) (err error) {
	var req *http.Request
	if req, err = s.NewRequest(ctx, http.MethodDelete, endpoint, nil, nil); err != nil {
		return err
	}

	if _, err = s.Do(req, nil, true); err != nil {
		return err
	}
	return nil
}

//===========================================================================
// Helper Methods
//===========================================================================

const (
	userAgent    = "TRISA Node API Client/v1"
	accept       = "application/json"
	acceptLang   = "en-US,en"
	acceptEncode = "gzip, deflate, br"
	contentType  = "application/json; charset=utf-8"
)

func (s *APIv1) NewRequest(ctx context.Context, method, path string, data interface{}, params *url.Values) (req *http.Request, err error) {
	// Resolve the URL reference from the path
	url := s.endpoint.ResolveReference(&url.URL{Path: path})
	if params != nil && len(*params) > 0 {
		url.RawQuery = params.Encode()
	}

	var body io.ReadWriter
	switch {
	case data == nil:
		body = nil
	default:
		body = &bytes.Buffer{}
		if err = json.NewEncoder(body).Encode(data); err != nil {
			return nil, fmt.Errorf("could not serialize request data as json: %s", err)
		}
	}

	// Create the http request
	if req, err = http.NewRequestWithContext(ctx, method, url.String(), body); err != nil {
		return nil, fmt.Errorf("could not create request: %s", err)
	}

	// Set the headers on the request
	req.Header.Add("User-Agent", userAgent)
	req.Header.Add("Accept", accept)
	req.Header.Add("Accept-Language", acceptLang)
	req.Header.Add("Accept-Encoding", acceptEncode)
	req.Header.Add("Content-Type", contentType)

	// If there is a request ID on the context, set it on the request, otherwise generate one
	var requestID string
	if requestID, _ = RequestIDFromContext(ctx); requestID == "" {
		requestID = ulid.Make().String()
	}
	req.Header.Add("X-Request-ID", requestID)

	// Add authentication and authorization header.
	if s.creds != nil {
		var token string
		if token, err = s.creds.AccessToken(); err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+token)
	}

	// Add CSRF protection if its available
	if s.client.Jar != nil {
		cookies := s.client.Jar.Cookies(url)
		for _, cookie := range cookies {
			if cookie.Name == "csrf_token" {
				req.Header.Add("X-CSRF-TOKEN", cookie.Value)
			}
		}
	}

	return req, nil
}

// Do executes an http request against the server, performs error checking, and
// deserializes the response data into the specified struct.
func (s *APIv1) Do(req *http.Request, data interface{}, checkStatus bool) (rep *http.Response, err error) {
	if rep, err = s.client.Do(req); err != nil {
		return rep, fmt.Errorf("could not execute request: %s", err)
	}
	defer rep.Body.Close()

	// Detect http status errors if they've occurred
	if checkStatus {
		if rep.StatusCode < 200 || rep.StatusCode >= 300 {
			// Attempt to read the error response from JSON, if available
			serr := &StatusError{
				StatusCode: rep.StatusCode,
			}

			if err = json.NewDecoder(rep.Body).Decode(&serr.Reply); err == nil {
				return rep, serr
			}

			serr.Reply = Unsuccessful
			return rep, serr
		}
	}

	// Deserialize the JSON data from the body
	if data != nil && rep.StatusCode >= 200 && rep.StatusCode < 300 && rep.StatusCode != http.StatusNoContent {
		ct := rep.Header.Get("Content-Type")
		if ct != "" {
			mt, _, err := mime.ParseMediaType(ct)
			if err != nil {
				return nil, fmt.Errorf("malformed content-type header: %w", err)
			}

			if mt != accept {
				return nil, fmt.Errorf("unexpected content type: %q", mt)
			}
		}

		if err = json.NewDecoder(rep.Body).Decode(data); err != nil {
			return nil, fmt.Errorf("could not deserialize response data: %s", err)
		}
	}

	return rep, nil
}
