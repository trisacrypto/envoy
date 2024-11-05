package web

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

// If the API is disabled, this middleware factory function returns a middleware that
// returns 529 Unavailable on all API requests. If the API is enabled, it returns nil
// and should not be used as a middleware function.
func (s *Server) APIEnabled() gin.HandlerFunc {
	if s.conf.Web.APIEnabled {
		return nil
	}

	return func(c *gin.Context) {
		// If this is an API request and the API is disabled, return unavailable.
		if IsAPIRequest(c) {
			c.AbortWithStatus(http.StatusServiceUnavailable)
			return
		}

		// Otherwise serve the UI request.
		c.Next()
	}
}

// If the UI is disabled, this middleware factory function returrns a middleware that
// returns 529 Unvailable on all UI requests. If the UI is enabled, it returns nil.
func (s *Server) UIEnabled() gin.HandlerFunc {
	if s.conf.Web.UIEnabled {
		return nil
	}

	return func(c *gin.Context) {
		// If this is not an API request, and the UI is disabled, return unavailable.
		if !IsAPIRequest(c) {
			c.AbortWithStatus(http.StatusServiceUnavailable)
			return
		}

		// Otherwise serve the API request.
		c.Next()
	}
}

// Determines if the request being handled is an API request by inspecting the request
// path and Accept header. If the request path starts in /v1 and the Accept header is
// nil or json, then this function returns true.
func IsAPIRequest(c *gin.Context) bool {
	if strings.HasPrefix(c.Request.URL.RequestURI(), "/v1") {
		return !(c.NegotiateFormat(binding.MIMEJSON, binding.MIMEHTML) == binding.MIMEHTML)
	}
	return false
}

func (s *Server) SunriseEnabled() gin.HandlerFunc {
	enabled := s.conf.Sunrise.Enabled
	return func(c *gin.Context) {
		if !enabled {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		// Handle the request if the route is enabled
		c.Next()
	}
}
