package web

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/trisacrypto/envoy/pkg/logger"
	dberr "github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/envoy/pkg/trisa/peers"
	api "github.com/trisacrypto/envoy/pkg/web/api/v1"
	"github.com/trisacrypto/trisa/pkg/openvasp/traddr"
	trisa "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
	"github.com/trisacrypto/trisa/pkg/trisa/crypto/rsaoeap"
	"github.com/trisacrypto/trisa/pkg/trisa/envelope"
	"github.com/trisacrypto/trisa/pkg/trisa/keys"
)

// SendEnvelope performs the bulk of the work to send a TRISA or TRP transaction to the
// counterparty specified and storing both the outgoing and incoming secure envelopes in
// the database. This method is used to send the prepared transaction, to send envelopes
// for a transaction, and in the accept/reject workflows.
func (s *Server) SendEnvelope(ctx context.Context, outgoing *envelope.Envelope, counterparty *models.Counterparty, db models.EnvelopeStorage) (err error) {
	// Step 1: Determine if this is a TRISA or TRP transaction and use the correct handler
	// to send the outgoing message (which might be updated during the send process) and to
	// receive the incoming reply from the counterparty.
	var incoming *envelope.Envelope
	switch counterparty.Protocol {
	case models.ProtocolTRISA:
		if outgoing, incoming, err = s.SendTRISATransfer(ctx, outgoing, counterparty); err != nil {
			return err
		}
	case models.ProtocolTRP:
		// TODO: handle TRP transfers
		return errors.New("the outgoing TRP send protocol is not implemented yet but is coming soon")
	default:
		return fmt.Errorf("could not send secure envelope: unknown protocol %q", counterparty.Protocol)
	}

	// Step 2: Prepare to store the outgoing envelope by fetching the public key used to
	// seal the incoming envelope from key storage.
	var storageKey keys.PublicKey
	if storageKey, err = s.trisa.StorageKey(incoming.Proto().PublicKeySignature, counterparty.CommonName); err != nil {
		// TODO: use the default keys if the incoming key is not known
		return fmt.Errorf("could not fetch storage key: %w", err)
	}

	// Create the secure envelope model for the outgoing message
	storeOutgoing := models.FromOutgoingEnvelope(outgoing)

	// Update the cryptography on the outgoing message for storage (it needs to be
	// stored with local keys since it was encrypted for the recipient).
	if err = storeOutgoing.Reseal(storageKey, outgoing.Crypto()); err != nil {
		return fmt.Errorf("could not encrypt outgoing message for storage: %w", err)
	}

	// Save the outgoing envelope to the database
	if err = db.AddEnvelope(storeOutgoing); err != nil {
		return fmt.Errorf("could not store outgoing message: %w", err)
	}

	// Step 3: Save incoming envelope to the database (should be encrypted with keys we
	// sent during the key exchange process of the transfer).
	storeIncoming := models.FromIncomingEnvelope(incoming)
	if err = db.AddEnvelope(storeIncoming); err != nil {
		return fmt.Errorf("could not store incoming message: %w", err)
	}

	return nil
}

