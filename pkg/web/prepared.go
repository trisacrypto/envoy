package web

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/trisacrypto/envoy/pkg/logger"
	dberr "github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/envoy/pkg/trisa/peers"
	"github.com/trisacrypto/envoy/pkg/web/api/v1"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/trisacrypto/trisa/pkg/ivms101"
	"github.com/trisacrypto/trisa/pkg/openvasp/traddr"
	trisa "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
	"github.com/trisacrypto/trisa/pkg/trisa/crypto/rsaoeap"
	"github.com/trisacrypto/trisa/pkg/trisa/envelope"
	"github.com/trisacrypto/trisa/pkg/trisa/keys"
)

func (s *Server) PrepareTransaction(c *gin.Context) {
	var (
		err             error
		in              *api.Prepare
		out             *api.Prepared
		beneficiaryVASP *models.Counterparty
		originatorVASP  *models.Counterparty
	)

	in = &api.Prepare{}
	if err = c.BindJSON(in); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse prepare transaction data"))
		return
	}

	if err = in.Validate(); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error(err))
		return
	}

	// Get originator VASP information from database
	if originatorVASP, err = s.Localparty(c.Request.Context()); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not complete prepare request"))
		return
	}

	// Parse the TravelAddress to identify the beneficiary VASP and lookup the
	// counterparty in the local database for IVMS101 information if any.
	if beneficiaryVASP, err = s.CounterpartyFromTravelAddress(c, in.TravelAddress); err != nil {
		// NOTE: CounterpartyFromTravelAddress handles API response back to user.
		return
	}

	// Convert the incoming data into the appropriate TRISA data structures
	out = &api.Prepared{
		TravelAddress: in.TravelAddress,
		Identity: &ivms101.IdentityPayload{
			Originator: &ivms101.Originator{
				OriginatorPersons: []*ivms101.Person{
					in.Originator.NaturalPerson(),
				},
				AccountNumbers: []string{
					in.Originator.CryptoAddress,
				},
			},
			Beneficiary: &ivms101.Beneficiary{
				BeneficiaryPersons: []*ivms101.Person{
					in.Beneficiary.NaturalPerson(),
				},
				AccountNumbers: []string{
					in.Beneficiary.CryptoAddress,
				},
			},
			OriginatingVasp: &ivms101.OriginatingVasp{
				OriginatingVasp: &ivms101.Person{
					Person: &ivms101.Person_LegalPerson{
						LegalPerson: originatorVASP.IVMSRecord,
					},
				},
			},
			BeneficiaryVasp: &ivms101.BeneficiaryVasp{
				BeneficiaryVasp: &ivms101.Person{
					Person: &ivms101.Person_LegalPerson{
						LegalPerson: beneficiaryVASP.IVMSRecord,
					},
				},
			},
			TransferPath:    nil,
			PayloadMetadata: nil,
		},
		Transaction: in.Transaction(),
	}

	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     out,
		HTMLName: "transaction_preview.html",
	})
}

