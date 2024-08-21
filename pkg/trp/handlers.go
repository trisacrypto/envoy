package trp

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/trisacrypto/trisa/pkg/openvasp"
)

// OnInquiry implements the openvasp.InquiryHandler interface for handling incoming TRP
// requests and creating transactions in the database.
func (s *Server) OnInquiry(*openvasp.Inquiry) (*openvasp.InquiryResolution, error) {
	log.Info().Msg("TRP inquiry received")
	return nil, errors.New("endpoint not implemented")
}

// OnConfirmation implements the openvasp.ConfirmationHandler interface for finalizing
// TRP transfer requests with details about the completed on-chain transaction.
func (s *Server) OnConfirmation(*openvasp.Confirmation) error {
	log.Info().Msg("TRP confirmation received")
	return errors.New("endpoint not implemented")
}

// Implementation of the Discoverability Extension: returns the version and vendor.
// See: https://gitlab.com/OpenVASP/travel-rule-protocol/-/blob/master/extensions/discoverability.md
func (s *Server) TRPVersion(c *gin.Context) {
	log.Debug().Msg("trp version discoverability request received")
	c.JSON(http.StatusOK, gin.H{
		"version": openvasp.APIVersion,
		"vendor":  "Rotational Labs",
	})
}

// Implementation of the Discoverability Extension: returns supported and required extensions.
// See: https://gitlab.com/OpenVASP/travel-rule-protocol/-/blob/master/extensions/discoverability.md
func (s *Server) TRPExtensions(c *gin.Context) {
	log.Debug().Msg("extensions discoverability request received")
	// TODO: allow configuration of these values
	c.JSON(http.StatusOK, gin.H{
		"required": []string{},
		"supported": []string{
			"extended-ivms101",
			"message-signing",
			"sealed-trisa-envelope",
			"unsealed-trisa-envelope",
		},
	})
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

	uptime := int64(time.Since(s.started).Seconds())
	c.Data(http.StatusOK, "text/plain", []byte(strconv.FormatInt(uptime, 10)))
}

func (s *Server) Identity(c *gin.Context) {
	log.Info().Msg("identity request received")
	c.Status(http.StatusNoContent)
}
