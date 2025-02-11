package trp

import (
	"mime"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/trisacrypto/envoy/pkg"
	"github.com/trisacrypto/envoy/pkg/web/api/v1"
	"github.com/trisacrypto/trisa/pkg/openvasp"
)

var (
	ctxAPIVersionKey = "apiVersion"
	ctxExtensionsKey = "apiExtensions"
	ctxIdentifierKey = "requestIdentifier"
)

// Checks the headers in the TRP request are correct and valid and that the core TRP
// protocol requirements are met before passing the request to the handler.
func APICheck(c *gin.Context) {
	// Enforce application version
	apiVersion := c.Request.Header.Get(openvasp.APIVersionHeader)
	if !supportedAPIVersions(apiVersion) {
		log.Warn().Str("version", apiVersion).Msg("unsupported API version")
		c.AbortWithError(http.StatusBadRequest, ErrSupportedVersions)
		return
	}

	// Set the APIVersion header in the outgoing response
	c.Header(openvasp.APIVersionHeader, openvasp.APIVersion)
	c.Set(ctxAPIVersionKey, apiVersion)

	// If API Extenions are set, add them to the context.
	if apiExtensions := c.Request.Header.Get(openvasp.APIExtensionsHeader); apiExtensions != "" {
		c.Set(ctxExtensionsKey, apiExtensions)
	}

	c.Next()
}

// VerifyTRPCore checks the request identifier and content type match the core TRP protocol.
// NOTE that all core TRP requests must be POST requests with JSON content type.
func VerifyTRPCore(c *gin.Context) {
	// This will override anything in the routes definition.
	if c.Request.Method != http.MethodPost {
		c.AbortWithError(http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	// A request identifier is required for all TRP requests (even discoverability)
	var requestIdentifier string
	if requestIdentifier = c.Request.Header.Get(openvasp.RequestIdentifierHeader); requestIdentifier == "" {
		c.AbortWithError(http.StatusBadRequest, ErrMissingRequestIdentifier)
		return
	}

	// Echo back the request identifier in the response
	c.Header(openvasp.RequestIdentifierHeader, requestIdentifier)
	c.Set(ctxIdentifierKey, requestIdentifier)

	// Enforce JSON content type; if no content-type is specified assume JSON.
	if contentType := c.Request.Header.Get(openvasp.ContentTypeHeader); contentType != "" {
		mt, _, err := mime.ParseMediaType(contentType)
		if err != nil {
			log.Debug().Err(err).Str("content_type", contentType).Msg("could not parse media type")
			c.AbortWithError(http.StatusUnsupportedMediaType, ErrMalformedContentType)
			return
		}

		if mt != openvasp.MIMEJSON {
			log.Warn().Str("content_type", contentType).Str("media_type", mt).Msg("unsupported media type")
			c.AbortWithError(http.StatusUnsupportedMediaType, ErrUnsupportedContentType)
			return
		}
	}

	c.Next()
}

// If the server is in maintenance mode, aborts the current request and renders the
// maintenance mode page instead. Returns nil if not in maintenance mode.
func (s *Server) Maintenance() gin.HandlerFunc {
	if s.conf.Maintenance {
		return func(c *gin.Context) {
			c.JSON(http.StatusServiceUnavailable, &api.StatusReply{
				Status:  serverStatusMaintenance,
				Version: pkg.Version(),
				Uptime:  time.Since(s.started).String(),
			})
			c.Abort()
		}
	}
	return nil
}
