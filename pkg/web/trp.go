package web

import (
	"context"

	"github.com/trisacrypto/envoy/pkg/postman"
)

// Send either a TRP Inquiry or a TRP Confirmation message depending on the state of the
// the transfer (e.g. either new inquiry or accept vs reject).
func (s *Server) SendTRPMessage(ctx context.Context, p *postman.Packet) (err error) {

	// Create the logger for sending the message
	p.Log = p.Log.With().Str("method", "trp").Str("envelope_id", p.EnvelopeID()).Logger()

	return nil
}

func (s *Server) SendTRPInquiry(ctx context.Context, p *postman.Packet) (err error) {
	p.Log.Debug().Msg("started outgoing TRP inquiry")
	return nil
}

func (s *Server) SendTRPConfirmation(ctx context.Context, p *postman.Packet) (err error) {
	p.Log.Debug().Msg("started outgoing TRP confirmation")
	return nil
}
