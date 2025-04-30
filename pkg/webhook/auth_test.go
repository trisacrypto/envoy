package webhook_test

import (
	"encoding/hex"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/webhook"
)

func TestHMAC(t *testing.T) {
	key, _ := hex.DecodeString("cfbabc4715b4759d45ba26953dd2fc0bfc2344ef70a2005432e7f16b5081610d")
	keyID := "01JT4B3R5Z6AHJXV87QHPPKRBM"

	headers := make(http.Header)
	headers.Set("Host", "example.com")
	headers.Set("X-Transaction-ID", "f17c9693-c280-4836-b544-245d832a11e0")

	mac := webhook.NewHMAC(keyID, key)
	for header := range headers {
		mac.Append(header, headers.Get(header))
	}

	sig, err := mac.Signature()
	require.NoError(t, err, "could not create signature")

	require.Regexp(t, `^[A-Za-z0-9\-_]{22}$`, mac.Nonce(), "nonce is not 16 base64 encoded bytes")
	require.Regexp(t, `^[A-Za-z0-9\-_]{43}$`, sig, "signature is not 32 base64 encoded bytes")
	require.Equal(t, "host;x-transaction-id", mac.Headers(), "headers do not match")
	require.Equal(t, keyID, mac.Key(), "key ID does not match")

	auth, err := mac.Authorization()
	require.NoError(t, err, "could not create authorization header")
	require.True(t, strings.HasPrefix(auth, "HMAC "))

	token, err := webhook.ParseHMAC(auth)
	require.NoError(t, err, "could not parse authorization header")
	require.Equal(t, mac.Key(), token.KeyID(), "key ID does not match")

	token.Collect(headers)
	ok, err := token.Verify(key)
	require.NoError(t, err, "could not verify token")
	require.True(t, ok, "token verification failed")
}

func TestInvalidHMAC(t *testing.T) {
	key, _ := hex.DecodeString("cfbabc4715b4759d45ba26953dd2fc0bfc2344ef70a2005432e7f16b5081610d")
	keyID := "01JT4B3R5Z6AHJXV87QHPPKRBM"

	headers := make(http.Header)
	headers.Set("Host", "example.com")
	headers.Set("X-Transaction-ID", "f17c9693-c280-4836-b544-245d832a11e0")

	mac := webhook.NewHMAC(keyID, key)
	for header := range headers {
		mac.Append(header, headers.Get(header))
	}

	auth, err := mac.Authorization()
	require.NoError(t, err, "could not create authorization header")

	token, err := webhook.ParseHMAC(auth)
	require.NoError(t, err, "could not parse authorization header")

	token.Collect(headers)

	ok, err := token.Verify([]byte("bad key"))
	require.NoError(t, err, "could not verify token with bad key")
	require.False(t, ok, "token verification should fail with bad key")
}

func TestBadTokens(t *testing.T) {
	// Test invalid tokens
	tokens := []struct {
		token string
		err   string
	}{
		{
			"foo", "invalid authorization hmac token",
		},
		{
			"", "invalid authorization hmac token",
		},
		{
			"HMAC", "invalid authorization hmac token",
		},
		{
			"HMAC sig=f+, nonce=wm0eHegl4lD0Uw_sSYYQCw, headers=host;x-transaction-id, kid=01JT4B3R5Z6AHJXV87QHPPKRBM",
			"could not decode signature: illegal base64 data at input byte 1",
		},
		{
			"HMAC nonce=wm0eHegl4lD0Uw_sSYYQCw, headers=host;x-transaction-id, kid=01JT4B3R5Z6AHJXV87QHPPKRBM",
			"missing signature in HMAC token",
		},
		{
			"HMAC sig=zFeIxdyVnJtpMUExK7HoL37VN4tF6sMQZPEr58MBpMQ, nonce=f+, headers=host;x-transaction-id, kid=01JT4B3R5Z6AHJXV87QHPPKRBM",
			"could not decode nonce: illegal base64 data at input byte 1",
		},
		{
			"HMAC sig=zFeIxdyVnJtpMUExK7HoL37VN4tF6sMQZPEr58MBpMQ, headers=host;x-transaction-id, kid=01JT4B3R5Z6AHJXV87QHPPKRBM",
			"missing nonce in HMAC token",
		},
		{
			"HMAC sig=zFeIxdyVnJtpMUExK7HoL37VN4tF6sMQZPEr58MBpMQ, nonce=wm0eHegl4lD0Uw_sSYYQCw, headers=, kid=01JT4B3R5Z6AHJXV87QHPPKRBM",
			"could not decode headers in HMAC token",
		},
		{
			"HMAC sig=zFeIxdyVnJtpMUExK7HoL37VN4tF6sMQZPEr58MBpMQ, nonce=wm0eHegl4lD0Uw_sSYYQCw, kid=01JT4B3R5Z6AHJXV87QHPPKRBM",
			"missing headers in HMAC token",
		},
		{
			"HMAC sig=zFeIxdyVnJtpMUExK7HoL37VN4tF6sMQZPEr58MBpMQ, nonce=wm0eHegl4lD0Uw_sSYYQCw, headers=host;x-transaction-id, kid=",
			"could not decode key ID in HMAC token",
		},
		{
			"HMAC sig=zFeIxdyVnJtpMUExK7HoL37VN4tF6sMQZPEr58MBpMQ, nonce=wm0eHegl4lD0Uw_sSYYQCw, headers=host;x-transaction-id",
			"missing key ID in HMAC token",
		},
	}

	for i, tc := range tokens {
		tok, err := webhook.ParseHMAC(tc.token)
		require.EqualError(t, err, tc.err, "test case %d: expected error %q, got %q", i, tc.err, err)
		require.Nil(t, tok, "test case %d: expected nil token, got %v", i, tok)
	}
}

func TestExtraKeys(t *testing.T) {
	token := "HMAC foo=bar, sig=zFeIxdyVnJtpMUExK7HoL37VN4tF6sMQZPEr58MBpMQ, nonce=wm0eHegl4lD0Uw_sSYYQCw, headers=host;x-transaction-id, kid=01JT4B3R5Z6AHJXV87QHPPKRBM,color=red"
	_, err := webhook.ParseHMAC(token)
	require.NoError(t, err, "could not parse token with extra keys")
}
