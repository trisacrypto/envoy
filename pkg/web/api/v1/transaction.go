package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/trisacrypto/envoy/pkg/store/models"

	"github.com/trisacrypto/trisa/pkg/iso3166"
	"github.com/trisacrypto/trisa/pkg/ivms101"
	trisa "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
	generic "github.com/trisacrypto/trisa/pkg/trisa/data/generic/v1beta1"
	"github.com/trisacrypto/trisa/pkg/trisa/envelope"

	"github.com/google/uuid"
	"go.rtnl.ai/ulid"
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
	OriginatorAddress  string     `json:"originator_address,omitempty"`
	Beneficiary        string     `json:"beneficiary,omitempty"`
	BeneficiaryAddress string     `json:"beneficiary_address,omitempty"`
	VirtualAsset       string     `json:"virtual_asset"`
	Amount             float64    `json:"amount"`
	Archived           bool       `json:"archived,omitempty"`
	ArchivedOn         *time.Time `json:"archived_on,omitempty"`
	LastUpdate         *time.Time `json:"last_update,omitempty"`
	EnvelopeCount      int64      `json:"envelope_count,omitempty"`
	Created            time.Time  `json:"created"`
	Modified           time.Time  `json:"modified"`
}

type TransactionQuery struct {
	Detail string `json:"detail" url:"detail,omitempty" form:"detail"`
}

type SecureEnvelope struct {
	ID                  ulid.ULID    `json:"id"`
	EnvelopeID          uuid.UUID    `json:"envelope_id"`
	Direction           string       `json:"direction"`
	Remote              string       `json:"remote,omitempty"`
	ReplyTo             *ulid.ULID   `json:"reply_to"`
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
	TransferState       string       `json:"transfer_state,omitempty"`
	Original            []byte       `json:"original,omitempty"`
}

type Envelope struct {
	ID                 ulid.ULID                `json:"id"`
	EnvelopeID         string                   `json:"envelope_id,omitempty"`
	Direction          string                   `json:"direction"`
	Remote             string                   `json:"remote,omitempty"`
	ReplyTo            *ulid.ULID               `json:"reply_to"`
	IsError            bool                     `json:"is_error"`
	Error              *trisa.Error             `json:"error,omitempty"`
	Identity           *ivms101.IdentityPayload `json:"identity,omitempty"`
	Transaction        *generic.Transaction     `json:"transaction,omitempty"`
	Pending            *generic.Pending         `json:"pending,omitempty"`
	Sunrise            *generic.Sunrise         `json:"sunrise,omitempty"`
	SentAt             *time.Time               `json:"sent_at"`
	ReceivedAt         *time.Time               `json:"received_at,omitempty"`
	Timestamp          time.Time                `json:"timestamp,omitempty"`
	PublicKeySignature string                   `json:"public_key_signature,omitempty"`
	TransferState      string                   `json:"transfer_state,omitempty"`
	SecureEnvelope     *SecureEnvelope          `json:"secure_envelope,omitempty"`
}

type Rejection struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Retry   bool   `json:"retry"`
}

type Repair struct {
	Error    *Rejection
	Envelope *Envelope
}

type EnvelopeQuery struct {
	Decrypt   bool   `json:"decrypt" url:"decrypt,omitempty" form:"decrypt"`
	Archives  bool   `json:"archives" url:"archives,omitempty" form:"archives"`
	Direction string `json:"direction,omitempty" url:"direction,omitempty" form:"direction"`
}

type TransactionsList struct {
	Page         *TransactionListQuery `json:"page"`
	Transactions []*Transaction        `json:"transactions"`
}

type TransactionListQuery struct {
	PageQuery
	Status       []string `json:"status,omitempty" url:"status,omitempty" form:"status"`
	VirtualAsset []string `json:"asset,omitempty" url:"asset,omitempty" form:"asset"`
	Archives     bool     `json:"archives,omitempty" url:"archives,omitempty" form:"archives"`
}

