package web

import (
	"context"
	"fmt"

	"github.com/trisacrypto/envoy/pkg/postman"
	"github.com/trisacrypto/envoy/pkg/store/models"
	trisa "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
	"github.com/trisacrypto/trisa/pkg/trisa/envelope"
	"github.com/trisacrypto/trisa/pkg/trisa/keys"
)

func (s *Server) SendTRISATransfer(ctx context.Context, p *postman.Packet) (err error) {
	// Get the peer from the specified counterparty
	if p.Peer, err = s.trisa.LookupPeer(ctx, p.Counterparty.CommonName, ""); err != nil {
		return fmt.Errorf("could not lookup peer for counterparty %q (%s): %w", p.Counterparty.CommonName, p.Counterparty.ID, err)
	}

	p.Log = p.Log.With().Str("method", "trisa").Str("peer", p.Peer.Name()).Str("envelope_id", p.EnvelopeID()).Logger()
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
	if in.Direction == models.DirectionOutgoing {
		in.Envelope.EncryptionKey = in.EncryptionKey
		in.Envelope.HmacSecret = in.HMACSecret
	}

	if out, _, err = envelope.Open(in.Envelope, envelope.WithUnsealingKey(unsealingKey)); err != nil {
		return nil, err
	}

	return out, nil
}
