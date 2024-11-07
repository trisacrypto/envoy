package sunrise_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/sunrise"
)

func TestVerification(t *testing.T) {
	// Generate verification token and signature
	verify, signature, err := sunrise.NewToken(uuid.New(), time.Now().Add(1*time.Hour)).Sign()
	require.NoError(t, err, "could not sign token")

	// Create tokens string to send to user; assume that the signature is saved to db and loaded again
	tks := verify.String()

	// Pretend to save the signature to the database by marshaling it
	dbd, err := signature.MarshalBinary()
	require.NoError(t, err, "could not marshal signature")

	// Parse the incoming token from the user
	token, err := sunrise.ParseVerification(tks)
	require.NoError(t, err, "could not parse verification token")

	// Pretend to load the signature from the database by unmarshaling it
	signature = &sunrise.SignedToken{}
	err = signature.UnmarshalBinary(dbd)
	require.NoError(t, err, "could not unmarshal signature")

	// Complete the workflow by verifying that everything is correct and secure
	secure, err := signature.Verify(token)
	require.NoError(t, err, "could not verify token")
	require.True(t, secure, "verification returned false")
}

func TestNewToken(t *testing.T) {
	t.Run("DefaultExpiration", func(t *testing.T) {
		token := sunrise.NewToken(uuid.New(), time.Time{})
		require.False(t, token.Expiration.IsZero(), "expected an expiration timestamp to be set")
		require.True(t, token.Expiration.After(time.Now()), "expiration is not set in the future")
	})

	t.Run("NonceGeneration", func(t *testing.T) {
		token := sunrise.NewToken(uuid.New(), time.Now())
		data, err := token.MarshalBinary()
		require.NoError(t, err, "could not marshal binary")

		// Expects nonce length to be 64!
		require.NotEqual(t, bytes.Repeat([]byte{0x0}, 64), data[len(data)-64:], "zero-valued nonce!")
	})

	t.Run("Randomness", func(t *testing.T) {
		envelopeID := uuid.New()
		expiration := time.Now().Add(1 * time.Hour)

		// Generate 16 tokens with the same envelopeID and expiration timestamp
		tokens := make([]*sunrise.Token, 0, 16)
		for i := 0; i < 16; i++ {
			tokens = append(tokens, sunrise.NewToken(envelopeID, expiration))
		}

		// Ensure that all marshaled tokens are different (because of the nonce)
		for i, alpha := range tokens {
			for j, bravo := range tokens {
				if i == j {
					// Don't compare the same token to itself
					continue
				}

				da, err := alpha.MarshalBinary()
				require.NoError(t, err, "could not marshal token %d", i)

				db, err := bravo.MarshalBinary()
				require.NoError(t, err, "could not marshal token %d", j)

				require.False(t, bytes.Equal(da, db), "tokens %d and %d were identical!", i, j)
			}
		}

	})
}

func TestTokenExpiration(t *testing.T) {
	testCases := []struct {
		token  *sunrise.Token
		assert require.BoolAssertionFunc
	}{
		{
			sunrise.NewToken(uuid.New(), time.Now().Add(7*24*time.Hour)),
			require.False,
		},
		{
			sunrise.NewToken(uuid.New(), time.Now().Add(-7*24*time.Hour)),
			require.True,
		},
		{
			&sunrise.Token{EnvelopeID: uuid.New()},
			require.True,
		},
	}

	for i, tc := range testCases {
		tc.assert(t, tc.token.IsExpired(), "test case %d failed", i)
	}
}

