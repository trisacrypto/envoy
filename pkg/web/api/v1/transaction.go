package api

import (
	"database/sql"
	"strings"
	"time"

	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/envoy/pkg/ulids"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
	"github.com/trisacrypto/trisa/pkg/ivms101"
	trisa "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
	generic "github.com/trisacrypto/trisa/pkg/trisa/data/generic/v1beta1"
	"google.golang.org/protobuf/proto"
)

//===========================================================================
// Transaction Resource
//===========================================================================

type Transaction struct {
	ID                 uuid.UUID `json:"id"`
	Source             string    `json:"source"`
	Status             string    `json:"status"`
	Counterparty       string    `json:"counterparty"`
	CounterpartyID     ulid.ULID `json:"counterparty_id,omitempty"`
	Originator         string    `json:"originator,omitempty"`
	OriginatorAddress  string    `json:"orginator_address,omitempty"`
	Beneficiary        string    `json:"beneficiary,omitempty"`
	BeneficiaryAddress string    `json:"beneficiary_address,omitempty"`
	VirtualAsset       string    `json:"virtual_asset"`
	Amount             float64   `json:"amount"`
	LastUpdate         time.Time `json:"last_update,omitempty"`
	EnvelopeCount      int64     `json:"envelope_count,omitempty"`
	Created            time.Time `json:"created"`
	Modified           time.Time `json:"modified"`
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

type DecryptedEnvelope struct {
	Identity    *ivms101.IdentityPayload `json:"identity,omitempty"`
	Transaction *generic.Transaction     `json:"transaction,omitempty"`
	Pending     *generic.Pending         `json:"pending,omitempty"`
	SentAt      time.Time                `json:"sent_at"`
	ReceivedAt  time.Time                `json:"received_at,omitempty"`
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
	Page               *PageQuery           `json:"page"`
	IsDecrypted        bool                 `json:"is_decrypted"`
	SecureEnvelopes    []*SecureEnvelope    `json:"secure_envelopes,omitempty"`
	DecryptedEnvelopes []*DecryptedEnvelope `json:"decrypted_envelopes,omitempty"`
}

func NewTransaction(model *models.Transaction) (*Transaction, error) {
	return &Transaction{
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
		LastUpdate:         model.LastUpdate.Time,
		EnvelopeCount:      model.NumEnvelopes(),
		Created:            model.Created,
		Modified:           model.Modified,
	}, nil
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
		LastUpdate:         sql.NullTime{Time: c.LastUpdate, Valid: !c.LastUpdate.IsZero()},
	}

	return model, nil
}

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
