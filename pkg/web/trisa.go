package web

import (
	"context"
	"fmt"

	"github.com/trisacrypto/envoy/pkg/enum"
	"github.com/trisacrypto/envoy/pkg/postman"
	"github.com/trisacrypto/envoy/pkg/store/models"
	trisa "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
	"github.com/trisacrypto/trisa/pkg/trisa/envelope"
	"github.com/trisacrypto/trisa/pkg/trisa/keys"
)

// Deprecated: callers should be updated to use SendPacket instead.
func (s *Server) SendEnvelope(ctx context.Context, packet *postman.TRISAPacket) (err error) {
	// Step 1: Determine if this is a TRISA or TRP transaction and use the correct handler
	// to send the outgoing message (which might be updated during the send process) and to
	// receive the incoming reply from the counterparty.
	switch packet.Counterparty.Protocol {
	case enum.ProtocolTRISA:
		if err = s.SendTRISA(ctx, packet); err != nil {
			return err
		}
	default:
		return fmt.Errorf("could not send secure envelope: unknown protocol %q", packet.Counterparty.Protocol)
	}

	// Step 2: Store the outgoing envelope by fetching the public key used to seal the
	// incoming envelope from key storage. and saving to the database.
	if packet.Out.StorageKey, err = s.trisa.StorageKey(packet.In.PublicKeySignature(), packet.Counterparty.CommonName); err != nil {
		// TODO: use the default keys if the incoming key is not known
		return fmt.Errorf("could not fetch storage key: %w", err)
	}

	if err = packet.DB.AddEnvelope(packet.Out.Model()); err != nil {
		return fmt.Errorf("could not store outgoing envelope: %w", err)
	}

	// Step 3: Save incoming envelope to the database (should be encrypted with keys we
	// sent during the key exchange process of the transfer).
	if err = packet.DB.AddEnvelope(packet.In.Model()); err != nil {
		return fmt.Errorf("could not store incoming message: %w", err)
	}

	return nil
}

func (s *Server) SendTRISA(ctx context.Context, p *postman.TRISAPacket) (err error) {
	// Get the peer from the specified counterparty
	if p.Peer, err = s.trisa.LookupPeer(ctx, p.Counterparty.CommonName, ""); err != nil {
		return fmt.Errorf("could not lookup peer for counterparty %q (%s): %w", p.Counterparty.CommonName, p.Counterparty.ID, err)
	}

	p.Log = p.Log.With().Str("peer", p.Peer.Name()).Str("envelope_id", p.EnvelopeID()).Logger()
	p.Log.Debug().Msg("started outgoing TRISA transfer")

	// Add the peer info to the packet
	if p.PeerInfo, err = p.Peer.Info(); err != nil {
		p.Log.Debug().Err(err).Msg("unable to update peer info on packet")
		return fmt.Errorf("could not fetch peer info for counterparty: %w", err)
	}

	// Fetch cached sealing keys, if not available, perform a key exchange
	if p.Out.SealingKey, err = s.trisa.SealingKey(p.Peer.Name()); err != nil {
		p.Log.Debug().Msg("conducting key exchange prior to transer")
		if p.Out.SealingKey, err = s.trisa.KeyExchange(ctx, p.Peer); err != nil {
			p.Log.Error().Err(err).Msg("cannot complete transfer without remote sealing keys")
			return fmt.Errorf("remote sealing keys unavailable, key exchange failed: %w", err)
		}
	}
	// Prepare outgoing envelope
	if !p.Out.Envelope.IsError() {
		if _, err = p.Out.Seal(); err != nil {
			p.Log.Error().Err(err).Msg("could not seal outgoing envelope")
			return fmt.Errorf("could not seal outgoing envelope: %w", err)
		}
	}

	var reply *trisa.SecureEnvelope
	if reply, err = p.Peer.Transfer(ctx, p.Out.Proto()); err != nil {
		p.Log.Error().Err(err).Msg("unable to send trisa transfer to remote peer")
		return fmt.Errorf("unexpected error returned from remote peer on transfer: %w", err)
	}

	if err = p.Receive(reply); err != nil {
		p.Log.Error().Err(err).Msg("unable to prepare incoming message")
		return err
	}

	// Load the unsealing key to unseal the response after transfer
	if p.In.UnsealingKey, err = s.trisa.UnsealingKey(reply.PublicKeySignature, p.Peer.Name()); err != nil {
		p.Log.Error().Err(err).Str("pks", reply.PublicKeySignature).Msg("cannot identify unsealing keys used by remote")
		return fmt.Errorf("unsealing keys unavailable: %w", err)
	}

	// If the response is sealed, unseal and decrypt it (validating the HMAC signature)
	if p.In.Envelope.State() == envelope.Sealed {
		if _, err = p.In.Open(); err != nil {
			p.Log.Error().Err(err).Msg("unable to unseal incoming secure envelope response from remote peer")
			return fmt.Errorf("unable to unseal secure envelope from peer: %w", err)
		}
	}

	return nil
}

func (s *Server) Decrypt(in *models.SecureEnvelope) (out *envelope.Envelope, err error) {
	// No decryption is necessary if this is an error envelope
	if in.IsError {
		return envelope.Wrap(in.Envelope)
	}

	// Ensure that we have a public key to decrypt with
	if !in.PublicKey.Valid {
		return nil, ErrNoPublicKey
	}

	var unsealingKey keys.PrivateKey
	if unsealingKey, err = s.trisa.UnsealingKey(in.PublicKey.String, in.Remote.String); err != nil {
		return nil, fmt.Errorf("could not lookup unsealing key for secure envelope: %w", err)
	}

	// If the direction is outgoing, update the keys on the envelope
	if in.Direction == enum.DirectionOutgoing {
		in.Envelope.EncryptionKey = in.EncryptionKey
		in.Envelope.HmacSecret = in.HMACSecret
	}

	if out, _, err = envelope.Open(in.Envelope, envelope.WithUnsealingKey(unsealingKey)); err != nil {
		return nil, err
	}

	return out, nil
}
