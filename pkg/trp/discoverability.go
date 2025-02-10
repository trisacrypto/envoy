package trp

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/trisacrypto/trisa/pkg/openvasp"
	"github.com/trisacrypto/trisa/pkg/openvasp/extensions/discoverability"
)

const (
	vendor              = "TRISA Envoy"
	extExtendedIVMS     = "extended-ivms101"
	extMessageSigning   = "message-signing"
	extSealedEnvelope   = "sealed-trisa-envelope"
	extUnsealedEnvelope = "unsealed-trisa-envelope"
)

// Implementation of the Discoverability Extension: returns the version and vendor.
// See: https://gitlab.com/OpenVASP/travel-rule-protocol/-/blob/master/extensions/discoverability.md
func (s *Server) TRPVersion(c *gin.Context) {
	c.JSON(http.StatusOK, s.version)
}

// Implementation of the Discoverability Extension: returns supported and required extensions.
// See: https://gitlab.com/OpenVASP/travel-rule-protocol/-/blob/master/extensions/discoverability.md
func (s *Server) TRPExtensions(c *gin.Context) {
	c.JSON(http.StatusOK, s.extensions)
}

// Implementation of the Discoverability Extension: returns the uptime in number of seconds
// See: https://gitlab.com/OpenVASP/travel-rule-protocol/-/blob/master/extensions/discoverability.md
func (s *Server) Uptime(c *gin.Context) {
	log.Debug().Msg("uptime discoverability request received")

	s.RLock()
	defer s.RUnlock()

	if !s.healthy || !s.ready {
		c.AbortWithStatus(http.StatusServiceUnavailable)
		return
	}

	uptime := discoverability.UptimeSince(s.started)
	data, err := uptime.MarshalText()
	if err != nil {
		log.Error().Err(err).Msg("could not marshal uptime")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.Data(http.StatusOK, openvasp.MIMEPlainText, data)
}

func (s *Server) initializeDiscoverability() {
	s.version = discoverability.Version{
		Version: openvasp.APIVersion,
		Vendor:  vendor,
	}

	s.extensions = discoverability.Extensions{
		Required: []string{
			extExtendedIVMS,
		},
		Supported: []string{
			// extMessageSigning,
			extSealedEnvelope,
			extUnsealedEnvelope,
		},
	}
}
