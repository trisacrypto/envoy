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
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/envoy/pkg/trisa/peers"

	api "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
	"github.com/trisacrypto/trisa/pkg/trisa/envelope"
)

type Direction uint8

const (
	Unknown Direction = iota
	DirectionIncoming
	DirectionOutgoing
)

// Packets contain both an incoming and an outgoing message and are used to ensure that
// an entire tranfer packet can be correctly constructed for both envelopes.
type Packet struct {
	DB            models.PreparedTransaction // Database interaction methods
	In            *Incoming                  // The incoming message that needs to be decrypted
	Out           *Outgoing                  // The outgoing message that needs to be encrypted
	Log           zerolog.Logger             // The log context for more effective logging
	TravelAddress string                     // The original travel address (if TRP) to send the packet to
	Counterparty  *models.Counterparty       // The remote identified counterparty
	Transaction   *models.Transaction        // The associated transaction with the packet
	Peer          peers.Peer                 // The remote peer the transfer is being conducted with (if TRISA)
	PeerInfo      *peers.Info                // The peer info for finding the counterparty (if TRISA)
	User          *models.User               // The user that created the request, if any
	APIKey        *models.APIKey             // The api key that created the request, if any
	Request       Direction                  // Determines if the initial message was incoming or outgoing
	Reply         Direction                  // Determines if the reply was incoming or outgoing
	protocol      string                     // The protocol the packet is being sent with
}

func Send(payload *api.Payload, envelopeID uuid.UUID, transferState api.TransferState, log zerolog.Logger) (packet *Packet, err error) {
	packet = &Packet{
		In:      &Incoming{},
		Out:     &Outgoing{},
		Log:     log,
		Request: DirectionOutgoing,
		Reply:   DirectionIncoming,
	}

	// Add parent to submessages
	packet.In.packet = packet
	packet.Out.packet = packet

	opts := []envelope.Option{
		envelope.WithEnvelopeID(envelopeID.String()),
		envelope.WithTransferState(transferState),
	}

	if packet.Out.Envelope, err = envelope.New(payload, opts...); err != nil {
		return nil, fmt.Errorf("could not create envelope for payload: %w", err)
	}

	return packet, nil
}

func SendReject(reject *api.Error, envelopeID uuid.UUID, log zerolog.Logger) (packet *Packet, err error) {
	packet = &Packet{
		In:      &Incoming{},
		Out:     &Outgoing{},
		Log:     log,
		Request: DirectionOutgoing,
		Reply:   DirectionIncoming,
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

func Receive(in *api.SecureEnvelope, log zerolog.Logger, peer peers.Peer) (packet *Packet, err error) {
	packet = &Packet{
		In:      &Incoming{original: in},
		Out:     &Outgoing{},
		Log:     log,
		Peer:    peer,
		Request: DirectionIncoming,
		Reply:   DirectionOutgoing,
	}

	// Add parent to submessages
	packet.In.packet = packet
	packet.Out.packet = packet

	if packet.In.Envelope, err = envelope.Wrap(packet.In.original); err != nil {
		return nil, fmt.Errorf("could not wrap incoming secure envelope: %w", err)
	}

	if packet.PeerInfo, err = peer.Info(); err != nil {
		return nil, fmt.Errorf("could not identify counterparty in transaction: %w", err)
	}

	packet.Counterparty = packet.PeerInfo.Model()
	return packet, nil
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

// Returns the transfer state from the request envelope (e.g. the first envelope in the
// packet) - so if it is an outgoing message, it returns the transfer state from the
// outgoing payload and if it is incoming from the incoming payload.
func (p *Packet) TransferState() api.TransferState {
	switch p.Request {
	case DirectionIncoming:
		return p.In.Envelope.TransferState()
	case DirectionOutgoing:
		return p.Out.Envelope.TransferState()
	default:
		panic("request direction not set on packet")
	}
}

// Gets the protocol that the packet is being sent with; if no protocol has been set
// then the protocol of the counterparty is used.
func (p *Packet) Protocol() string {
	if p.protocol == "" {
		if p.Counterparty != nil {
			p.protocol = p.Counterparty.Protocol
		}
	}
	return p.protocol
}

// Receive updates the incoming message with the specified secure envelope, e.g. in the
// case where the outgoing message has been sent and this is the reply that was received
// from the remote server.
func (p *Packet) Receive(in *api.SecureEnvelope) (err error) {
	p.In = &Incoming{
		original: in,
		packet:   p,
	}

	if p.In.Envelope, err = envelope.Wrap(in); err != nil {
		return fmt.Errorf("could not wrap incoming secure envelope: %w", err)
	}

	return nil
}

// Reject creates an outgoing message with a TRISA error that contains the specified
// TRISA rejection error, e.g. in the case where an incoming message is being handled.
func (p *Packet) Reject(code api.Error_Code, message string, retry bool) error {
	reject := &api.Error{
		Code: code, Message: message, Retry: retry,
	}
	return p.Error(reject)
}

// Creates a TRISA error envelope from the TRISA error message and sets it as the
// outgoing message (e.g. in the case where an incoming message is being handled).
func (p *Packet) Error(reject *api.Error, opts ...envelope.Option) (err error) {
	if p.Out.Envelope, err = p.In.Envelope.Reject(reject, opts...); err != nil {
		p.Log.Debug().Err(err).Msg("could not prepare rejection envelope")
		return fmt.Errorf("could not create rejection envelope: %w", err)
	}
	return nil
}

// Creates a TRISA payload envelope to as the outgoing message in order to send a
// response back to the remote that initiated the transfer.
func (p *Packet) Send(payload *api.Payload, state api.TransferState) (err error) {
	if p.Out.Envelope, err = p.In.Envelope.Update(payload, envelope.WithTransferState(state)); err != nil {
		p.Log.Debug().Err(err).Msg("could not prepare outgoing payload")
		return fmt.Errorf("could not create outgoing payload: %w", err)
	}
	return nil
}

func (p *Packet) ResolveCounterparty() (err error) {
	if p.Counterparty == nil {
		if p.PeerInfo == nil {
			if p.Peer == nil {
				return ErrNoCounterpartyInfo
			}

			if p.PeerInfo, err = p.Peer.Info(); err != nil {
				return err
			}
		}

		p.Counterparty = p.PeerInfo.Model()
	}
	return nil
}

func (p *Packet) RefreshTransaction() (err error) {
	if p.Transaction, err = p.DB.Fetch(); err != nil {
		return err
	}
	return nil
}
