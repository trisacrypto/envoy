package models

import (
	"database/sql"
	"time"

	"self-hosted-node/pkg/store/errors"
	"self-hosted-node/pkg/ulids"

	"github.com/google/uuid"
	api "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
)

type Transaction struct {
	ID                 uuid.UUID         // Transaction IDs are UUIDs not ULIDs per the TRISA spec, this is also used for the envelope ID
	Source             string            // Either "local" meaning the transaction was created by the user, or "remote" meaning it is an incoming message
	Status             string            // Can be "draft", "pending", "action required", "completed", "archived"
	Counterparty       string            // The name of the counterparty in the transaction
	CounterpartyID     ulids.NullULID    // A reference to the counterparty in the database, if any
	Originator         sql.NullString    // Full name of the originator natural person or account
	OriginatorAddress  sql.NullString    // The crypto address of the originator
	Beneficiary        sql.NullString    // Full name of the beneficiary natural person or account
	BeneficiaryAddress sql.NullString    // The crypto address of the beneficiary
	VirtualAsset       string            // A representation of the network/asset type
	Amount             float64           // The amount of the transaction
	LastUpdate         sql.NullTime      // The last time a TRISA RPC occurred for this transaction
	Created            time.Time         // Timestamp the transaction was created
	Modified           time.Time         // Timestamp the transaction was last modified, including when a new secure envelope was received
	numEnvelopes       int64             // The number of secure envelopes associated with the transaction
	envelopes          []*SecureEnvelope // Associated secure envelopes
}

type SecureEnvelope struct {
	Model
	EnvelopeID    uuid.UUID           // Also a foreign key reference to the Transaction
	Direction     string              // Either "out" outgoing or "in" incoming
	IsError       bool                // If the envelope contains an error/rejection rather than a payload
	EncryptionKey []byte              // The encryption key, encrypted with the public key of the local node. Note this may differ from the value in the envelope for outgoing messages
	HMACSecret    []byte              // The hmac secret, encrypted with the public key of the local node. Note that this may differ from the value in the envelope for outgoing messages
	Timestamp     time.Time           // The timestamp of the envelope as defined by the envelope
	PublicKey     string              // The signature of the public key that sealed the encryption key and hmac secret, may differ from the value in the envelope for ougoing envelopes.
	Envelope      *api.SecureEnvelope // The secure envelope protocol buffer stored as a BLOB
	transaction   *Transaction        // The transaction this envelope is associated with
}

func (t *Transaction) Scan(scanner Scanner) error {
	return scanner.Scan(
		&t.ID,
		&t.Source,
		&t.Status,
		&t.Counterparty,
		&t.CounterpartyID,
		&t.Originator,
		&t.OriginatorAddress,
		&t.Beneficiary,
		&t.BeneficiaryAddress,
		&t.VirtualAsset,
		&t.Amount,
		&t.LastUpdate,
		&t.Created,
		&t.Modified,
	)
}

func (t *Transaction) ScanWithCount(scanner Scanner) error {
	return scanner.Scan(
		&t.ID,
		&t.Source,
		&t.Status,
		&t.Counterparty,
		&t.CounterpartyID,
		&t.Originator,
		&t.OriginatorAddress,
		&t.Beneficiary,
		&t.BeneficiaryAddress,
		&t.VirtualAsset,
		&t.Amount,
		&t.LastUpdate,
		&t.Created,
		&t.Modified,
		&t.numEnvelopes,
	)
}

func (t *Transaction) Params() []any {
	return []any{
		sql.Named("id", t.ID),
		sql.Named("source", t.Source),
		sql.Named("status", t.Status),
		sql.Named("counterparty", t.Counterparty),
		sql.Named("counterpartyID", t.CounterpartyID),
		sql.Named("originator", t.Originator),
		sql.Named("originatorAddress", t.OriginatorAddress),
		sql.Named("beneficiary", t.Beneficiary),
		sql.Named("beneficiaryAddress", t.BeneficiaryAddress),
		sql.Named("virtualAsset", t.VirtualAsset),
		sql.Named("amount", t.Amount),
		sql.Named("lastUpdate", t.LastUpdate),
		sql.Named("created", t.Created),
		sql.Named("modified", t.Modified),
	}
}

func (t *Transaction) NumEnvelopes() int64 {
	if len(t.envelopes) > 0 {
		return int64(len(t.envelopes))
	}
	return t.numEnvelopes
}

func (t *Transaction) SetNumEnvelopes(count int64) {
	t.numEnvelopes = count
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
	return scanner.Scan(
		&e.ID,
		&e.EnvelopeID,
		&e.Direction,
		&e.IsError,
		&e.EncryptionKey,
		&e.HMACSecret,
		&e.Timestamp,
		&e.PublicKey,
		&e.Envelope,
		&e.Created,
		&e.Modified,
	)
}

func (e *SecureEnvelope) Params() []any {
	return []any{
		sql.Named("id", e.ID),
		sql.Named("envelopeID", e.EnvelopeID),
		sql.Named("direction", e.Direction),
		sql.Named("isError", e.IsError),
		sql.Named("encryptionKey", e.EncryptionKey),
		sql.Named("hmacSecret", e.HMACSecret),
		sql.Named("timestamp", e.Timestamp),
		sql.Named("publicKey", e.PublicKey),
		sql.Named("envelope", e.Envelope),
		sql.Named("created", e.Created),
		sql.Named("modified", e.Modified),
	}
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
