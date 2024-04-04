package models

import (
	"self-hosted-node/pkg/store/errors"
	"time"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
)

type Transaction struct {
	ID             uuid.UUID         // Transaction IDs are UUIDs not ULIDs per the TRISA spec, this is also used for the envelope ID
	Source         string            // Either "local" meaning the transaction was created by the user, or "remote" meaning it is an incoming message
	Status         string            // Can be "draft", "pending", "action required", "completed", "archived"
	Counterparty   string            // The name of the counterparty in the transaction
	CounterpartyID ulid.ULID         // A reference to the counterparty in the database, if any
	Created        time.Time         // Timestamp the transaction was created
	Modified       time.Time         // Timestamp the transaction was last modified, including when a new secure envelope was received
	envelopes      []*SecureEnvelope // Associated secure envelopes
}

type SecureEnvelope struct {
	Model
	transaction *Transaction // The transaction this envelope is associated with
}

func (t *Transaction) Scan(scanner Scanner) error {
	return scanner.Scan()
}

func (t *Transaction) ScanSummary(scanner Scanner) error {
	return scanner.Scan()
}

func (t *Transaction) Params() []any {
	return []any{}
}

func (t *Transaction) SecureEnvelopes() ([]*SecureEnvelope, error) {
	if t.envelopes == nil {
		return nil, errors.ErrMissingAssociation
	}
	return t.envelopes, nil
}

func (t *Transaction) SetSecureEnvelopes(envelopes []*SecureEnvelope) {
	t.envelopes = envelopes
}

func (e *SecureEnvelope) Scan(scanner Scanner) error {
	return scanner.Scan()
}

func (e *SecureEnvelope) Params() []any {
	return []any{}
}

func (e *SecureEnvelope) Transaction() (*Transaction, error) {
	if e.transaction == nil {
		return nil, errors.ErrMissingAssociation
	}
	return e.transaction, nil
}

func (e *SecureEnvelope) SetTransaction(tx *Transaction) {
	e.transaction = tx
}
