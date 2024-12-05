package sunrise

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql/driver"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/trisacrypto/envoy/pkg/ulids"
)

const (
	nonceLength        = 64
	keyLength          = 64
	hmacLength         = 32
	minTokenLength     = 16 + nonceLength + 1
	maxTokenLength     = 16 + nonceLength + binary.MaxVarintLen64
	minSignTokenLength = minTokenLength + hmacLength
	maxSignTokenLength = maxTokenLength + hmacLength
	verifyTokenLength  = 16 + keyLength
	defaultTTL         = 7 * 24 * time.Hour
)

// A Token is a data representation of the information needed to create a secure
// sunrise verification token to send to the compliance officer of the counterparty.
// Tokens can be used to generate SignedTokens and SignedTokens can be used to send a
// secure verification token and to verify that tokens belong to the specified user.
type Token struct {
	SunriseID  ulid.ULID // ID of the sunrise record in the database
	Expiration time.Time // Expiration date of the token (not after)
	nonce      []byte    // Random nonce for cryptographic security
}

// A signed token contains a signature that can be stored in the local database in
// order to verify an incoming verification token from a client.
type SignedToken struct {
	Token
	signature []byte // The HMAC signature computed from the Token data (read-only)
}

// A verification token is sent to the client and contains the information needed to
// lookup a signed token in the database and to verify that the message is authentic.
type VerificationToken []byte

//===========================================================================
// Token Methods
//===========================================================================

// Create a new token with the specified ID and expiration timestamp. If the timestamp
// is zero valued, then a timestamp in the future will be generated with the default
// expiration deadline.
func NewToken(sunriseID ulid.ULID, expiration time.Time) *Token {
	if expiration.IsZero() {
		expiration = time.Now().Add(defaultTTL)
	}

	token := &Token{
		SunriseID:  sunriseID,
		Expiration: expiration,
		nonce:      make([]byte, nonceLength),
	}

	if _, err := rand.Read(token.nonce); err != nil {
		panic(fmt.Errorf("no crypto random generator available: %w", err))
	}

	return token
}

// Sign a token creating a verification token that should be sent as a string to the
// counterparty and a signed token that should be stored in the database.
func (t *Token) Sign() (token VerificationToken, signature *SignedToken, err error) {
	// Generate nonce if the token was instantiated without New
	if t.nonce == nil {
		t.nonce = make([]byte, nonceLength)
		if _, err := rand.Read(t.nonce); err != nil {
			panic(fmt.Errorf("no crypto random generator available: %w", err))
		}
	}

	// Create a random secret key for signing
	secret := make([]byte, keyLength)
	if _, err := rand.Read(secret); err != nil {
		panic(fmt.Errorf("no crypto random generator available: %w", err))
	}

	// Marshal the token for signing
	var data []byte
	if data, err = t.MarshalBinary(); err != nil {
		return nil, nil, err
	}

	// Create HMAC signature for the token
	mac := hmac.New(sha256.New, secret)
	if _, err = mac.Write(data); err != nil {
		return nil, nil, err
	}

	// Get the HMAC signature and append it to the verification data
	// NOTE: this must happen after HMAC signing!
	signature = &SignedToken{
		Token:     *t,
		signature: mac.Sum(nil),
	}

	// Create the verification token
	token = make(VerificationToken, verifyTokenLength)
	copy(token[0:16], t.SunriseID[:])
	copy(token[16:], secret)

	return token, signature, nil
}

func (t *Token) IsExpired() bool {
	if t.Expiration.IsZero() {
		return true
	}
	return t.Expiration.Before(time.Now())
}

func (t *Token) MarshalBinary() ([]byte, error) {
	if err := t.Validate(); err != nil {
		return nil, err
	}

	data := make([]byte, maxTokenLength)
	copy(data[:16], t.SunriseID[:])

	i := binary.PutVarint(data[16:], t.Expiration.UnixNano())
	l := 16 + i
	copy(data[l:], t.nonce)

	l = l + nonceLength
	return data[:l], nil
}

func (t *Token) UnmarshalBinary(data []byte) error {
	if _, err := t.readFrom(data); err != nil {
		return err
	}
	return t.Validate()
}

