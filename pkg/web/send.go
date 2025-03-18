package web

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/trisacrypto/envoy/pkg/enum"
	"github.com/trisacrypto/envoy/pkg/logger"
	"github.com/trisacrypto/envoy/pkg/postman"
	api "github.com/trisacrypto/envoy/pkg/web/api/v1"

	trisa "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
)

// Send performs the bulk of the work to send a travel rule transfer to the
// counterparty specified and storing both the outgoing and incoming secure envelopes in
// the database. This method is used to send the prepared transaction, to send envelopes
// for a transaction, and in the accept/reject workflows.
func (s *Server) Send(c *gin.Context, routing *api.Routing, payload *trisa.Payload) (packet *postman.Packet, err error) {
	// Create a packet to begin the sending process
	envelopeID := uuid.New()
	if packet, err = postman.Send(envelopeID, payload, trisa.TransferStarted); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process send prepared transaction request"))
		return nil, err
	}

	// Add the log to the packet for debugging
	ctx := c.Request.Context()
	packet.Log = logger.Tracing(ctx).With().Str("envelope_id", envelopeID.String()).Logger()

	// Lookup the counterparty from the travel address in the request
	if packet.Counterparty, err = s.ResolveCounterparty(c, routing); err != nil {
		// NOTE: CounterpartyFromTravelAddress handles API response back to user.
		return nil, err
	}

	// Create the transaction in the database
	if packet.DB, err = s.store.PrepareTransaction(ctx, envelopeID); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process send prepared transaction request"))
		return nil, err
	}
	defer packet.DB.Rollback()

	// Add the counterparty to the database associated with the transaction
	// If the update fails, log the error but do not cancel processing.
	if err = packet.Out.UpdateTransaction(); err != nil {
		c.Error(err)
	}

	// The protocol was already parsed in ResolveCounterparty
	protocol, _ := enum.ParseProtocol(routing.Protocol)
	if packet, err = s.SendPacket(c, protocol, packet); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not send transfer message to counterparty"))
		return nil, err
	}

	// Update transaction state based on response from counterparty
	// If the update fails rollback and return the error.
	if err = packet.In.UpdateTransaction(); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process send prepared transaction request"))
		return nil, err
	}

	// Read the record from the database to return to the user
	if err = packet.RefreshTransaction(); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process send prepared transaction request"))
		return nil, err
	}

	// Commit the transaction to the database
	if err = packet.DB.Commit(); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process send prepared transaction request"))
		return nil, err
	}

	return packet, nil
}

func (s *Server) SendPacket(ctx context.Context, protocol enum.Protocol, packet *postman.Packet) (_ *postman.Packet, err error) {
	// Step 1: Determine the protocol and use the correct handler to send the outgoing
	// packet (which might be updated during the send process) and to receive the
	// incoming reply from the counterparty.
	switch protocol {
	case enum.ProtocolTRISA:
		wrapped := packet.TRISA()
		if err = s.SendTRISA(ctx, wrapped); err != nil {
			return nil, err
		}
		packet = &wrapped.Packet
	case enum.ProtocolTRP:
		return nil, errors.New("TRP sending is temporarily disabled as we refresh Envoy to v1.0.0")
		// if err = s.SendTRP(ctx, packet.TRP()); err != nil {
		// 	return err
		// }
	case enum.ProtocolSunrise:
		wrapped := packet.Sunrise()
		if err = s.SendSunrise(ctx, wrapped); err != nil {
			return nil, err
		}
		packet = &wrapped.Packet
	default:
		return nil, fmt.Errorf("unhandled protocol in send packet: %q", protocol.String())
	}

	// TODO: right now sunrise has a special envelope storage method, so we exit early
	// but we should unify this with the TRISA and TRP methods.
	if protocol == enum.ProtocolSunrise {
		return packet, nil
	}

	// Step 2: Store the outgoing envelope by fetching the public key used to seal the
	// incoming envelope from key storage. and saving to the database.
	if packet.Out.StorageKey, err = s.trisa.StorageKey(packet.In.PublicKeySignature(), packet.Counterparty.CommonName); err != nil {
		// TODO: use the default keys if the incoming key is not known
		return nil, fmt.Errorf("could not fetch storage key: %w", err)
	}

	if err = packet.DB.AddEnvelope(packet.Out.Model()); err != nil {
		return nil, fmt.Errorf("could not store outgoing envelope: %w", err)
	}

	// Step 3: Save incoming envelope to the database (should be encrypted with keys we
	// sent during the key exchange process of the transfer).
	if err = packet.DB.AddEnvelope(packet.In.Model()); err != nil {
		return nil, fmt.Errorf("could not store incoming message: %w", err)
	}

	return packet, nil
}
