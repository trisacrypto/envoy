package api

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/envoy/pkg/ulids"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
	"github.com/trisacrypto/trisa/pkg/ivms101"
	trisa "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
	generic "github.com/trisacrypto/trisa/pkg/trisa/data/generic/v1beta1"
	"github.com/trisacrypto/trisa/pkg/trisa/envelope"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/anypb"
)

//===========================================================================
// Transaction Resource
//===========================================================================

const (
	DetailFull    = "full"
	DetailPreview = "preview"
)

type Transaction struct {
	ID                 uuid.UUID  `json:"id"`
	Source             string     `json:"source"`
	Status             string     `json:"status"`
	Counterparty       string     `json:"counterparty"`
	CounterpartyID     ulid.ULID  `json:"counterparty_id,omitempty"`
	Originator         string     `json:"originator,omitempty"`
	OriginatorAddress  string     `json:"orginator_address,omitempty"`
	Beneficiary        string     `json:"beneficiary,omitempty"`
	BeneficiaryAddress string     `json:"beneficiary_address,omitempty"`
	VirtualAsset       string     `json:"virtual_asset"`
	Amount             float64    `json:"amount"`
	LastUpdate         *time.Time `json:"last_update,omitempty"`
	EnvelopeCount      int64      `json:"envelope_count,omitempty"`
	Created            time.Time  `json:"created"`
	Modified           time.Time  `json:"modified"`
}

type SecureEnvelope struct {
	ID                  ulid.ULID    `json:"id"`
	Direction           string       `json:"direction"`
	EnvelopeID          uuid.UUID    `json:"envelope_id"`
	Payload             []byte       `json:"payload,omitempty"`
	EncryptionKey       []byte       `json:"encryption_key,omitempty"`
	EncryptionAlgorithm string       `json:"encryption_algorithm,omitempty"`
	ValidHMAC           bool         `json:"valid_hmac"`
	HMAC                []byte       `json:"hmac,omitempty"`
	HMACSecret          []byte       `json:"hmac_secret,omitempty"`
	HMACAlgorithm       string       `json:"hmac_algorithm,omitempty"`
	IsError             bool         `json:"is_error"`
	Error               *trisa.Error `json:"error,omitempty"`
	Timestamp           time.Time    `json:"timestamp"`
	Sealed              bool         `json:"sealed"`
	PublicKeySignature  string       `json:"public_key_signature,omitempty"`
	Original            []byte       `json:"original,omitempty"`
}

type Envelope struct {
	Error       *trisa.Error             `json:"error,omitempty"`
	Identity    *ivms101.IdentityPayload `json:"identity,omitempty"`
	Transaction *generic.Transaction     `json:"transaction,omitempty"`
	Pending     *generic.Pending         `json:"pending,omitempty"`
	SentAt      *time.Time               `json:"sent_at"`
	ReceivedAt  *time.Time               `json:"received_at,omitempty"`
}

type Rejection struct {
	Code         string `json:"code"`
	Message      string `json:"message"`
	RequestRetry bool   `json:"request_retry"`
}

type TransactionQuery struct {
	Detail string `json:"detail" url:"detail,omitempty" form:"detail"`
}

type EnvelopeQuery struct {
	Decrypt  bool `json:"decrypt" url:"decrypt,omitempty" form:"decrypt"`
	Archives bool `json:"archives" url:"archives,omitempty" form:"archives"`
}

type EnvelopeListQuery struct {
	PageQuery
	EnvelopeQuery
}

type TransactionsList struct {
	Page         *PageQuery     `json:"page"`
	Transactions []*Transaction `json:"transactions"`
}

type EnvelopesList struct {
	Page               *PageQuery        `json:"page"`
	IsDecrypted        bool              `json:"is_decrypted"`
	SecureEnvelopes    []*SecureEnvelope `json:"secure_envelopes,omitempty"`
	DecryptedEnvelopes []*Envelope       `json:"decrypted_envelopes,omitempty"`
}

//===========================================================================
// Transactions
//===========================================================================

func NewTransaction(model *models.Transaction) (*Transaction, error) {
	tx := &Transaction{
		ID:                 model.ID,
		Source:             model.Source,
		Status:             model.Status,
		Counterparty:       model.Counterparty,
		CounterpartyID:     model.CounterpartyID.ULID,
		Originator:         model.Originator.String,
		OriginatorAddress:  model.OriginatorAddress.String,
		Beneficiary:        model.Beneficiary.String,
		BeneficiaryAddress: model.BeneficiaryAddress.String,
		VirtualAsset:       model.VirtualAsset,
		Amount:             model.Amount,
		EnvelopeCount:      model.NumEnvelopes(),
		Created:            model.Created,
		Modified:           model.Modified,
	}

	// If last update is not NULL in the database, then add it to the response.
	if model.LastUpdate.Valid {
		tx.LastUpdate = &model.LastUpdate.Time
	}
	return tx, nil
}

