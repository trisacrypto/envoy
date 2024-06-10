package models

import (
	"crypto/rsa"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/ulids"

	"github.com/google/uuid"
	api "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
	"github.com/trisacrypto/trisa/pkg/trisa/crypto"
	"github.com/trisacrypto/trisa/pkg/trisa/crypto/rsaoeap"
	"github.com/trisacrypto/trisa/pkg/trisa/envelope"
	"github.com/trisacrypto/trisa/pkg/trisa/keys"
)

const (
	SourceLocal       = "local"
	SourceRemote      = "remote"
	StatusDraft       = "draft"
	StatusPending     = "pending"
	StatusAction      = "action required"
	StatusComplete    = "completed"
	StatusArchived    = "archived"
	StatusErrored     = "errored"
	DirectionOut      = "out"
	DirectionOutgoing = DirectionOut
	DirectionIn       = "in"
	DirectionIncoming = DirectionIn
	DirectionAny      = "any"
)

func ValidStatus(status string) bool {
	status = strings.TrimSpace(strings.ToLower(status))
	switch status {
	case StatusDraft, StatusPending, StatusAction, StatusComplete, StatusArchived, StatusErrored:
		return true
	default:
		return false
	}
}

type Transaction struct {
	ID                 uuid.UUID         // Transaction IDs are UUIDs not ULIDs per the TRISA spec, this is also used for the envelope ID
	Source             string            // Either "local" meaning the transaction was created by the user, or "remote" meaning it is an incoming message
	Status             string            // Can be "draft", "pending", "action required", "completed", "archived", "errored"
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
	ValidHMAC     sql.NullBool        // If the hmac has been validated against the payload and non-repudiation properties are satisfied
	Timestamp     time.Time           // The timestamp of the envelope as defined by the envelope
	PublicKey     sql.NullString      // The signature of the public key that sealed the encryption key and hmac secret, may differ from the value in the envelope for ougoing envelopes.
	Envelope      *api.SecureEnvelope // The secure envelope protocol buffer stored as a BLOB
	transaction   *Transaction        // The transaction this envelope is associated with
}

// PreparedTransaction allows you to manage the creation/modification of a transaction
// w.r.t a secure envelope. It is unified in a single interface to allow backend stores
// that have database transactions to perform all operations in a single transaction
// without concurrency issues.
type PreparedTransaction interface {
	Created() bool                       // Returns true if the transaction was newly created, false if it already existed
	Fetch() (*Transaction, error)        // Fetches the current transaction record from the database
	Update(*Transaction) error           // Update the transaction with new information; e.g. data from decryption
	AddCounterparty(*Counterparty) error // Add counterparty by database ULID, counterparty name, or registered directory ID; if the counterparty doesn't exist, it is created
	AddEnvelope(*SecureEnvelope) error   // Associate a secure envelope with the prepared transaction
	Rollback() error                     // Rollback the prepared transaction and conclude it
	Commit() error                       // Commit the prepared transaction and conclude it
}

// EnvelopeStorage is a subset of the PreparedTransaction interface that focuses on
// the database interactions with SecureEnvelopes. This sub-interface makes it easier to
// write tests, though in practice the PreparedTransaction is passed in for this.
type EnvelopeStorage interface {
	AddEnvelope(*SecureEnvelope) error
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

// Update the transaction t with values from other if the field in other is non-zero;
// e.g. if a nullable field is valid or an empty string is empty. This method skips the
// ID and Modified fields.
func (t *Transaction) Update(other *Transaction) {
	if other.Source != "" {
		t.Source = other.Source
	}

	if other.Status != "" {
		t.Status = other.Status
	}

	if other.Counterparty != "" {
		t.Counterparty = other.Counterparty
	}

	if other.CounterpartyID.Valid {
		t.CounterpartyID = other.CounterpartyID
	}

	if other.Originator.Valid {
		t.Originator = other.Originator
	}

	if other.OriginatorAddress.Valid {
		t.OriginatorAddress = other.OriginatorAddress
	}

	if other.Beneficiary.Valid {
		t.Beneficiary = other.Beneficiary
	}

	if other.BeneficiaryAddress.Valid {
		t.BeneficiaryAddress = other.BeneficiaryAddress
	}

	if other.VirtualAsset != "" {
		t.VirtualAsset = other.VirtualAsset
	}

	if other.Amount != 0.0 {
		t.Amount = other.Amount
	}

	if other.LastUpdate.Valid {
		t.LastUpdate = other.LastUpdate
	}

	if !other.Created.IsZero() {
		t.Created = other.Created
	}
}

func FromEnvelope(env *envelope.Envelope) *SecureEnvelope {
	model := &SecureEnvelope{
		Direction: "",
		IsError:   env.IsError(),
		Envelope:  env.Proto(),
	}

	model.EnvelopeID, _ = env.UUID()
	model.EncryptionKey = model.Envelope.EncryptionKey
	model.HMACSecret = model.Envelope.HmacSecret
	model.Timestamp, _ = env.Timestamp()
	model.PublicKey = sql.NullString{Valid: model.Envelope.PublicKeySignature != "", String: model.Envelope.PublicKeySignature}

	return model
}

func FromOutgoingEnvelope(env *envelope.Envelope) *SecureEnvelope {
	out := FromEnvelope(env)
	out.Direction = DirectionOutgoing
	out.ValidHMAC = sql.NullBool{Valid: true, Bool: true}
	return out
}

func FromIncomingEnvelope(env *envelope.Envelope) *SecureEnvelope {
	in := FromEnvelope(env)
	in.Direction = DirectionIncoming
	in.ValidHMAC = sql.NullBool{Valid: true, Bool: in.Envelope.Sealed}

	in.PublicKey.String = env.Proto().PublicKeySignature
	in.PublicKey.Valid = in.PublicKey.String != ""

	return in
}

func (e *SecureEnvelope) Reseal(storageKey keys.PublicKey, sec crypto.Crypto) (err error) {
	// Set the public key signature of the storage key on the model
	if e.PublicKey.String, err = storageKey.PublicKeySignature(); err != nil {
		return err
	}

	// Ensure the null value is set to valid
	e.PublicKey.Valid = e.PublicKey.String != ""

	// Create a cipher to seal the new storage keys
	var (
		pubkey interface{}
		seal   crypto.Cipher
	)

	if pubkey, err = storageKey.SealingKey(); err != nil {
		return err
	}

	switch t := pubkey.(type) {
	case *rsa.PublicKey:
		if seal, err = rsaoeap.New(t); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown key type %T", t)
	}

	// Encrypt the encryption key and hmac secret with the new cipher
	if e.EncryptionKey, err = seal.Encrypt(sec.EncryptionKey()); err != nil {
		return err
	}

	if e.HMACSecret, err = seal.Encrypt(sec.HMACSecret()); err != nil {
		return err
	}

	return nil
}

func (e *SecureEnvelope) Scan(scanner Scanner) error {
	return scanner.Scan(
		&e.ID,
		&e.EnvelopeID,
		&e.Direction,
		&e.IsError,
		&e.EncryptionKey,
		&e.HMACSecret,
		&e.ValidHMAC,
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
		sql.Named("validHMAC", e.ValidHMAC),
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
