package trp

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/trisacrypto/envoy/pkg/logger"
	"github.com/trisacrypto/envoy/pkg/postman"
	"github.com/trisacrypto/trisa/pkg/openvasp"
	"github.com/trisacrypto/trisa/pkg/openvasp/trp/v3"
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
		Msg("processing TRP inquiry")

	if err = in.Validate(); err != nil {
		c.AbortWithError(http.StatusUnprocessableEntity, err)
		return
	}

	if err = in.IVMS101.Validate(); err != nil {
		c.AbortWithError(http.StatusUnprocessableEntity, err)
		return
	}

	if packet, err = postman.ReceiveTRPInquiry(in, c.Request.TLS); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	packet.Log = log

	envelopeID, _ := packet.In.Envelope.UUID()
	if packet.DB, err = s.store.PrepareTransaction(c.Request.Context(), envelopeID); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	// TODO: handle auto approve and auto reject
	out = &trp.Resolution{
		Version: openvasp.APIVersion,
	}

	log.Info().Msg("TRP inquiry received")
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