func TestTokenSign(t *testing.T) {
	t.Run("Happy", func(t *testing.T) {
		token := sunrise.NewToken(uuid.New(), time.Now())
		verification, signature, err := token.Sign()
		require.NoError(t, err, "could not sign token")
		require.Len(t, verification, 16+64, "unexpected length of verification token (16 byte uuid + 64 byte secret)")
		require.Len(t, signature.Signature(), 32, "unexpected length of hmac signature (32 bytes for sha256)")
	})

	t.Run("WithoutNonce", func(t *testing.T) {
		token := &sunrise.Token{EnvelopeID: uuid.New(), Expiration: time.Now()}
		verification, signature, err := token.Sign()
		require.NoError(t, err, "could not sign token")
		require.Len(t, verification, 16+64, "unexpected length of verification token (16 byte uuid + 64 byte secret)")
		require.Len(t, signature.Signature(), 32, "unexpected length of hmac signature (32 bytes for sha256)")
	})

	t.Run("VerificationToken", func(t *testing.T) {
		envelopeID := uuid.New()
		verify, _, err := sunrise.NewToken(envelopeID, time.Time{}).Sign()
		require.NoError(t, err, "could not sign token")

		require.Equal(t, envelopeID, verify.EnvelopeID(), "expected envelope ID to match")
		require.Len(t, verify.Secret(), 64, "expected secret to be 64 bytes long")
	})

	t.Run("Sad", func(t *testing.T) {
		testCases := []*sunrise.Token{
			{},
			{EnvelopeID: uuid.New()},
			{EnvelopeID: uuid.Nil, Expiration: time.Now()},
		}

		for i, token := range testCases {
			verify, signature, err := token.Sign()
			require.Error(t, err, "expected an error on test case %d", i)
			require.Nil(t, verify, "expected nil verification on test case %d", i)
			require.Nil(t, signature, "expected nil signed token on test case %d", i)
		}
	})

	t.Run("SecretRandomness", func(t *testing.T) {
		// Create a token with constant nonce, uuid, and expiration
		token := sunrise.NewToken(uuid.New(), time.Now().Add(1*time.Hour))

		// Create 16 verification tokens from the same token

		tokens := make([]sunrise.VerificationToken, 0, 16)
		signatures := make([]*sunrise.SignedToken, 0, 16)
		for i := 0; i < 16; i++ {
			verify, signed, err := token.Sign()
			require.NoError(t, err, "could not sign token")

			tokens = append(tokens, verify)
			signatures = append(signatures, signed)
		}

		for i, alpha := range tokens {
			for j, bravo := range tokens {
				if i == j {
					// Don't compare the same token
					continue
				}

				// No two verification tokens and signatures should be the same
				require.False(t, bytes.Equal(alpha, bravo), "verification token %d is equal to token %d", i, j)
				require.False(t, bytes.Equal(signatures[i].Signature(), signatures[j].Signature()), "signature %d is equal to signature %d", i, j)
			}
		}

	})

}

func TestTokenBinary(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		testCases := []*sunrise.Token{
			sunrise.NewToken(uuid.MustParse("24035c84-ff3d-4da2-aef7-8683d9c00978"), time.Date(1994, 12, 20, 15, 21, 1, 3213, time.UTC)),
			sunrise.NewToken(uuid.New(), time.Now()),
			sunrise.NewToken(uuid.New(), time.Now().Add(312391*time.Hour)),
		}

		for i, token := range testCases {
			data, err := token.MarshalBinary()
			require.NotNil(t, data, "test case %d returned nil data", i)
			require.NoError(t, err, "test case %d errored on marshal", i)

			cmpt := &sunrise.Token{}
			err = cmpt.UnmarshalBinary(data)
			require.NoError(t, err, "test case %d errored on unmarshal", i)

			require.True(t, token.Equal(cmpt), "deserialization mismatch for test case %d", i)
		}
	})

	t.Run("BadMarshal", func(t *testing.T) {
		testCases := []struct {
			token *sunrise.Token
			err   error
		}{
			{
				sunrise.NewToken(uuid.Nil, time.Now()),
				sunrise.ErrInvalidEnvelopeID,
			},
			{
				&sunrise.Token{EnvelopeID: uuid.New(), Expiration: time.Time{}},
				sunrise.ErrInvalidExpiration,
			},
		}

		for i, tc := range testCases {
			data, err := tc.token.MarshalBinary()
			require.Nil(t, data, "test case %d returned non-nil data", i)
			require.ErrorIs(t, err, tc.err, "test case %d return the wrong error", i)
		}
	})

	t.Run("BadUnmarshal", func(t *testing.T) {
		testCases := []struct {
			data []byte
			err  error
		}{
			{
				nil,
				sunrise.ErrSize,
			},
			{
				[]byte{},
				sunrise.ErrSize,
			},
			{
				[]byte{0x1, 0x2, 0x3, 0x4, 0xf, 0xfe},
				sunrise.ErrSize,
			},
			{
				bytes.Repeat([]byte{0x1, 0x2, 0x3, 0x4, 0xf, 0xfe}, 64),
				sunrise.ErrSize,
			},
			{
				[]byte{
					0x22, 0x1, 0x33, 0x41, 0xd3, 0x7a, 0x12, 0xc2, 0xab, 0x41, 0x0, 0xfc, 0xe1, 0x7b, 0x7d, 0x15,
					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
					0x22, 0x1, 0x33, 0x41, 0xd3, 0x7a, 0x12, 0xc2, 0xab, 0x41, 0x0, 0xfc, 0xe1, 0x7b, 0x7d, 0x15,
					0x22, 0x1, 0x33, 0x41, 0xd3, 0x7a, 0x12, 0xc2, 0xab, 0x41, 0x0, 0xfc, 0xe1, 0x7b, 0x7d, 0x15,
					0x22, 0x1, 0x33, 0x41, 0xd3, 0x7a, 0x12, 0xc2, 0xab, 0x41, 0x0, 0xfc, 0xe1, 0x7b, 0x7d, 0x15,
					0x22, 0x1, 0x33, 0x41, 0xd3, 0x7a, 0x12, 0xc2, 0xab, 0x41, 0x0, 0xfc, 0xe1, 0x7b, 0x7d, 0x15,
				},
				sunrise.ErrDecode,
			},
			{
				[]byte{
					0x22, 0x1, 0x33, 0x41, 0xd3, 0x7a, 0x12, 0xc2, 0xab, 0x41, 0x0, 0xfc, 0xe1, 0x7b, 0x7d, 0x15,
					0xff, 0x00,
					0x22, 0x1, 0x33, 0x41, 0xd3, 0x7a, 0x12, 0xc2, 0xab, 0x41, 0x0, 0xfc, 0xe1, 0x7b, 0x7d, 0x15,
					0x22, 0x1, 0x33, 0x41, 0xd3, 0x7a, 0x12, 0xc2, 0xab, 0x41, 0x0, 0xfc, 0xe1, 0x7b, 0x7d, 0x15,
					0x22, 0x1, 0x33, 0x41, 0xd3, 0x7a, 0x12, 0xc2, 0xab, 0x41, 0x0, 0xfc, 0xe1, 0x7b, 0x7d, 0x15,
					0x22, 0x1, 0x33, 0x41, 0xd3, 0x7a, 0x12, 0xc2, 0xab, 0x41, 0x0, 0xfc, 0xe1, 0x7b, 0x7d,
				},
				sunrise.ErrInvalidNonce,
			},
		}

		for i, tc := range testCases {
			token := &sunrise.Token{}
			err := token.UnmarshalBinary(tc.data)
			require.ErrorIs(t, err, tc.err, "test case %d return the wrong error", i)
		}
	})
}