func NewTransactionList(page *models.TransactionPage) (out *TransactionsList, err error) {
	out = &TransactionsList{
		Page:         &PageQuery{},
		Transactions: make([]*Transaction, 0, len(page.Transactions)),
	}

	for _, model := range page.Transactions {
		var tx *Transaction
		if tx, err = NewTransaction(model); err != nil {
			return nil, err
		}
		out.Transactions = append(out.Transactions, tx)
	}

	return out, nil
}

func (c *Transaction) Validate() (err error) {
	if c.Source == "" {
		err = ValidationError(err, MissingField("source"))
	}

	if c.Source != models.SourceLocal && c.Source != models.SourceRemote {
		err = ValidationError(err, IncorrectField("source", "source must either be local or remote"))
	}

	c.Status = strings.TrimSpace(strings.ToLower(c.Status))
	if c.Status == "" {
		err = ValidationError(err, MissingField("status"))
	} else if !models.ValidStatus(c.Status) {
		err = ValidationError(err, IncorrectField("status", "status must be one of draft, pending, action required, completed, or archived"))
	}

	if c.Counterparty == "" {
		err = ValidationError(err, MissingField("counterparty"))
	}

	if c.VirtualAsset == "" {
		err = ValidationError(err, MissingField("virtual_asset"))
	}

	if c.Amount == 0.0 {
		err = ValidationError(err, MissingField("amount"))
	}

	return err
}

func (c *Transaction) Model() (model *models.Transaction, err error) {
	model = &models.Transaction{
		ID:                 c.ID,
		Source:             c.Source,
		Status:             c.Status,
		Counterparty:       c.Counterparty,
		CounterpartyID:     ulids.NullULID{ULID: c.CounterpartyID, Valid: !ulids.IsZero(c.CounterpartyID)},
		Originator:         sql.NullString{String: c.Originator, Valid: c.Originator != ""},
		OriginatorAddress:  sql.NullString{String: c.OriginatorAddress, Valid: c.OriginatorAddress != ""},
		Beneficiary:        sql.NullString{String: c.Beneficiary, Valid: c.Beneficiary != ""},
		BeneficiaryAddress: sql.NullString{String: c.BeneficiaryAddress, Valid: c.BeneficiaryAddress != ""},
		VirtualAsset:       c.VirtualAsset,
		Amount:             c.Amount,
	}

	if c.LastUpdate != nil {
		model.LastUpdate = sql.NullTime{Valid: !c.LastUpdate.IsZero(), Time: *c.LastUpdate}
	}

	return model, nil
}

//===========================================================================
// SecureEnvelopes
//===========================================================================

func NewSecureEnvelope(model *models.SecureEnvelope) (out *SecureEnvelope, err error) {
	out = &SecureEnvelope{
		ID:                  model.ID,
		Direction:           model.Direction,
		EnvelopeID:          model.EnvelopeID,
		Payload:             model.Envelope.Payload,
		EncryptionKey:       model.EncryptionKey,
		EncryptionAlgorithm: model.Envelope.EncryptionAlgorithm,
		ValidHMAC:           model.ValidHMAC.Bool,
		HMAC:                model.Envelope.Hmac,
		HMACSecret:          model.HMACSecret,
		HMACAlgorithm:       model.Envelope.HmacAlgorithm,
		IsError:             model.IsError,
		Error:               model.Envelope.Error,
		Timestamp:           model.Timestamp,
		Sealed:              model.Envelope.Sealed,
		PublicKeySignature:  model.PublicKey.String,
	}

	if out.Original, err = proto.Marshal(model.Envelope); err != nil {
		return nil, err
	}
	return out, nil
}

func NewSecureEnvelopeList(page *models.SecureEnvelopePage) (out *EnvelopesList, err error) {
	out = &EnvelopesList{
		Page:            &PageQuery{},
		SecureEnvelopes: make([]*SecureEnvelope, 0, len(page.Envelopes)),
	}

	for _, model := range page.Envelopes {
		// TODO: how to validate HMAC signature?
		var env *SecureEnvelope
		if env, err = NewSecureEnvelope(model); err != nil {
			return nil, err
		}
		out.SecureEnvelopes = append(out.SecureEnvelopes, env)
	}

	return out, nil
}

//===========================================================================
// Envelopes
//===========================================================================

