package models

import (
	"self-hosted-node/pkg/store/errors"
)

type Transaction struct {
	Model
	envelopes []*SecureEnvelope // Associated secure envelopes
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