func TestSignedTokenBinary(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		testCases := []*sunrise.Token{
			sunrise.NewToken(uuid.MustParse("24035c84-ff3d-4da2-aef7-8683d9c00978"), time.Date(1994, 12, 20, 15, 21, 1, 3213, time.UTC)),
			sunrise.NewToken(uuid.New(), time.Now()),
			sunrise.NewToken(uuid.New(), time.Now().Add(312391*time.Hour)),
		}

		for i, token := range testCases {
			_, signed, err := token.Sign()
			require.NoError(t, err, "could not sign token")

			data, err := signed.MarshalBinary()
			require.NotNil(t, data, "test case %d returned nil data", i)
			require.NoError(t, err, "test case %d errored on marshal", i)

			cmpt := &sunrise.SignedToken{}
			err = cmpt.UnmarshalBinary(data)
			require.NoError(t, err, "test case %d errored on unmarshal", i)

			require.True(t, signed.Equal(cmpt), "deserialization mismatch for test case %d", i)
		}
	})

	t.Run("BadMarshal", func(t *testing.T) {
		testCases := []struct {
			token *sunrise.SignedToken
			err   error
		}{
			{
				&sunrise.SignedToken{},
				sunrise.ErrInvalidSignature,
			},
		}

		for i, tc := range testCases {
			data, err := tc.token.MarshalBinary()
			require.Nil(t, data, "test case %d returned non-nil data", i)
			require.ErrorIs(t, err, tc.err, "test case %d return the wrong error", i)
		}
	})

	t.Run("BadUnmarshal", func(t *testing.T) {
		testCases := []struct {
			data []byte
			err  error
		}{
			{
				nil,
				sunrise.ErrSize,
			},
			{
				[]byte{},
				sunrise.ErrSize,
			},
			{
				[]byte{0x1, 0x2, 0x3, 0x4, 0xf, 0xfe},
				sunrise.ErrSize,
			},
			{
				bytes.Repeat([]byte{0x1, 0x2, 0x3, 0x4, 0xf, 0xfe}, 64),
				sunrise.ErrSize,
			},
			{
				[]byte{
					0x22, 0x1, 0x33, 0x41, 0xd3, 0x7a, 0x12, 0xc2, 0xab, 0x41, 0x0, 0xfc, 0xe1, 0x7b, 0x7d, 0x15,
					0xff, 0x00,
					0x22, 0x1, 0x33, 0x41, 0xd3, 0x7a, 0x12, 0xc2, 0xab, 0x41, 0x0, 0xfc, 0xe1, 0x7b, 0x7d,
					0x22, 0x1, 0x33, 0x41, 0xd3, 0x7a, 0x12, 0xc2, 0xab, 0x41, 0x0, 0xfc, 0xe1, 0x7b, 0x7d, 0x15,
					0x22, 0x1, 0x33, 0x41, 0xd3, 0x7a, 0x12, 0xc2, 0xab, 0x41, 0x0, 0xfc, 0xe1, 0x7b, 0x7d, 0x15,
					0x22, 0x1, 0x33, 0x41, 0xd3, 0x7a, 0x12, 0xc2, 0xab, 0x41, 0x0, 0xfc, 0xe1, 0x7b, 0x7d, 0x15,
					0x22, 0x1, 0x33, 0x41, 0xd3, 0x7a, 0x12, 0xc2, 0xab, 0x41, 0x0, 0xfc, 0xe1, 0x7b, 0x7d, 0x15,
					0x22, 0x1, 0x33, 0x41, 0xd3, 0x7a, 0x12, 0xc2, 0xab, 0x41, 0x0, 0xfc, 0xe1, 0x7b, 0x7d, 0x15,
				},
				sunrise.ErrInvalidSignature,
			},
		}

		for i, tc := range testCases {
			token := &sunrise.SignedToken{}
			err := token.UnmarshalBinary(tc.data)
			require.ErrorIs(t, err, tc.err, "test case %d return the wrong error", i)
		}
	})
}

