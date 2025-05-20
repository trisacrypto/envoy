package scene

import (
	"encoding/json"
	"time"

	"github.com/trisacrypto/envoy/pkg/web/api/v1"
	generic "github.com/trisacrypto/trisa/pkg/trisa/data/generic/v1beta1"
)

// Wraps an *api.Envelope to provide additional UI-specific functionality.
type Envelope struct {
	api.Envelope
}

// Wraps a *trisa.Error to provide additional UI-specific functionality.
type Rejection struct {
	api.Rejection
}

// Wraps a *generic.Transaction to provide additional UI-specific functionality.
type TransactionPayload struct {
	generic.Transaction
}

//===========================================================================
// Scene Envelope Helpers
//===========================================================================

func (s Scene) Rejection() *Rejection {
	if data, ok := s[APIData]; ok {
		if env, ok := data.(*api.Repair); ok {
			return &Rejection{
				Rejection: *env.Error,
			}
		}
	}
	return nil
}

func (s Scene) Envelope() *Envelope {
	if data, ok := s[APIData]; ok {
		if env, ok := data.(*api.Envelope); ok {
			return &Envelope{
				Envelope: *env,
			}
		}

		if env, ok := data.(*api.Repair); ok {
			return &Envelope{
				Envelope: *env.Envelope,
			}
		}
	}
	return nil
}

func (s Scene) TransactionPayload() *TransactionPayload {
	if data, ok := s[APIData]; ok {
		if payload, ok := data.(*generic.Transaction); ok {
			return &TransactionPayload{
				Transaction: generic.Transaction{
					Txid:        payload.Txid,
					Originator:  payload.Originator,
					Beneficiary: payload.Beneficiary,
					Amount:      payload.Amount,
					Network:     payload.Network,
					Timestamp:   payload.Timestamp,
					ExtraJson:   payload.ExtraJson,
					AssetType:   payload.AssetType,
					Tag:         payload.Tag,
				},
			}
		}
	}
	return nil
}

//===========================================================================
// Envelope Methods
//===========================================================================

func (e *Envelope) Identity() *IVMS101 {
	return NewIVMS101(e)
}

func (e *Envelope) Transaction() *generic.Transaction {
	if tx := e.TransactionPayload(); tx != nil {
		return tx
	}
	return &generic.Transaction{}
}

func (e *Envelope) TransactionJSON() string {
	if tx := e.TransactionPayload(); tx != nil {
		data, _ := json.Marshal(tx)
		return string(data)
	}
	return ""
}

func (e *Envelope) SentAtRepr() string {
	if e.SentAt != nil {
		return e.SentAt.Format(time.RFC3339)
	}
	return ""
}

func (e *Envelope) ReceivedAtRepr() string {
	if e.ReceivedAt != nil {
		return e.ReceivedAt.Format(time.RFC3339)
	}
	return ""
}
