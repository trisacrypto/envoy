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
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/trisacrypto/envoy/pkg/store/models"
	api "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
	"github.com/trisacrypto/trisa/pkg/trisa/envelope"
)

// Packet is the base struct for all messaging interactions including TRISA, TRP and
// Sunrise messages. It handles all of the common functionality such as validation,
// envelope storage, transaction updates, and audit logging. Generally a user will not
// use this struct directly but will use the protocol structs which embed this struct.
// TODO: add audit log information
type Packet struct {
	log          zerolog.Logger             // the log context so that postman actions can be debugged
	db           models.PreparedTransaction // database context for storing and updating transactions
	in           *Incoming                  // the incoming message received from the remote counteparty
	out          *Outgoing                  // the outgoing message sent to the remote counterparty
	txn          *models.Transaction        // the transaction db record that this packet is associated with
	counterparty *models.Counterparty       // the counterparty db record associated with the transaction
	request      Direction                  // the direction of the request (e.g. the initial message)
	reply        Direction                  // direction of the reply (e.g. the response to the request)
}

//===========================================================================
// Packet Constructors
//===========================================================================

// Send is the basic mechanism to create a new packet whose request is outgoing.
func Send(payload *api.Payload, envelopeID uuid.UUID, transferState api.TransferState, opts ...Option) (packet *Packet, err error) {
	packet = &Packet{
		in:      &Incoming{},
		out:     &Outgoing{},
		request: DirectionOutgoing,
		reply:   DirectionIncoming,
	}

	// Create the logger to log messages with (can be overriden by external caller)
	packet.log = log.With().
		Str("envelope_id", envelopeID.String()).
		Str("direction", packet.request.String()).
		Logger()

	// Add parent to submessages
	packet.in.packet = packet
	packet.out.packet = packet

	envelopeOpts := []envelope.Option{
		envelope.WithEnvelopeID(envelopeID.String()),
		envelope.WithTransferState(transferState),
	}

	if packet.out.Envelope, err = envelope.New(payload, envelopeOpts...); err != nil {
		return nil, fmt.Errorf("could not create envelope for payload: %w", err)
	}

	// Apply options
	// NOTE: subclasses should not pass options into this method if using it to instantiate a packet.
	for _, opt := range opts {
		opt(packet)
	}

	return packet, nil
}

// SendReject creates a new packet with an error payload to send as an outgoing request.
func SendReject(reject *api.Error, envelopeID uuid.UUID, opts ...Option) (packet *Packet, err error) {
	packet = &Packet{
		in:      &Incoming{},
		out:     &Outgoing{},
		request: DirectionOutgoing,
		reply:   DirectionIncoming,
	}

	// Add parent to submessages
	packet.in.packet = packet
	packet.out.packet = packet

	// The envelope package should set the correct transfer state based on retry.
	if packet.out.Envelope, err = envelope.WrapError(reject, envelope.WithEnvelopeID(envelopeID.String())); err != nil {
		return nil, fmt.Errorf("could not create rejection envelope: %w", err)
	}

	// Apply options
	// NOTE: subclasses should not pass options into this method if using it to instantiate a packet.
	for _, opt := range opts {
		opt(packet)
	}

	return packet, nil
}

// Receive is the basic mechanism to create a new packet whose request is incoming.
func Receive(envelope *envelope.Envelope, opts ...Option) (packet *Packet) {
	packet = &Packet{
		in:      &Incoming{Envelope: envelope},
		out:     &Outgoing{},
		request: DirectionIncoming,
		reply:   DirectionOutgoing,
	}

	// Create the logger to log messages with (can be overriden by external caller)
	packet.log = log.With().
		Str("envelope_id", envelope.ID()).
		Str("direction", packet.request.String()).
		Logger()

	// Add parent to submessages
	packet.in.packet = packet
	packet.out.packet = packet

	// Apply options
	// NOTE: subclasses should not pass options into this method if using it to instantiate a packet.
	for _, opt := range opts {
		opt(packet)
	}

	return packet
}

//===========================================================================
// Packet Helper Methods
//===========================================================================

// Returns the envelopeID from the request envelope (e.g. the first envelope in the packet)
func (p *Packet) EnvelopeID() string {
	switch p.request {
	case DirectionIncoming:
		return p.in.Envelope.ID()
	case DirectionOutgoing:
		return p.out.Envelope.ID()
	default:
		panic("request direction not set on packet")
	}
}

// Updates the transaction from the current state in the database.
func (p *Packet) RefreshTransaction() (err error) {
	if p.txn, err = p.db.Fetch(); err != nil {
		return err
	}
	return nil
}

// Ready checks if the packet has everything it needs to perform its work.
func (p *Packet) Ready() (err error) {
	if p.db == nil {
		err = errors.Join(err, ErrDatabaseNotReady)
	}

	if p.txn == nil {
		err = errors.Join(err, ErrTransactionNotReady)
	}

	if p.counterparty == nil {
		err = errors.Join(err, ErrCounterpartyNotReady)
	}

	if !p.request.Valid() || !p.reply.Valid() || p.in == nil || p.out == nil {
		err = errors.Join(err, ErrPacketNotReady)
	}

	return err
}

//===========================================================================
// Packet Accessor Methods (to avoid direct setting of fields)
//===========================================================================

func (p *Packet) DB() models.PreparedTransaction {
	return p.db
}

func (p *Packet) In() *Incoming {
	return p.in
}

func (p *Packet) Out() *Outgoing {
	return p.out
}

func (p *Packet) Transaction() *models.Transaction {
	return p.txn
}

func (p *Packet) Counterparty() *models.Counterparty {
	return p.counterparty
}

func (p *Packet) Request() Direction {
	return p.request
}

func (p *Packet) Reply() Direction {
	return p.reply
}
