package sunrise

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const (
	nonceLength    = 64
	keyLength      = 64
	minTokenLength = 16 + nonceLength + 1
	maxTokenLength = 16 + nonceLength + binary.MaxVarintLen64
)

type Token struct {
	EnvelopeID uuid.UUID
	Expiration time.Time
	nonce      []byte
}

func NewToken(envelopeID uuid.UUID, expiration time.Time) *Token {
	token := &Token{
		EnvelopeID: envelopeID,
		Expiration: expiration,
		nonce:      make([]byte, nonceLength),
	}

	if _, err := rand.Read(token.nonce); err != nil {
		panic(fmt.Errorf("no crypto random generator available: %w", err))
	}

	return token
}

func (t *Token) MarshalBinary() ([]byte, error) {
	if err := t.Validate(); err != nil {
		return nil, err
	}

	data := make([]byte, maxTokenLength)
	copy(data[:16], t.EnvelopeID[:])

	i := binary.PutVarint(data[16:], t.Expiration.UnixNano())
	copy(data[16+i:], t.nonce)

	return data[:16+i+nonceLength], nil
}

func (t *Token) UnmarshalBinary(data []byte) error {
	if len(data) > maxTokenLength || len(data) < minTokenLength {
		return ErrSize
	}

	// Parse envelope ID
	t.EnvelopeID = uuid.UUID(data[:16])

	// Parse expiration time
	exp, i := binary.Varint(data[16:])
	if i <= 0 {
		return ErrDecode
	}
	t.Expiration = time.Unix(0, exp)

	// Read the nonce data
	t.nonce = data[16+i:]

	// Validate the binary data
	return t.Validate()
}

func (t *Token) Validate() (err error) {
	if t.EnvelopeID == uuid.Nil {
		err = errors.Join(err, ErrInvalidEnvelopeID)
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
	return bytes.Equal(t.EnvelopeID[:], o.EnvelopeID[:]) &&
		t.Expiration.Equal(o.Expiration) &&
		bytes.Equal(t.nonce, o.nonce)
}
