package webhook

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
)

var (
	ErrInvalidHMACToken = errors.New("invalid authorization hmac token")
)

// HMAC implements an authorization header to authenticate webhook requests using a
// similar shared secret mechanism as AWS4-HMAC-SHA256 as defined here:
// https://docs.aws.amazon.com/AmazonS3/latest/API/sigv4-auth-using-authorization-header.html
//
// Create an HMAC object with a 32 byte key and a string that represents the key ID.
// Headers then must be appended to the HMAC in order to generate the data to sign.
// Once signed, an HMAC-SHA256 signature is generated and can be used to create the
// Authorization header.
type HMAC struct {
	headers []string
	data    []byte
	key     []byte
	keyID   string
}

func NewHMAC(keyID string, key []byte) *HMAC {
	// Create a nonce to prevent replay attacks as a prefix to the signed data.
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		panic(fmt.Errorf("could not create nonce for HMAC: %w", err))
	}

	return &HMAC{
		headers: make([]string, 0),
		data:    nonce,
		key:     key,
		keyID:   keyID,
	}
}

// Add the header and the header value to the HMAC object. Note that the order that
// the headers are added is important as the values will be appended without
// modification to the data to be signed. If the header is already present, it will
// not be added again. Headers are stored as lowercase values.
func (h *HMAC) Append(header, value string) {
	header = strings.ToLower(header)
	if slices.Contains(h.headers, header) {
		return
	}

	h.headers = append(h.headers, header)
	h.data = append(h.data, []byte(value)...)
}

// Nonce returns the base64 encoded nonce that is used to prevent replay attacks.
func (h *HMAC) Nonce() string {
	return base64.RawURLEncoding.EncodeToString(h.data[:16])
}

// Signature returns the base64 encoded HMAC-SHA256 signature of the data.
func (h *HMAC) Signature() (_ string, err error) {
	mac := hmac.New(sha256.New, h.key)
	if _, err = mac.Write(h.data); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

// Headers returns the semicolon list of request headers used to compute the signature.
// The list includes header names only, and the header names must be in lowercase.
// The order of the headers indicates the order the data was appended to create the
// signature.
func (h *HMAC) Headers() string {
	return strings.Join(h.headers, ";")
}

// Key returns the key ID used to create the HMAC signature.
func (h *HMAC) Key() string {
	return h.keyID
}

// Authorization returns the full Authorization header that can be added to the
// request headers of an outgoing webhook request.
func (h *HMAC) Authorization() (_ string, err error) {
	var sig string
	if sig, err = h.Signature(); err != nil {
		return "", err
	}

	return fmt.Sprintf("HMAC sig=%s, nonce=%s, headers=%s, kid=%s", sig, h.Nonce(), h.Headers(), h.Key()), nil
}

// HMACToken is parsed from an Authorization or Server-Authorization header and is
// used to verify the signature of a webhook request.
type HMACToken struct {
	headers   []string
	data      []byte
	keyID     string
	signature []byte
}

func ParseHMAC(token string) (tok *HMACToken, err error) {
	if !strings.HasPrefix(token, "HMAC ") {
		return nil, ErrInvalidHMACToken
	}

	token = strings.TrimPrefix(token, "HMAC ")
	parts := strings.Split(token, ",")
	kv := make(map[string]string, len(parts))

	for _, part := range parts {
		kvp := strings.SplitN(part, "=", 2)
		kv[strings.TrimSpace(kvp[0])] = strings.TrimSpace(kvp[1])
	}

	tok = &HMACToken{}

	if sig, ok := kv["sig"]; ok {
		if tok.signature, err = base64.RawURLEncoding.DecodeString(sig); err != nil {
			return nil, fmt.Errorf("could not decode signature: %w", err)
		}
	} else {
		return nil, fmt.Errorf("missing signature in HMAC token")
	}

	if nonce, ok := kv["nonce"]; ok {
		if tok.data, err = base64.RawURLEncoding.DecodeString(nonce); err != nil {
			return nil, fmt.Errorf("could not decode nonce: %w", err)
		}
	} else {
		return nil, fmt.Errorf("missing nonce in HMAC token")
	}

	if headers, ok := kv["headers"]; ok {
		tok.headers = strings.Split(headers, ";")
		for i := range tok.headers {
			tok.headers[i] = strings.TrimSpace(tok.headers[i])
			if tok.headers[i] == "" {
				tok.headers = append(tok.headers[:i], tok.headers[i+1:]...)
			}
		}

		if len(tok.headers) == 0 {
			return nil, fmt.Errorf("could not decode headers in HMAC token")
		}
	} else {
		return nil, fmt.Errorf("missing headers in HMAC token")
	}

	if kid, ok := kv["kid"]; ok {
		tok.keyID = kid
		if tok.keyID == "" {
			return nil, fmt.Errorf("could not decode key ID in HMAC token")
		}
	} else {
		return nil, fmt.Errorf("missing key ID in HMAC token")
	}

	return tok, nil
}

func (h *HMACToken) KeyID() string {
	return h.keyID
}

func (h *HMACToken) Collect(headers http.Header) {
	for _, header := range h.headers {
		if value := headers.Get(header); value != "" {
			h.data = append(h.data, []byte(value)...)
		}
	}
}

func (h *HMACToken) Verify(key []byte) (bool, error) {
	mac := hmac.New(sha256.New, key)
	if _, err := mac.Write(h.data); err != nil {
		return false, err
	}

	if !hmac.Equal(mac.Sum(nil), h.signature) {
		return false, nil
	}

	return true, nil
}
