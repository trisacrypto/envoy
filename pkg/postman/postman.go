/*
Postman providers helper functionality for managing TRISA and TRP transfers and
sorting them and storing them in the database. This package is intended to unify the
functionality across the TRISA node, the TRP node, and the Web API/UI.

On every single travel rule transaction, no matter if it's TRISA or TRP, no matter
if it's sent from the node or received into the node, whether or not it's a new
transaction or an update to an old transaction the following things must happen:

1. The message(s) must be validated
2. The transfer packet must be associated with a transaction
3. The transaction status must be updated, and potentially other parts of the transaction
4. The counterparty must be identified
5. Error envelopes have to be handled correctly
6. The keys for the envelope must be loaded for decryption
7. Sealed envelopes need to be decrypted
8. HMAC signatures need to be checked
9. The outgoing envelope must be resealed with internal keys
10. The envelopes and all changes must be saved to the database
11. The audit log must be updated
*/
package postman

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/trisacrypto/envoy/pkg/enum"
	"github.com/trisacrypto/envoy/pkg/store/models"
	trisa "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
	"github.com/trisacrypto/trisa/pkg/trisa/envelope"
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
	request      enum.Direction             // Determines if the initial message was incoming or outgoing
	reply        enum.Direction             // Determines if the reply was incoming or outgoing
	resolver     ResolveRemote              // Helper for resolving remote information from TRISA and TRP
}

func Send(envelopeID uuid.UUID, payload *trisa.Payload, transferState trisa.TransferState) (packet *Packet, err error) {
	packet = &Packet{
		In:      &Incoming{},
		Out:     &Outgoing{},
		request: enum.DirectionOutgoing,
		reply:   enum.DirectionIncoming,
	}

	// Add parent to submessages
	packet.In.packet = packet
	packet.Out.packet = packet

	// Create outgoing envelope
	env := []envelope.Option{
		envelope.WithEnvelopeID(envelopeID.String()),
		envelope.WithTransferState(transferState),
	}

	if packet.Out.Envelope, err = envelope.New(payload, env...); err != nil {
		return nil, fmt.Errorf("could not create envelope for payload: %w", err)
	}

	return packet, nil
}

func SendReject(envelopeID uuid.UUID, reject *trisa.Error) (packet *Packet, err error) {
	packet = &Packet{
		In:      &Incoming{},
		Out:     &Outgoing{},
		request: enum.DirectionOutgoing,
		reply:   enum.DirectionIncoming,
	}

	// Add parent to submessages
	packet.In.packet = packet
	packet.Out.packet = packet

	// The envelope package should set the correct transfer state based on retry.
	if packet.Out.Envelope, err = envelope.WrapError(reject, envelope.WithEnvelopeID(envelopeID.String())); err != nil {
		return nil, fmt.Errorf("could not create rejection envelope: %w", err)
	}

	return packet, nil
}

func Receive(in *trisa.SecureEnvelope) (packet *Packet, err error) {
	packet = &Packet{
		In:      &Incoming{original: in},
		Out:     &Outgoing{},
		request: enum.DirectionIncoming,
		reply:   enum.DirectionOutgoing,
	}

	// Add parent to submessages
	packet.In.packet = packet
	packet.Out.packet = packet

	if packet.In.Envelope, err = envelope.Wrap(packet.In.original); err != nil {
		return nil, fmt.Errorf("could not wrap incoming secure envelope: %w", err)
	}

	return packet, nil
}

func (p *Packet) TRISA() *TRISAPacket {
	packet := &TRISAPacket{
		Packet: *p,
	}

	// Add parent to submessages
	packet.In.packet = &packet.Packet
	packet.Out.packet = &packet.Packet

	packet.Packet.resolver = packet
	return packet
}

func (p *Packet) Sunrise() *SunrisePacket {
	packet := &SunrisePacket{
		Packet: *p,
	}

	// Add parent to submessages
	packet.In.packet = &packet.Packet
	packet.Out.packet = &packet.Packet

	// Keep track of the original payload
	payload, _ := packet.Out.Envelope.Payload()
	packet.payload = payload

	return packet
}

func (p *Packet) TRP() *TRPPacket {
	return nil
}

func (p *Packet) RefreshTransaction() (err error) {
	if p.Transaction, err = p.DB.Fetch(); err != nil {
		return err
	}
	return nil
}

// Returns the envelopeID from the request envelope (e.g. the first envelope in the packet)
func (p *Packet) EnvelopeID() string {
	switch p.request {
	case enum.DirectionIncoming:
		return p.In.Envelope.ID()
	case enum.DirectionOutgoing:
		return p.Out.Envelope.ID()
	default:
		panic("request direction not set on packet")
	}
}

// If a resolver is set, then we can resolve the remote information from the Peer,
// otherwise this method is a no-op and returns no error.
func (p *Packet) ResolveCounterparty() (err error) {
	if p.resolver != nil {
		return p.resolver.ResolveCounterparty()
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

// Returns the direction of the request.
func (p *Packet) Request() enum.Direction {
	return p.request
}

// Returns the direction of the reply.
func (p *Packet) Reply() enum.Direction {
	return p.reply
}
