package scene

import "github.com/trisacrypto/envoy/pkg/web/api/v1"

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
