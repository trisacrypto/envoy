package scene

import (
	"github.com/trisacrypto/envoy/pkg/web/api/v1"
	generic "github.com/trisacrypto/trisa/pkg/trisa/data/generic/v1beta1"
)

// Wraps an *api.Envelope to provide additional UI-specific functionality.
type Envelope struct {
	api.Envelope
}

//===========================================================================
// Scene Envelope Helpers
//===========================================================================

func (s Scene) Envelope() *Envelope {
	if data, ok := s[APIData]; ok {
		if env, ok := data.(*api.Envelope); ok {
			return &Envelope{
				Envelope: *env,
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