type EnvelopesList struct {
	Page               *PageQuery        `json:"page"`
	IsDecrypted        bool              `json:"is_decrypted"`
	SecureEnvelopes    []*SecureEnvelope `json:"secure_envelopes,omitempty"`
	DecryptedEnvelopes []*Envelope       `json:"decrypted_envelopes,omitempty"`
}

type EnvelopeListQuery struct {
	PageQuery
	EnvelopeQuery
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
		Archived:           model.Archived,
		EnvelopeCount:      model.NumEnvelopes(),
		Created:            model.Created,
		Modified:           model.Modified,
	}

	// If archivedOn is not NULL in the database, then add it to the response.
	if model.ArchivedOn.Valid {
		tx.ArchivedOn = &model.ArchivedOn.Time
	}

	// If last update is not NULL in the database, then add it to the response.
	if model.LastUpdate.Valid {
		tx.LastUpdate = &model.LastUpdate.Time
	}
	return tx, nil
}

func NewTransactionList(page *models.TransactionPage) (out *TransactionsList, err error) {
	out = &TransactionsList{
		Page: &TransactionListQuery{
			PageQuery: PageQuery{
				PageSize: int(page.Page.PageSize),
			},
			Status:       page.Page.Status,
			VirtualAsset: page.Page.VirtualAsset,
			Archives:     page.Page.Archives,
		},
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
		CounterpartyID:     ulid.NullULID{ULID: c.CounterpartyID, Valid: !c.CounterpartyID.IsZero()},
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
		EnvelopeID:          model.EnvelopeID,
		Direction:           model.Direction,
		Remote:              model.Remote.String,
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
		TransferState:       model.Envelope.TransferState.String(),
	}

	if model.ReplyTo.Valid {
		out.ReplyTo = &model.ReplyTo.ULID
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
		var env *SecureEnvelope
		if env, err = NewSecureEnvelope(model); err != nil {
			return nil, err
		}

		// Reduce the amount of information being sent in a list request
		// These fields can be obtained using a detail request
		env.Payload = nil
		env.EncryptionKey = nil
		env.HMACSecret = nil
		env.HMAC = nil
		env.Original = nil

		out.SecureEnvelopes = append(out.SecureEnvelopes, env)
	}

	return out, nil
}

//===========================================================================
// Envelopes
//===========================================================================