func NewEnvelope(env *envelope.Envelope) (out *Envelope, err error) {
	switch state := env.State(); state {
	case envelope.Error:
		return &Envelope{Error: env.Error()}, nil
	case envelope.Clear:
		break
	default:
		return nil, fmt.Errorf("envelope is in an unhandled state: %s", state)
	}

	out = &Envelope{}

	var payload *trisa.Payload
	if payload, err = env.Payload(); err != nil {
		return nil, err
	}

	out.Identity = &ivms101.IdentityPayload{}
	if err = payload.Identity.UnmarshalTo(out.Identity); err != nil {
		return nil, err
	}

	switch payload.Transaction.TypeUrl {
	case "type.googleapis.com/trisa.data.generic.v1beta1.Transaction":
		out.Transaction = &generic.Transaction{}
		if err = payload.Transaction.UnmarshalTo(out.Transaction); err != nil {
			return nil, err
		}
	case "type.googleapis.com/trisa.data.generic.v1beta1.Pending":
		out.Pending = &generic.Pending{}
		if err = payload.Transaction.UnmarshalTo(out.Pending); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unknown transaction protobuf type: %q", payload.Transaction.TypeUrl)
	}

	if out.SentAt, err = parseTimestamp(payload.SentAt); err != nil {
		return nil, fmt.Errorf("could not parse sent at timestamp: %s", err)
	}

	if out.ReceivedAt, err = parseTimestamp(payload.ReceivedAt); err != nil {
		return nil, fmt.Errorf("could not parse received at timestamp: %s", err)
	}

	return out, nil
}

func (e *Envelope) Validate() (err error) {
	// Perform lightweight validation of the payload
	if e.Error != nil {
		if e.Identity != nil || e.Transaction != nil || e.Pending != nil {
			return ValidationError(OneOfTooMany("error", "identity"))
		}
		return nil
	}

	if e.Identity == nil {
		err = ValidationError(err, MissingField("identity"))
	}

	if e.Transaction == nil && e.Pending == nil {
		err = ValidationError(err, OneOfMissing("transaction", "pending"))
	}

	if e.Transaction != nil && e.Pending != nil {
		err = ValidationError(err, OneOfTooMany("transaction", "pending"))
	}

	return err
}

func (e *Envelope) Payload() (payload *trisa.Payload, err error) {
	payload = &trisa.Payload{}

	if payload.Identity, err = anypb.New(e.Identity); err != nil {
		return nil, err
	}

	var data protoreflect.ProtoMessage
	switch {
	case e.Transaction != nil:
		data = e.Transaction
	case e.Pending != nil:
		data = e.Pending
	default:
		return nil, OneOfMissing("transaction", "pending")
	}

	if payload.Transaction, err = anypb.New(data); err != nil {
		return nil, err
	}

	if e.SentAt != nil && !e.SentAt.IsZero() {
		payload.SentAt = e.SentAt.Format(time.RFC3339)
	} else {
		payload.SentAt = time.Now().UTC().Format(time.RFC3339)
	}

	if e.ReceivedAt != nil {
		payload.ReceivedAt = e.ReceivedAt.Format(time.RFC3339)
	}

	return payload, nil
}

//===========================================================================
// Transaction Query
//===========================================================================

func (q *TransactionQuery) Validate() (err error) {
	// Handle parsing and default values
	q.Detail = strings.ToLower(strings.TrimSpace(q.Detail))
	if q.Detail == "" {
		q.Detail = DetailFull
	}

	if q.Detail != DetailFull && q.Detail != DetailPreview {
		err = ValidationError(err, IncorrectField("detail", "should either be 'full' or 'preview'"))
	}
	return err
}

//===========================================================================
// Rejection
//===========================================================================

func (r *Rejection) Validate() (err error) {
	// Check that the error code is valid
	r.Code = strings.ToUpper(strings.TrimSpace(r.Code))
	if _, ok := trisa.Error_Code_value[r.Code]; !ok {
		err = ValidationError(err, IncorrectField("code", "not a valid TRISA error code as defined by the TRISA protocol buffers"))
	}

	// A rejection message is required from the user
	r.Message = strings.TrimSpace(r.Message)
	if r.Message == "" {
		err = ValidationError(err, MissingField("message"))
	}

	return err
}

func (r *Rejection) Proto() *trisa.Error {
	// Convert the Code into a TRISA error code; if it fails, use Unhandled. Ensure the
	// Rejection message is validated before calling this method to catch errors.
	return &trisa.Error{
		Code:    trisa.Error_Code(trisa.Error_Code_value[r.Code]),
		Message: r.Message,
		Retry:   r.RequestRetry,
	}
}

//===========================================================================
// Helper Utilities
//===========================================================================

func parseTimestamp(ts string) (_ *time.Time, err error) {
	ts = strings.TrimSpace(ts)
	if ts == "" {
		return nil, nil
	}

	layouts := []string{time.RFC3339, time.RFC3339Nano}
	for _, layout := range layouts {
		var t time.Time
		if t, err = time.Parse(layout, ts); err == nil {
			return &t, nil
		}
	}

	return nil, ErrInvalidTimestamp
}
