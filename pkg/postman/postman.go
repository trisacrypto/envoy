package postman

import (
	"database/sql"

	"github.com/rs/zerolog"
	"github.com/trisacrypto/envoy/pkg/store/models"
)

type ResolveRemote interface {
	Remote() sql.NullString
	ResolveCounterparty() error
}

// Packets contain both an incoming and an outgoing message and are used to ensure that
// an entire tranfer packet can be correctly constructed for both envelopes.
type Packet struct {
	DB           models.PreparedTransaction // Database interaction methods
	In           *Incoming                  // The incoming message that needs to be decrypted
	Out          *Outgoing                  // The outgoing message that needs to be encrypted
	Log          zerolog.Logger             // The log context for more effective logging
	Counterparty *models.Counterparty       // The remote identified counterparty
	Transaction  *models.Transaction        // The associated transaction with the packet
	Request      Direction                  // Determines if the initial message was incoming or outgoing
	Reply        Direction                  // Determines if the reply was incoming or outgoing
	resolver     ResolveRemote              // Helper for resolving remote information from TRISA and TRP
}

func (p *Packet) RefreshTransaction() (err error) {
	if p.Transaction, err = p.DB.Fetch(); err != nil {
		return err
	}
	return nil
}

// Returns the envelopeID from the request envelope (e.g. the first envelope in the packet)
func (p *Packet) EnvelopeID() string {
	switch p.Request {
	case DirectionIncoming:
		return p.In.Envelope.ID()
	case DirectionOutgoing:
		return p.Out.Envelope.ID()
	default:
		panic("request direction not set on packet")
	}
}

// If a resolver is set, then we can resolve the remote information from the Peer,
// otherwise this method is a no-op and returns no error.
func (p *Packet) ResolveCounterparty() (err error) {
	if p.resolver != nil {
		return p.ResolveCounterparty()
	}
	return nil
}

// Returns the remote information for storage in the database using the underlying
// resolver if available, otherwise a NULL string is returned.
func (p *Packet) Remote() sql.NullString {
	if p.resolver != nil {
		return p.resolver.Remote()
	}
	return sql.NullString{Valid: false, String: ""}
}
