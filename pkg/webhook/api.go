package webhook

import (
	"fmt"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/trisacrypto/envoy/pkg/web/api/v1"

	"github.com/trisacrypto/trisa/pkg/ivms101"
	trisa "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
	generic "github.com/trisacrypto/trisa/pkg/trisa/data/generic/v1beta1"
)

// The Request object is sent to the webhook via a POST http call. The request
// represents an incoming message to the server as an unsealed, decrypted secure
// envelope (whether the request came from a TRISA or TRP remote client). The request
// is guaranteed to have a transaction ID, timestamp, and counterparty. If it has a
// payload, then it will also have an HMAC signature and public key signature. Requests
// will have either errors or payloads, but not both.
type Request struct {
	TransactionID uuid.UUID         `json:"transaction_id"`
	Timestamp     string            `json:"timestamp"`
	Counterparty  *api.Counterparty `json:"counterparty"`
	HMAC          string            `json:"hmac_signature,omitempty"`
	PKS           string            `json:"public_key_signature,omitempty"`
	TransferState string            `json:"transfer_state,omitempty"`
	Error         *trisa.Error      `json:"error,omitempty"`
	Payload       *Payload          `json:"payload,omitempty"`
}

// Payload is a denormalized representation of a TRISA payload that includes
// type-specific data structures. The payload should always have an identity IVMS101
// payload and a sent at timestamp. It will have either a pending message or a
// transaction but not both. If payload is in an envelope with an accepted or completed
// transfer state it will have a received at timestamp as well.
type Payload struct {
	Identity    *ivms101.IdentityPayload `json:"identity"`
	Pending     *generic.Pending         `json:"pending,omitempty"`
	Transaction *generic.Transaction     `json:"transaction,omitempty"`
	SentAt      string                   `json:"sent_at"`
	ReceivedAt  string                   `json:"received_at,omitempty"`
}

// Reply represents the expected response from the callback webhook to the Envoy node.
// Either an error or a pending message is returned in the common case, though Envoy
// will also handle synchronous compliance responses.
type Reply struct {
	TransactionID uuid.UUID    `json:"transaction_id"`
	Error         *trisa.Error `json:"error,omitempty"`
	Payload       *Payload     `json:"payload,omitempty"`
}

const (
	transactionPBType = "type.googleapis.com/trisa.data.generic.v1beta1.Transaction"
	pendingPBType     = "type.googleapis.com/trisa.data.generic.v1beta1.Pending"
)

// Add a TRISA protocol buffer payload to the webhook request, unmarshaling it into its
// denormalized JSON representation to conduct the request.
func (r *Request) AddPayload(payload *trisa.Payload) (err error) {
	r.Payload = &Payload{
		SentAt:     payload.SentAt,
		ReceivedAt: payload.ReceivedAt,
		Identity:   &ivms101.IdentityPayload{},
	}

	if err = payload.Identity.UnmarshalTo(r.Payload.Identity); err != nil {
		return fmt.Errorf("could not unmarshal identity payload: %s", err)
	}

	switch payload.Transaction.TypeUrl {
	case transactionPBType:
		r.Payload.Transaction = &generic.Transaction{}
		if err = payload.Transaction.UnmarshalTo(r.Payload.Transaction); err != nil {
			return fmt.Errorf("could not unmarshal transaction payload: %s", err)
		}
	case pendingPBType:
		r.Payload.Pending = &generic.Pending{}
		if err = payload.Transaction.UnmarshalTo(r.Payload.Pending); err != nil {
			return fmt.Errorf("could not unmarshal pending payload: %s", err)
		}
	default:
		return fmt.Errorf("unknown transaction type %q", payload.Transaction.TypeUrl)
	}

	return nil
}

// Convert payload (usually from a reply) into a TRISA protocol buffer struct.
func (p *Payload) Proto() (payload *trisa.Payload, err error) {
	if p.Identity == nil {
		return nil, ErrIdentityRequired
	}

	if p.Pending == nil && p.Transaction == nil {
		return nil, ErrTransactionRequired
	}

	payload = &trisa.Payload{
		SentAt:     p.SentAt,
		ReceivedAt: p.ReceivedAt,
	}

	// Marshal the identity into the payload
	if payload.Identity, err = anypb.New(p.Identity); err != nil {
		return nil, fmt.Errorf("could not marshal identity into any: %s", err)
	}

	// If both p.Transaction and p.Pending are not nil, then p.Pending should be set.
	if p.Transaction != nil {
		if payload.Transaction, err = anypb.New(p.Transaction); err != nil {
			return nil, fmt.Errorf("could not marshal transaction into any: %s", err)
		}
	}

	if p.Pending != nil {
		if payload.Transaction, err = anypb.New(p.Pending); err != nil {
			return nil, fmt.Errorf("could not marshal pending into any: %s", err)
		}
	}

	return payload, nil
}

func (p *Payload) IsZero() bool {
	return p.Identity == nil && p.Pending == nil && p.Transaction == nil && p.SentAt == "" && p.ReceivedAt == ""
}