func NewEnvelope(model *models.SecureEnvelope, env *envelope.Envelope) (out *Envelope, err error) {
	out = &Envelope{
		ID:                 model.ID,
		EnvelopeID:         model.EnvelopeID.String(),
		Direction:          model.Direction,
		Remote:             model.Remote.String,
		Timestamp:          model.Timestamp,
		PublicKeySignature: model.PublicKey.String,
	}

	if model.ReplyTo.Valid {
		out.ReplyTo = &model.ReplyTo.ULID
	}

	// Add the secure envelope to the envelope detail to include metadata
	if out.SecureEnvelope, err = NewSecureEnvelope(model); err != nil {
		return nil, err
	}

	// If the envelope is nil, it's likely because the envelope could not be decrypted.
	if env == nil {
		return out, nil
	}

	// Use the decrypted envelope to populate the payload.
	out.TransferState = env.TransferState().String()
	switch state := env.State(); state {
	case envelope.Error:
		out.IsError = true
		out.Error = env.Error()
		return out, nil
	case envelope.Clear:
		break
	default:
		return nil, fmt.Errorf("envelope is in an unhandled state: %s", state)
	}

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
	case "type.googleapis.com/trisa.data.generic.v1beta1.Sunrise":
		out.Sunrise = &generic.Sunrise{}
		if err = payload.Transaction.UnmarshalTo(out.Sunrise); err != nil {
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

func NewEnvelopeList(page *models.SecureEnvelopePage, envelopes []*envelope.Envelope) (out *EnvelopesList, err error) {
	if len(page.Envelopes) != len(envelopes) {
		return nil, fmt.Errorf("page of %d secure envelopes does not match %d decrypted envelopes", len(page.Envelopes), len(envelopes))
	}

	out = &EnvelopesList{
		Page:               &PageQuery{},
		IsDecrypted:        true,
		DecryptedEnvelopes: make([]*Envelope, 0, len(page.Envelopes)),
	}

	for i, model := range page.Envelopes {
		var env *Envelope
		if env, err = NewEnvelope(model, envelopes[i]); err != nil {
			return nil, err
		}

		// Reduce the amount of information being sent in a list request
		// These fields can be obtained using a detail request
		env.Identity = nil
		env.Transaction = nil
		env.Pending = nil
		env.Sunrise = nil
		env.SecureEnvelope = nil

		out.DecryptedEnvelopes = append(out.DecryptedEnvelopes, env)
	}

	return out, nil
}

func (e *Envelope) Dump() string {
	data, err := json.Marshal(e)
	if err != nil {
		log.Warn().Err(err).Msg("could not marshal envelope data")
		return ""
	}
	return string(data)
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
	case e.Sunrise != nil:
		data = e.Sunrise
	default:
		return nil, OneOfMissing("transaction", "pending", "sunrise")
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

func (e *Envelope) ParseTransferState() trisa.TransferState {
	state, _ := trisa.ParseTransferState(e.TransferState)
	return state
}

// Retrieves the Originator VASP LegalPerson record
func (e *Envelope) OriginatorVASP() *ivms101.LegalPerson {
	if e.Identity != nil {
		if e.Identity.OriginatingVasp != nil {
			if e.Identity.OriginatingVasp.OriginatingVasp != nil {
				if person := e.Identity.OriginatingVasp.OriginatingVasp.GetLegalPerson(); person != nil {
					return person
				}
			}
		}
	}

	log.Debug().Msg("could not identify originator VASP in identity payload")
	return nil
}

// Retrieves the Beneficiary VASP LegalPerson record
func (e *Envelope) BeneficiaryVASP() *ivms101.LegalPerson {
	if e.Identity != nil {
		if e.Identity.BeneficiaryVasp != nil {
			if e.Identity.BeneficiaryVasp.BeneficiaryVasp != nil {
				if person := e.Identity.BeneficiaryVasp.BeneficiaryVasp.GetLegalPerson(); person != nil {
					return person
				}
			}
		}
	}

	log.Debug().Msg("could not identify beneficiary VASP in identity payload")
	return nil
}

// Retrieves first originator account in the identity payload that has a legal name.
func (e *Envelope) FirstOriginator() *ivms101.NaturalPerson {
	if e.Identity != nil {
		if e.Identity.Originator != nil {
			if len(e.Identity.Originator.OriginatorPersons) > 0 {
				// Search for the first natural person to have a legal name.
				for _, originator := range e.Identity.Originator.OriginatorPersons {
					if person := originator.GetNaturalPerson(); person != nil {
						if nameIdx := FindLegalName(person); nameIdx >= 0 {
							return person
						}
					}
				}

				// If no legal person with a legal name is found, return first originator
				if person := e.Identity.Originator.OriginatorPersons[0].GetNaturalPerson(); person != nil {
					return person
				}
			}
		}
	}

	log.Debug().Msg("could not identify any originator(s) in identity payload")
	return nil
}

// Retrieves first beneficiary account in the identity payload that has a legal name.
func (e *Envelope) FirstBeneficiary() *ivms101.NaturalPerson {
	if e.Identity != nil {
		if e.Identity.Beneficiary != nil {
			if len(e.Identity.Beneficiary.BeneficiaryPersons) > 0 {
				// Search for the first natural person to have a legal name.
				for _, beneficiary := range e.Identity.Beneficiary.BeneficiaryPersons {
					if person := beneficiary.GetNaturalPerson(); person != nil {
						if nameIdx := FindLegalName(person); nameIdx >= 0 {
							return person
						}
					}
				}

				// If no legal person with a legal name is found, return first originator
				if person := e.Identity.Beneficiary.BeneficiaryPersons[0].GetNaturalPerson(); person != nil {
					return person
				}
			}
		}
	}

	log.Debug().Msg("could not identify a beneficiary in identity payload")
	return nil
}

// Searches for the the transaction payload in the envelope, unwrapping transactions in
// pending or sunrise messages as a quick helper for transaction details. If no
// transaction is available or an error occurs, then nil is returned.
func (e *Envelope) TransactionPayload() *generic.Transaction {
	switch {
	case e.Transaction != nil:
		return e.Transaction
	case e.Pending != nil && e.Pending.Transaction != nil:
		return e.Pending.Transaction
	case e.Sunrise != nil && e.Sunrise.Transaction != nil:
		return e.Sunrise.Transaction
	default:
		log.Debug().Msg("could not identify transaction payload")
		return nil
	}
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

func (q *TransactionListQuery) Validate() (err error) {
	if len(q.Status) > 0 {
		for i, status := range q.Status {
			q.Status[i] = strings.ToLower(strings.TrimSpace(status))
			if !models.ValidStatus(q.Status[i]) {
				err = ValidationError(err, IncorrectField("status", "invalid status enum"))
				break
			}
		}
	}

	if len(q.VirtualAsset) > 0 {
		for i, asset := range q.VirtualAsset {
			q.VirtualAsset[i] = strings.ToUpper(strings.TrimSpace(asset))
		}
	}

	return err
}

func (q *TransactionListQuery) Query() (query *models.TransactionPageInfo) {
	query = &models.TransactionPageInfo{
		PageInfo: models.PageInfo{
			PageSize: uint32(q.PageSize),
		},
		Status:       q.Status,
		VirtualAsset: q.VirtualAsset,
		Archives:     q.Archives,
	}
	return query
}

//===========================================================================
// Envelope Query
//===========================================================================

func (q *EnvelopeQuery) Validate() (err error) {
	// Handle parsing and default values
	q.Direction = strings.ToLower(strings.TrimSpace(q.Direction))
	if q.Direction == "" {
		q.Direction = models.DirectionAny
	}

	if q.Direction != models.DirectionAny && q.Direction != models.DirectionIn && q.Direction != models.DirectionOut {
		err = ValidationError(err, IncorrectField("direction", "should either be 'in', 'out', or 'any'"))
	}

	return err
}

//===========================================================================
// Rejection
//===========================================================================

func NewRejection(env *models.SecureEnvelope) (out *Rejection, err error) {
	if !env.IsError {
		return nil, ErrInvalidRejection
	}

	out = &Rejection{
		Code:    env.Envelope.Error.Code.String(),
		Message: env.Envelope.Error.Message,
		Retry:   env.Envelope.Error.Retry,
	}

	return out, nil
}

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
		Retry:   r.Retry,
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

func filterSpaces(arr []string) []string {
	i := 0
	for _, s := range arr {
		if strings.TrimSpace(s) != "" {
			arr[i] = s
			i++
		}
	}
	return arr[:i]
}

// Find the index in the name identifiers of the legal name of either a legal or natural person.
func FindLegalName(person interface{}) int {
	switch p := person.(type) {
	case *ivms101.Person:
		if np := p.GetNaturalPerson(); np != nil {
			return FindLegalName(np)
		}

		if lp := p.GetLegalPerson(); lp != nil {
			return FindLegalName(lp)
		}

		log.Debug().Str("person", p.String()).Msg("unhandled person identifier type")
		return -1
	case *ivms101.LegalPerson:
		if p.Name != nil {
			for i, name := range p.Name.NameIdentifiers {
				if name.LegalPersonNameIdentifierType == ivms101.LegalPersonLegal {
					return i
				}
			}
		}

		log.Debug().Msg("could not find legal name on legal person")
		return -1
	case *ivms101.NaturalPerson:
		if p.Name != nil {
			for i, name := range p.Name.NameIdentifiers {
				if name.NameIdentifierType == ivms101.NaturalPersonLegal {
					return i
				}
			}
		}

		log.Debug().Msg("could not find legal name on natural person")
		return -1
	default:
		log.Debug().Type("person", person).Msg("unhandled type to find person name")
		return -1
	}
}

// Find primary geographic address of a person in the IVMS101 dataset; the address is
// returned as a series of address lines to simplify the representation.
func FindPrimaryAddress(person interface{}) *ivms101.Address {
	switch p := person.(type) {
	case *ivms101.Person:
		if np := p.GetNaturalPerson(); np != nil {
			return FindPrimaryAddress(np)
		}

		if lp := p.GetLegalPerson(); lp != nil {
			return FindPrimaryAddress(lp)
		}

		log.Debug().Str("person", p.String()).Msg("unhandled person identifier type")
		return nil

	case *ivms101.LegalPerson:
		if len(p.GeographicAddresses) > 0 {
			for _, addr := range p.GeographicAddresses {
				if addr.AddressType == ivms101.AddressTypeBusiness {
					return addr
				}
			}

			// Otherwise just return the first address in the list
			return p.GeographicAddresses[0]
		}
		return nil

	case *ivms101.NaturalPerson:
		if len(p.GeographicAddresses) > 0 {
			for _, addr := range p.GeographicAddresses {
				if addr.AddressType == ivms101.AddressTypeHome {
					return addr
				}
			}

			// Otherwise just return the first address in the list
			return p.GeographicAddresses[0]
		}
		return nil
	default:
		log.Debug().Type("person", person).Msg("unhandled type to find person primary address")
		return nil
	}
}

func MakeAddressLines(addr *ivms101.Address) (address []string) {
	if addr == nil {
		return nil
	}

	// Handle the simple case where there are address lines.
	if len(addr.AddressLine) > 0 {
		address = make([]string, 0, len(addr.AddressLine)+2)
		address = append(address, AddressTypeRepr(addr.AddressType))
		address = append(address, addr.AddressLine...)
		address = append(address, CountryName(addr.Country))
		return filterSpaces(address)
	}

	// Otherwise, construct the address from the individual components.
	// TODO: ensure all components are included and correctly formatted for the country
	address = make([]string, 0, 8)
	address = append(address, AddressTypeRepr(addr.AddressType))
	address = append(address, AddrLineRepr(fmt.Sprintf("%s %s %s", addr.BuildingNumber, addr.BuildingName, addr.StreetName)))
	address = append(address, AddrLineRepr(addr.PostBox))
	address = append(address, AddrLineRepr(fmt.Sprintf("%s %s", addr.Department, addr.SubDepartment)))
	address = append(address, AddrLineRepr(fmt.Sprintf("%s %s", addr.Floor, addr.Room)))
	address = append(address, AddrLineRepr(fmt.Sprintf("%s %s %s %s", addr.TownLocationName, addr.TownName, addr.DistrictName, addr.CountrySubDivision)))
	address = append(address, AddrLineRepr(addr.PostCode))
	address = append(address, CountryName(addr.Country))

	return filterSpaces(address)
}

func AddressTypeRepr(t ivms101.AddressTypeCode) string {
	switch t {
	case ivms101.AddressTypeGeographic:
		return "Geographic Address"
	case ivms101.AddressTypeBusiness:
		return "Business Address"
	case ivms101.AddressTypeHome:
		return "Home Address"
	case ivms101.AddressTypeMisc:
		return "Other Address"
	default:
		return "Address"
	}
}

var dupspace = regexp.MustCompile(`\s+`)

func AddrLineRepr(line string) string {
	line = dupspace.ReplaceAllString(line, " ")
	return strings.TrimSpace(line)
}

func CountryName(country string) string {
	if code, err := iso3166.Find(country); err == nil {
		return code.Country
	}
	return country
}