func TestVerificationToken(t *testing.T) {
	t.Run("Static", func(t *testing.T) {
		tks := "k0ZmbMJcQeyFtAtZ0_2EXMHwJ1ufcB4831ozVeHzAcVpyKybKzelG0l9qbJ4K5IUjaGSx5EdJ_9rSR8RVry3g13DJ-Dh4NktFFSY0ULIkMY"
		token, err := sunrise.ParseVerification(tks)
		require.NoError(t, err, "could not parse good verification token")
		require.Equal(t, token.EnvelopeID(), uuid.MustParse("9346666c-c25c-41ec-85b4-0b59d3fd845c"), "unexpected envelope id")

		secret := []byte{
			0xc1, 0xf0, 0x27, 0x5b, 0x9f, 0x70, 0x1e, 0x3c, 0xdf, 0x5a, 0x33, 0x55, 0xe1, 0xf3, 0x1, 0xc5,
			0x69, 0xc8, 0xac, 0x9b, 0x2b, 0x37, 0xa5, 0x1b, 0x49, 0x7d, 0xa9, 0xb2, 0x78, 0x2b, 0x92, 0x14,
			0x8d, 0xa1, 0x92, 0xc7, 0x91, 0x1d, 0x27, 0xff, 0x6b, 0x49, 0x1f, 0x11, 0x56, 0xbc, 0xb7, 0x83,
			0x5d, 0xc3, 0x27, 0xe0, 0xe1, 0xe0, 0xd9, 0x2d, 0x14, 0x54, 0x98, 0xd1, 0x42, 0xc8, 0x90, 0xc6,
		}

		require.Equal(t, secret, token.Secret(), "unexpected secret")
	})

	t.Run("TooShort", func(t *testing.T) {
		tks := "k0ZmbMJcQeyFtAtZ0_2EXMHwJ1ufcB4831ozVeHzAcVpyKybKzelG0l9qbJ4K5IUjaGSx5EdJ_"
		token, err := sunrise.ParseVerification(tks)
		require.ErrorIs(t, err, sunrise.ErrSize, "expected size parsing error")
		require.Nil(t, token, "expected nil token returned")
	})

	t.Run("BadDecode", func(t *testing.T) {
		tks := "k0ZmbMJcQeyFtAtZ0_2}XMHwJ1ufcB4831ozVeHzAcVpyKybKzelG0l9qbJ4K5IUjaGSx5EdJ_"
		token, err := sunrise.ParseVerification(tks)
		require.EqualError(t, err, "illegal base64 data at input byte 19", "expected base64 parsing error")
		require.Nil(t, token, "expected nil token returned")
	})
}
