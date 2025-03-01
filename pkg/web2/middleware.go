package web

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/rs/zerolog/log"
	"github.com/trisacrypto/envoy/pkg/web/api/v1"
	"github.com/trisacrypto/envoy/pkg/web/auth"
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

// Aborts the request with a 404 error if sunrise is not enabled.
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

// Authenticates sunrise users verifying the access token in the request. Does not allow
// API or JSON requests to be authenticated and the claims must have a sunrise subject.
// If the user is not authenticated they are redirected to the sunrise_404.html page.
//
// This middleware should only be on routes intended for use by sunrise users.
func (s *Server) SunriseAuthenticate(issuer *auth.ClaimsIssuer) gin.HandlerFunc {
	authenticate := func(c *gin.Context) (claims *auth.Claims, err error) {
		// Fetch the access token from the request
		var accessToken string
		if accessToken, err = auth.GetAccessToken(c); err != nil {
			log.Debug().Err(err).Msg("no access token in sunrise request")
			return nil, auth.ErrAuthRequired
		}

		if claims, err = issuer.Verify(accessToken); err != nil {
			log.Debug().Err(err).Msg("invalid access token in sunrise request")
			return nil, auth.ErrAuthRequired
		}

		return claims, nil
	}

	return func(c *gin.Context) {
		var (
			err    error
			claims *auth.Claims
		)

		// If this is an API request, do not authenticate
		if IsAPIRequest(c) {
			c.AbortWithStatusJSON(http.StatusNotFound, api.NotFound)
			return
		}

		// Authenticate the user
		if claims, err = authenticate(c); err != nil {
			log.Debug().Err(err).Msg("unauthorized sunrise request")

			// Redirect the user to the 404 page
			c.Abort()
			c.HTML(http.StatusNotFound, "sunrise_404.html", nil)
			return
		}

		// Check that the subject is a sunrise user and that the subject is valid.
		var subjectType auth.SubjectType
		if subjectType, _, err = claims.SubjectID(); err != nil || subjectType != auth.SubjectSunrise {
			log.Debug().Err(err).Str("subject_type", subjectType.String()).Msg("invalid subject in sunrise request")

			// Redirect the user to the 404 page
			c.Abort()
			c.HTML(http.StatusNotFound, "sunrise_404.html", nil)
			return
		}

		// Add claims to context for use in downstream processing
		c.Set(auth.ContextUserClaims, claims)
		c.Next()
	}
}