func (t *Token) readFrom(data []byte) (int, error) {
	if len(data) > maxTokenLength || len(data) < minTokenLength {
		return 0, ErrSize
	}

	// Parse sunrise ID
	t.SunriseID = ulid.ULID(data[:16])

	// Parse expiration time
	exp, i := binary.Varint(data[16 : 16+binary.MaxVarintLen64])
	if i <= 0 {
		return 16, ErrDecode
	}
	t.Expiration = time.Unix(0, exp)

	// Read the nonce data
	l := 16 + i
	if len(data[l:]) < nonceLength {
		return l, ErrInvalidNonce
	}
	t.nonce = data[l : l+nonceLength]

	// Validate the binary data
	return l + nonceLength, nil
}

func (t *Token) Validate() (err error) {
	if ulids.IsZero(t.SunriseID) {
		err = errors.Join(err, ErrInvalidSunriseID)
	}

	if t.Expiration.IsZero() {
		err = errors.Join(err, ErrInvalidExpiration)
	}

	if len(t.nonce) != nonceLength {
		err = errors.Join(err, ErrInvalidNonce)
	}

	return err
}

func (t *Token) Equal(o *Token) bool {
	return bytes.Equal(t.SunriseID[:], o.SunriseID[:]) &&
		t.Expiration.Equal(o.Expiration) &&
		bytes.Equal(t.nonce, o.nonce)
}

//===========================================================================
// SignedToken Methods
//===========================================================================

// Verify that a signed token belongs with the associated verification token.
func (t *SignedToken) Verify(token VerificationToken) (secure bool, err error) {
	if len(token) != verifyTokenLength {
		return false, ErrSize
	}

	// Compute the hash of the current token for verification
	var data []byte
	if data, err = t.Token.MarshalBinary(); err != nil {
		return false, err
	}

	// Generate the HMAC signature of the current token
	mac := hmac.New(sha256.New, token.Secret())
	if _, err := mac.Write(data); err != nil {
		return false, err
	}

	return bytes.Equal(t.signature, mac.Sum(nil)), nil
}

// Retrieve the signature from the signed token.
func (t *SignedToken) Signature() []byte {
	return t.signature
}

// Scan the signed token from a database query.
func (t *SignedToken) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	data, ok := value.([]byte)
	if !ok {
		return ErrUnexpectedType
	}

	return t.UnmarshalBinary(data)
}

// Produce a database value from the signed token for inserts/updates to database.
func (t *SignedToken) Value() (_ driver.Value, err error) {
	if t == nil {
		return nil, nil
	}

	var data []byte
	if data, err = t.MarshalBinary(); err != nil {
		return nil, err
	}
	return data, nil
}

func (t *SignedToken) MarshalBinary() (out []byte, err error) {
	if err := t.Validate(); err != nil {
		return nil, err
	}

	var token []byte
	if token, err = t.Token.MarshalBinary(); err != nil {
		return nil, err
	}

	out = make([]byte, maxSignTokenLength)
	copy(out[:len(token)], token)
	copy(out[len(token):], t.signature)

	return out[:len(token)+len(t.signature)], nil
}

func (t *SignedToken) UnmarshalBinary(data []byte) (err error) {
	if len(data) > maxSignTokenLength || len(data) < minSignTokenLength {
		return ErrSize
	}

	// Parse Token
	var n int
	if n, err = t.Token.readFrom(data[:maxTokenLength]); err != nil {
		return err
	}

	// Extract the signature as the unread part of the data
	t.signature = data[n:]

	// Validate the binary data
	return t.Validate()
}

func (t *SignedToken) Validate() (err error) {
	err = t.Token.Validate()

	if len(t.signature) != hmacLength {
		err = errors.Join(err, ErrInvalidSignature)
	}

	return err
}

func (t *SignedToken) Equal(o *SignedToken) bool {
	return t.Token.Equal(&o.Token) && bytes.Equal(t.signature, o.signature)
}

//===========================================================================
// VerificationToken Methods
//===========================================================================

func ParseVerification(tks string) (_ VerificationToken, err error) {
	var token []byte
	if token, err = base64.RawURLEncoding.DecodeString(tks); err != nil {
		return nil, err
	}

	if len(token) != verifyTokenLength {
		return nil, ErrSize
	}

	return VerificationToken(token), nil
}

func (v VerificationToken) SunriseID() ulid.ULID {
	return ulid.ULID(v[:16])
}

func (v VerificationToken) Secret() []byte {
	return v[16:]
}

func (v VerificationToken) String() string {
	return base64.RawURLEncoding.EncodeToString(v)
}
