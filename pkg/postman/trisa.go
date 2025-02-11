package postman

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"github.com/trisacrypto/envoy/pkg/trisa/peers"

	api "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
	"github.com/trisacrypto/trisa/pkg/trisa/envelope"
)

type TRISAPacket struct {
	Packet
	Peer     peers.Peer  // The remote peer the transfer is being conducted with
	PeerInfo *peers.Info // The peer info for finding the counterparty
}

func SendTRISA(envelopeID uuid.UUID, payload *api.Payload, transferState api.TransferState) (packet *TRISAPacket, err error) {
	var parent *Packet
	if parent, err = Send(envelopeID, payload, transferState); err != nil {
		return nil, err
	}

	packet = &TRISAPacket{
		Packet: *parent,
	}

	// Add parent to submessages
	packet.In.packet = &packet.Packet
	packet.Out.packet = &packet.Packet

	packet.Packet.resolver = packet
	return packet, nil
}

func SendTRISAReject(envelopeID uuid.UUID, reject *api.Error) (packet *TRISAPacket, err error) {
	var parent *Packet
	if parent, err = SendReject(envelopeID, reject); err != nil {
		return nil, err
	}

	packet = &TRISAPacket{
		Packet: *parent,
	}

	// Add parent to submessages
	packet.In.packet = &packet.Packet
	packet.Out.packet = &packet.Packet

	packet.Packet.resolver = packet
	return packet, nil
}

func ReceiveTRISA(in *api.SecureEnvelope, peer peers.Peer) (packet *TRISAPacket, err error) {
	var parent *Packet
	if parent, err = Receive(in); err != nil {
		return nil, err
	}

	packet = &TRISAPacket{
		Packet: *parent,
		Peer:   peer,
	}

	// Add parent to submessages
	packet.In.packet = &packet.Packet
	packet.Out.packet = &packet.Packet

	if packet.PeerInfo, err = peer.Info(); err != nil {
		return nil, fmt.Errorf("could not identify counterparty in transaction: %w", err)
	}

	packet.Counterparty = packet.PeerInfo.Model()
	packet.Packet.resolver = packet
	return packet, nil
}

// Receive updates the incoming message with the specified secure envelope, e.g. in the
// case where the outgoing message has been sent and this is the reply that was received
// from the remote server.
func (p *TRISAPacket) Receive(in *api.SecureEnvelope) (err error) {
	p.In = &Incoming{
		original: in,
		packet:   &p.Packet,
	}

	if p.In.Envelope, err = envelope.Wrap(in); err != nil {
		return fmt.Errorf("could not wrap incoming secure envelope: %w", err)
	}

	return nil
}

// Creates a TRISA payload envelope to as the outgoing message in order to send a
// response back to the remote that initiated the transfer.
func (p *TRISAPacket) Send(payload *api.Payload, state api.TransferState) (err error) {
	if p.Out.Envelope, err = p.In.Envelope.Update(payload, envelope.WithTransferState(state)); err != nil {
		p.Log.Debug().Err(err).Msg("could not prepare outgoing payload")
		return fmt.Errorf("could not create outgoing payload: %w", err)
	}
	return nil
}

// Reject creates an outgoing message with a TRISA error that contains the specified
// TRISA rejection error, e.g. in the case where an incoming message is being handled.
func (p *TRISAPacket) Reject(code api.Error_Code, message string, retry bool) error {
	reject := &api.Error{
		Code: code, Message: message, Retry: retry,
	}
	return p.Error(reject)
}

// Creates a TRISA error envelope from the TRISA error message and sets it as the
// outgoing message (e.g. in the case where an incoming message is being handled).
func (p *TRISAPacket) Error(reject *api.Error, opts ...envelope.Option) (err error) {
	if p.Out.Envelope, err = p.In.Envelope.Reject(reject, opts...); err != nil {
		p.Log.Debug().Err(err).Msg("could not prepare rejection envelope")
		return fmt.Errorf("could not create rejection envelope: %w", err)
	}
	return nil
}

func (p *TRISAPacket) ResolveCounterparty() (err error) {
	if p.Counterparty == nil {
		if p.PeerInfo == nil {
			if p.Peer == nil {
				return ErrNoCounterpartyInfo
			}

			if p.PeerInfo, err = p.Peer.Info(); err != nil {
				return err
			}
		}

		// TRISA counterparties are guaranteed to be in the database because they
		// come from the GDS and are synchronized.
		p.Counterparty = p.PeerInfo.Model()
	}
	return nil
}

func (p *TRISAPacket) Remote() sql.NullString {
	return sql.NullString{Valid: p.PeerInfo.CommonName != "", String: p.PeerInfo.CommonName}
}
