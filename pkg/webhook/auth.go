package webhook

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"slices"
	"strings"
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
func (h *HMAC) AddHeader(header, value string) {
	header = strings.ToLower(header)
	if slices.Contains(h.headers, header) {
		return
	}

	h.headers = append(h.headers, header)
	h.data = append(h.data, []byte(value)...)
}

// Nonce returns the base64 encoded nonce that is used to prevent replay attacks.
func (h *HMAC) Nonce() string {
	return base64.URLEncoding.EncodeToString(h.data[:16])
}

// Signature returns the base64 encoded HMAC-SHA256 signature of the data.
func (h *HMAC) Signature() (_ string, err error) {
	mac := hmac.New(sha256.New, h.key)
	if _, err = mac.Write(h.data); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(mac.Sum(nil)), nil
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