func (s *Server) SendPreparedTransaction(c *gin.Context) {
	var (
		err          error
		in           *api.Prepared
		out          *api.Transaction
		model        *models.Transaction
		envelopeID   uuid.UUID
		db           models.PreparedTransaction
		counterparty *models.Counterparty
		payload      *trisa.Payload
		outgoing     *envelope.Envelope
		incoming     *envelope.Envelope
	)

	in = &api.Prepared{}
	if err = c.BindJSON(in); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse prepared transaction data"))
		return
	}

	if err = in.Validate(); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error(err))
		return
	}

	// Lookup the counterparty from the travel address in the request
	if counterparty, err = s.CounterpartyFromTravelAddress(c, in.TravelAddress); err != nil {
		// NOTE: CounterpartyFromTravelAddress handles API response back to user.
		return
	}

	// Create the transaction in the database
	envelopeID = uuid.New()
	if db, err = s.store.PrepareTransaction(c.Request.Context(), envelopeID); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not create transfer"))
		return
	}
	defer db.Rollback()

	// Add the counterparty to the database associated with the transaction
	if err = db.AddCounterparty(counterparty); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not update transfer with counterparty"))
		return
	}

	// Create the outgoing payload and envelope
	if payload, err = in.Payload(); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not create payload for transfer"))
		return
	}

	if outgoing, err = envelope.New(payload, envelope.WithEnvelopeID(envelopeID.String())); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not create outgoing envelope for transfer"))
		return
	}

	// Send the secure envelope and get secure envelope response
	// TODO: determine if this is a TRISA or TRP transfer and send TRP
	if outgoing, incoming, err = s.SendTRISATransfer(c.Request.Context(), outgoing, counterparty); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("unable to send transfer to remote counterparty"))
		return
	}

	// Save outgoing envelope to the database
	storeOutgoing := models.FromEnvelope(outgoing)
	storeOutgoing.Direction = models.DirectionOutgoing
	storeOutgoing.ValidHMAC = sql.NullBool{Valid: true, Bool: true}

	// Fetch the public key for storing the outgoing envelope
	var storageKey keys.PublicKey
	if storageKey, err = s.trisa.StorageKey(incoming.Proto().PublicKeySignature, counterparty.CommonName); err != nil {
		c.Error(fmt.Errorf("could not fetch storage key: %w", err))
		c.JSON(http.StatusInternalServerError, api.Error("could not complete send transfer request"))
		return
	}

	// Update the cryptography on the outgoing message for storage
	if err = storeOutgoing.Reseal(storageKey, outgoing.Crypto()); err != nil {
		c.Error(fmt.Errorf("could not encrypt outgoing message for storage: %w", err))
		c.JSON(http.StatusInternalServerError, api.Error("could not complete send transfer request"))
		return
	}

	if err = db.AddEnvelope(storeOutgoing); err != nil {
		c.Error(fmt.Errorf("could not store outgoing message: %w", err))
		c.JSON(http.StatusInternalServerError, api.Error("could not complete send transfer request"))
		return
	}

	// Saving incoming envelope to the database
	storeIncoming := models.FromEnvelope(incoming)
	storeIncoming.Direction = models.DirectionIncoming
	storeIncoming.ValidHMAC = sql.NullBool{Valid: true, Bool: storeIncoming.Envelope.Sealed}

	if err = db.AddEnvelope(storeIncoming); err != nil {
		c.Error(fmt.Errorf("could not store incoming message: %w", err))
		c.JSON(http.StatusInternalServerError, api.Error("could not complete send transfer request"))
		return
	}

	// Read the record from the database to return to the user
	if model, err = db.Fetch(); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not complete send transfer request"))
		return
	}

	// Commit the transaction to the database
	if err = db.Commit(); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not complete send transfer request"))
		return
	}

	// Create the API response to send back to the user
	if out, err = api.NewTransaction(model); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not complete send transfer request"))
		return
	}

	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     out,
		HTMLName: "transaction_sent.html",
	})
}

func (s *Server) CounterpartyFromTravelAddress(c *gin.Context, address string) (cp *models.Counterparty, err error) {
	var (
		dst    string
		dstURI *traddr.URL
	)

	if dst, err = traddr.Decode(address); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse the travel address"))
		return nil, err
	}

	if dstURI, err = traddr.Parse(dst); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse travel address url"))
		return nil, err
	}

	if cp, err = s.store.LookupCounterparty(c.Request.Context(), dstURI.Hostname()); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("could not identify counterparty from travel address"))
			return nil, err
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not complete request"))
		return nil, err
	}

	return cp, nil
}

func (s *Server) SendTRISATransfer(ctx context.Context, outgoing *envelope.Envelope, counterparty *models.Counterparty) (_, incoming *envelope.Envelope, err error) {
	// TODO: generalize the input and output context like we have to receive TRISA envelopes

	// Get the peer from the specified counterparty
	var peer peers.Peer
	if peer, err = s.trisa.LookupPeer(ctx, counterparty.CommonName, ""); err != nil {
		return nil, nil, fmt.Errorf("could not lookup peer for counterparty %q (%s): %w", counterparty.CommonName, counterparty.ID, err)
	}

	log := logger.Tracing(ctx).With().Str("peer", peer.Name()).Str("envelope_id", outgoing.ID()).Logger()
	log.Debug().Msg("started outgoing TRISA transfer")

	// Load the unsealing key to unseal the response after transfer; this also has the
	// effect of checking that we have public keys to exchange with the remote peer.
	var unsealingKey keys.PrivateKey
	if unsealingKey, err = s.trisa.UnsealingKey("", peer.Name()); err != nil {
		log.Error().Err(err).Msg("cannot start transfer without unsealing keys")
		return nil, nil, fmt.Errorf("local unsealing keys unavailable: %w", err)
	}

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

	if incoming, err = envelope.Wrap(reply, envelope.WithUnsealingKey(unsealingKey)); err != nil {
		log.Error().Err(err).Msg("unable to handle incoming secure envelope response from remote peer")
		return outgoing, nil, fmt.Errorf("unable to handle secure envelope from peer: %w", err)
	}

	// If the response is sealed, unseal and decrypt it (validating the HMAC signature)
	if incoming.State() == envelope.Sealed {
		if incoming, _, err = incoming.Unseal(); err != nil {
			log.Error().Err(err).Msg("unable to unseal incoming secure envelope response from remote peer")
			return outgoing, nil, fmt.Errorf("unable to unseal secure envelope from peer: %w", err)
		}

		if incoming, _, err = incoming.Decrypt(); err != nil {
			log.Error().Err(err).Msg("unable to decrypt incoming secure envelope response from remote peer")
			return outgoing, nil, fmt.Errorf("unable to decrypt secure envelope from peer: %w", err)
		}
	}

	return outgoing, incoming, nil
}
