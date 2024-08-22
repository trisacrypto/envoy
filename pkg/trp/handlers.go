package trp

import (
	"encoding/pem"
	"errors"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/trisacrypto/envoy/pkg/trp/api/v1"
	"github.com/trisacrypto/trisa/pkg/openvasp"
	"github.com/trisacrypto/trisa/pkg/trust"
)

var (
	static     sync.Once
	version    *api.TRPVersion
	extensions *api.TRPExtensions
	identity   *api.Identity
)

const (
	vendor              = "TRISA Envoy"
	extExtendedIVMS     = "extended-ivms101"
	extMessageSigning   = "message-signing"
	extSealedEnvelope   = "sealed-trisa-envelope"
	extUnsealedEnvelope = "unsealed-trisa-envelope"
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

// Middleware for validating TRP protocol headers and ensuring a correct request.
func (s *Server) VerifyTRPHeaders(c *gin.Context) {
	// TODO: verify the api version can be handled
	if apiVersion := c.Request.Header.Get(openvasp.APIVersionHeader); apiVersion == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "missing trp api version in header"})
		return
	}

	// Set the APIVersion header in the outgoing response
	c.Header(openvasp.APIVersionHeader, openvasp.APIVersion)

	var requestIdentifier string
	if requestIdentifier = c.Request.Header.Get(openvasp.RequestIdentifierHeader); requestIdentifier == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "missing request identifier in header"})
		return
	}

	// Set the request identifier in the outgoing response
	c.Header(openvasp.RequestIdentifierHeader, requestIdentifier)

	// If we're getting discovery requests, initialize the discovery static structs
	static.Do(func() {
		version = &api.TRPVersion{
			Version: openvasp.APIVersion,
			Vendor:  vendor,
		}

		extensions = &api.TRPExtensions{
			Required: []string{},
			Supported: []string{
				extExtendedIVMS,
				extMessageSigning,
				extSealedEnvelope,
				extUnsealedEnvelope,
			},
		}

		identity = &api.Identity{
			Name: s.conf.TRP.Identity.VASPName,
			LEI:  s.conf.TRP.Identity.LEI,
		}

		if identity.Name == "" {
			identity.Name = s.conf.Organization
		}

		if s.conf.TRP.UseMTLS {
			// NOTE: ignoring errors assuming that mTLS has already been configured.
			var certs *trust.Provider
			switch {
			case s.conf.TRP.Certs != "":
				certs, _ = s.conf.TRP.LoadCerts()
			case s.conf.Node.Certs != "":
				certs, _ = s.conf.Node.LoadCerts()
			}

			x509, _ := certs.GetLeafCertificate()
			block := &pem.Block{Type: "CERTIFICATE", Bytes: x509.Raw}
			identity.Certs = string(pem.EncodeToMemory(block))
		}

	})
}

// Implementation of the Discoverability Extension: returns the version and vendor.
// See: https://gitlab.com/OpenVASP/travel-rule-protocol/-/blob/master/extensions/discoverability.md
func (s *Server) TRPVersion(c *gin.Context) {
	log.Debug().Msg("trp version discoverability request received")
	c.JSON(http.StatusOK, version)
}

// Implementation of the Discoverability Extension: returns supported and required extensions.
// See: https://gitlab.com/OpenVASP/travel-rule-protocol/-/blob/master/extensions/discoverability.md
func (s *Server) TRPExtensions(c *gin.Context) {
	log.Debug().Msg("extensions discoverability request received")
	c.JSON(http.StatusOK, extensions)
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
	c.JSON(http.StatusOK, identity)
}