func (s *Server) SendTRISATransfer(ctx context.Context, outgoing *envelope.Envelope, counterparty *models.Counterparty) (_, incoming *envelope.Envelope, err error) {
	// Get the peer from the specified counterparty
	var peer peers.Peer
	if peer, err = s.trisa.LookupPeer(ctx, counterparty.CommonName, ""); err != nil {
		return nil, nil, fmt.Errorf("could not lookup peer for counterparty %q (%s): %w", counterparty.CommonName, counterparty.ID, err)
	}

	log := logger.Tracing(ctx).With().Str("peer", peer.Name()).Str("envelope_id", outgoing.ID()).Logger()
	log.Debug().Msg("started outgoing TRISA transfer")

	// Fetch cached sealing keys, if not available, perform a key exchange
	var sealingKey keys.PublicKey
	if sealingKey, err = s.trisa.SealingKey(peer.Name()); err != nil {
		log.Debug().Msg("conducting key exchange prior to transer")
		if sealingKey, err = s.trisa.KeyExchange(ctx, peer); err != nil {
			log.Error().Err(err).Msg("cannot complete transfer without remote sealing keys")
			return nil, nil, fmt.Errorf("remote sealing keys unavailable, key exchange failed: %w", err)
		}
	}

	skey, _ := sealingKey.SealingKey()
	seal, _ := rsaoeap.New(skey)

	// Prepare outgoing envelope
	if !outgoing.IsError() {
		// Encrypt and seal the payload if this doesn't contain an error message
		if outgoing, _, err = outgoing.Encrypt(); err != nil {
			log.Error().Err(err).Msg("could not encrypt the outgoing secure envelope")
			return outgoing, nil, fmt.Errorf("outgoing encryption error occurred: %w", err)
		}

		if outgoing, _, err = outgoing.Seal(envelope.WithSeal(seal)); err != nil {
			log.Error().Err(err).Msg("could not seal the outgoing secure envelope")
			return outgoing, nil, fmt.Errorf("outgoing public key encryption error occurred: %w", err)
		}
	}

	var reply *trisa.SecureEnvelope
	if reply, err = peer.Transfer(ctx, outgoing.Proto()); err != nil {
		log.Error().Err(err).Msg("unable to send trisa transfer to remote peer")
		return outgoing, nil, fmt.Errorf("unexpected error returned from remote peer on transfer: %w", err)
	}

	// Load the unsealing key to unseal the response after transfer
	var unsealingKey keys.PrivateKey
	if unsealingKey, err = s.trisa.UnsealingKey(reply.PublicKeySignature, peer.Name()); err != nil {
		log.Error().Err(err).Str("pks", reply.PublicKeySignature).Msg("cannot identify unsealing keys used by remote")
		return nil, nil, fmt.Errorf("unsealing keys unavailable: %w", err)
	}

	if incoming, err = envelope.Wrap(reply, envelope.WithUnsealingKey(unsealingKey)); err != nil {
		log.Error().Err(err).Msg("unable to handle incoming secure envelope response from remote peer")
		return outgoing, nil, fmt.Errorf("unable to handle secure envelope from peer: %w", err)
	}

	// If the response is sealed, unseal and decrypt it (validating the HMAC signature)
	if incoming.State() == envelope.Sealed {
		unsealingKey.(keys.KeyIdentifier).PublicKeySignature()

		var privateKey interface{}
		if privateKey, err = unsealingKey.UnsealingKey(); err != nil {
			log.Error().Err(err).Msg("no private key available for unsealing")
			return outgoing, nil, fmt.Errorf("no private key available: %w", err)
		}

		if incoming, _, err = incoming.Unseal(envelope.WithUnsealingKey(privateKey)); err != nil {
			log.Error().Err(err).Msg("unable to unseal incoming secure envelope response from remote peer")
			return outgoing, nil, fmt.Errorf("unable to unseal secure envelope from peer: %w", err)
		}

		if incoming, _, err = incoming.Decrypt(); err != nil {
			log.Error().Err(err).Msg("unable to decrypt incoming secure envelope response from remote peer")
			return outgoing, nil, fmt.Errorf("unable to decrypt secure envelope from peer: %w", err)
		}
	}

	// HACK: Make sure the public key signature is on the incoming envelope
	// TODO: ensure that envelope package doesn't erase public key signature from envelope!
	incoming.Proto().PublicKeySignature = reply.PublicKeySignature
	return outgoing, incoming, nil
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
	// TODO: do we need the counterparty common name?
	var unsealingKey keys.PrivateKey
	if unsealingKey, err = s.trisa.UnsealingKey(in.PublicKey.String, ""); err != nil {
		return nil, err
	}

	var unseal interface{}
	if unseal, err = unsealingKey.UnsealingKey(); err != nil {
		return nil, err
	}

	// If the direction is outgoing, update the keys on the envelope
	if in.Direction == models.DirectionOutgoing {
		in.Envelope.EncryptionKey = in.EncryptionKey
		in.Envelope.HmacSecret = in.HMACSecret
	}

	// Wrap the secure envelope and unseal then decrypt it
	if out, err = envelope.Wrap(in.Envelope); err != nil {
		return nil, err
	}

	if out, _, err = out.Unseal(envelope.WithUnsealingKey(unseal)); err != nil {
		return nil, err
	}

	if out, _, err = out.Decrypt(); err != nil {
		return nil, err
	}

	return out, nil
}
