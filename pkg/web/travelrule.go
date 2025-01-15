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
		if err = s.SendTRPMessage(ctx, packet); err != nil {
			return err
		}
	case models.ProtocolSunrise:
		return errors.New("sunrise protocol send is not implemented yet")
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
