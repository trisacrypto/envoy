package trp

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/trisacrypto/envoy/pkg/logger"
	"github.com/trisacrypto/envoy/pkg/postman"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/trisa/pkg/openvasp"
	"github.com/trisacrypto/trisa/pkg/openvasp/trp/v3"
	"github.com/trisacrypto/trisa/pkg/trisa/keys"
)

func (s *Server) Inquiry(c *gin.Context) {
	var (
		err    error
		in     *trp.Inquiry
		out    *trp.Resolution
		packet *postman.TRPPacket
	)

	in = &trp.Inquiry{
		Info: TRPInfo(c),
	}

	ctx := c.Request.Context()
	log := logger.Tracing(ctx).With().
		Str("request_identifier", in.Info.RequestIdentifier).
		Str("api_version", in.Info.APIVersion).
		Strs("api_extensions", in.Info.APIExtensions).
		Logger()

	// Parse and validate the JSON inquiry
	if err = c.BindJSON(in); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	log.Debug().
		Str("address", in.Info.Address).
		Msg("processing incoming trp inquiry")

	if err = in.Validate(); err != nil {
		c.AbortWithError(http.StatusUnprocessableEntity, err)
		return
	}

	// if err = in.IVMS101.Validate(); err != nil {
	// 	c.AbortWithError(http.StatusUnprocessableEntity, err)
	// 	return
	// }

	if packet, err = postman.ReceiveTRPInquiry(in, c.Request.TLS); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	packet.Log = log
	//FIXME: COMPLETE AUDIT LOG
	if packet.DB, err = s.store.PrepareTransaction(c.Request.Context(), packet.EnvelopeID(), &models.ComplianceAuditLog{}); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	// Rollback the prepared transaction if there are any errors in processing
	defer packet.DB.Rollback()

	// Create the transaction from the payload
	packet.Transaction = postman.TransactionFromPayload(packet.Payload())

	// Update the transaction record and add counterparty information and status
	// TODO: this may return an invalid counterparty error, which should return a different status error
	if err = packet.In.UpdateTransaction(); err != nil {
		log.Warn().Err(err).Bool("stored_to_database", false).Msg("could not update transaction details and counterparty information")
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	// TODO: load auto approve/reject policies for counterparty to determine response

	// Determine how to construct a response back to the remote counterparty; e.g. by
	// using the webhook, using an automated policy, or making an automatic response
	// determined by the transfer state .
	switch {
	case s.WebhookEnabled():
		if out, err = s.WebhookInquiry(packet); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
	default:
		out = &trp.Resolution{
			Version: openvasp.APIVersion,
		}
	}

	// Handle the outgoing message
	if err = packet.Resolve(out); err != nil {
		log.Error().Err(err).Bool("stored_to_database", false).Msg("could not resolve outgoing trp inquiry")
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	// Get the storage key and seal the envelope
	// TODO: handle the secure-trisa-envelope case where encryption is required
	var storageKey keys.PublicKey
	if storageKey, err = s.trisa.StorageKey("", packet.CommonName()); err != nil {
		log.Error().Err(err).Bool("stored_to_database", false).Msg("could not get storage key for trp inquiry")
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	if err = packet.Seal(storageKey); err != nil {
		log.Error().Err(err).Bool("stored_to_database", false).Msg("could not seal trp inquiry")
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	// Store Incoming Message
	//FIXME: COMPLETE AUDIT LOG
	if err = packet.DB.AddEnvelope(packet.In.Model(), &models.ComplianceAuditLog{}); err != nil {
		log.Error().Err(err).Bool("stored_to_database", false).Msg("could not store incoming trp inquiry in database")
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	// Store Outgoing message
	//FIXME: COMPLETE AUDIT LOG
	if err = packet.DB.AddEnvelope(packet.Out.Model(), &models.ComplianceAuditLog{}); err != nil {
		log.Error().Err(err).Bool("stored_to_database", false).Msg("could not store outgoing trp inquiry in database")
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	// Update the transaction with the outgoing message info
	if err = packet.Out.UpdateTransaction(); err != nil {
		log.Error().Err(err).Bool("stored_to_database", false).Msg("could not update transaction with outgoing info in database")
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	// Commit the transaction to the database (success!)
	if err = packet.DB.Commit(); err != nil {
		log.Warn().Err(err).Bool("stored_to_database", false).Msg("could not commit incoming trisa transfer to database")
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	log.Info().Bool("stored_to_database", true).Msg("incoming trp inquiry handling complete")
	c.JSON(http.StatusOK, out)
}

func (s *Server) Resolve(c *gin.Context) {
	log.Info().Msg("TRP resolve received")

	// A 204 should be sent in response to a transfer inquiry resolution.
	c.Status(http.StatusNoContent)
}

func (s *Server) Confirmation(c *gin.Context) {
	log.Info().Msg("TRP confirmation received")

	// A 204 should be sent in response to a transfer confirmation.
	c.Status(http.StatusNoContent)
}

// Get the TRP info from the context as set by the VerifyTRPCore middleware.
func TRPInfo(c *gin.Context) *trp.Info {
	info := &trp.Info{
		Address: c.Request.URL.String(),
	}

	if val, ok := c.Get(ctxAPIVersionKey); ok {
		info.APIVersion = val.(string)
	}

	if val, ok := c.Get(ctxIdentifierKey); ok {
		info.RequestIdentifier = val.(string)
	}

	if val, ok := c.Get(ctxExtensionsKey); ok {
		info.APIExtensions = strings.Split(val.(string), ",")
		for i, val := range info.APIExtensions {
			info.APIExtensions[i] = strings.TrimSpace(val)
		}
	}

	return info
}

//===========================================================================
// Webhook Interactions
//===========================================================================

func (s *Server) WebhookEnabled() bool {
	return false
}

func (s *Server) WebhookInquiry(packet *postman.TRPPacket) (out *trp.Resolution, err error) {
	return nil, errors.New("webhook inquiry not implemented")
}
