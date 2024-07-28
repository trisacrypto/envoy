package web

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/trisacrypto/envoy/pkg/postman"
	dberr "github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/models"
	api "github.com/trisacrypto/envoy/pkg/web/api/v1"
	"github.com/trisacrypto/trisa/pkg/openvasp/traddr"
	trisa "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
	"github.com/trisacrypto/trisa/pkg/trisa/envelope"
	"github.com/trisacrypto/trisa/pkg/trisa/keys"
)

// SendEnvelope performs the bulk of the work to send a TRISA or TRP transaction to the
// counterparty specified and storing both the outgoing and incoming secure envelopes in
// the database. This method is used to send the prepared transaction, to send envelopes
// for a transaction, and in the accept/reject workflows.
func (s *Server) SendEnvelope(ctx context.Context, packet *postman.Packet) (err error) {
	// Step 1: Determine if this is a TRISA or TRP transaction and use the correct handler
	// to send the outgoing message (which might be updated during the send process) and to
	// receive the incoming reply from the counterparty.
	switch packet.Counterparty.Protocol {
	case models.ProtocolTRISA:
		if err = s.SendTRISATransfer(ctx, packet); err != nil {
			return err
		}
	case models.ProtocolTRP:
		// TODO: handle TRP transfers
		return errors.New("the outgoing TRP send protocol is not implemented yet but is coming soon")
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

func (s *Server) SendTRISATransfer(ctx context.Context, p *postman.Packet) (err error) {
	// Get the peer from the specified counterparty
	if p.Peer, err = s.trisa.LookupPeer(ctx, p.Counterparty.CommonName, ""); err != nil {
		return fmt.Errorf("could not lookup peer for counterparty %q (%s): %w", p.Counterparty.CommonName, p.Counterparty.ID, err)
	}

	p.Log = p.Log.With().Str("peer", p.Peer.Name()).Str("envelope_id", p.EnvelopeID()).Logger()
	p.Log.Debug().Msg("started outgoing TRISA transfer")

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

func (s *Server) CounterpartyFromTravelAddress(c *gin.Context, address string) (cp *models.Counterparty, err error) {
	var (
		dst    string
		dstURI *traddr.URL
	)

	if dst, err = traddr.Decode(address); err != nil {
		c.Error(fmt.Errorf("could not decode travel address %q: %w", address, err))
		c.JSON(http.StatusBadRequest, api.Error("could not parse the travel address"))
		return nil, err
	}

	if dstURI, err = traddr.Parse(dst); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse travel address url"))
		return nil, err
	}

	if cp, err = s.findCounterparty(c.Request.Context(), dstURI); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.Error(fmt.Errorf("could not identify counterparty for %s or %s", dstURI.Hostname(), dstURI.Host))
			c.JSON(http.StatusNotFound, api.Error("could not identify counterparty from travel address"))
			return nil, err
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not complete request"))
		return nil, err
	}

	return cp, nil
}

func (s *Server) findCounterparty(ctx context.Context, uri *traddr.URL) (cp *models.Counterparty, err error) {
	// Lookup counterparty by hostname first (e.g. the common name).
	if cp, err = s.store.LookupCounterparty(ctx, uri.Hostname()); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			// If we couldn't find it, try again by endpoint
			// NOTE: this is primarily to assist with lookups for localhost where the
			// port number is the only differentiating aspect of the node.
			if cp, err = s.store.LookupCounterparty(ctx, uri.Host); err != nil {
				return nil, dberr.ErrNotFound
			}

			// Found! Short-circuit the error handling by returning early!
			return cp, err
		}

		// Return the internal error
		return nil, err
	}

	// Found on first try!
	return cp, nil
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
