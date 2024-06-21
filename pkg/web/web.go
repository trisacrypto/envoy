package web

import (
	"net/http"
	"strings"
	"time"

	"github.com/trisacrypto/envoy/pkg/config"
	"github.com/trisacrypto/envoy/pkg/store"
	"github.com/trisacrypto/envoy/pkg/trisa/network"
	"github.com/trisacrypto/envoy/pkg/web/auth"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

// Create a new web server that serves the compliance and admin web user interface.
func New(conf config.Config, store store.Store, network network.Network) (s *Server, err error) {
	if err = conf.Web.Validate(); err != nil {
		return nil, err
	}

	s = &Server{
		conf:       conf.Web,
		globalConf: conf,
		store:      store,
		trisa:      network,
	}

	// If not enabled, return just the server stub
	if !s.conf.Enabled {
		return s, nil
	}

	// Configure the token issuer if enabled
	if s.issuer, err = auth.NewIssuer(s.conf.Auth); err != nil {
		return nil, err
	}

	// Configure the gin router if enabled
	s.router = gin.New()
	s.router.RedirectTrailingSlash = true
	s.router.RedirectFixedPath = false
	s.router.HandleMethodNotAllowed = true
	s.router.ForwardedByClientIP = true
	s.router.UseRawPath = false
	s.router.UnescapePathValues = true
	if err = s.setupRoutes(); err != nil {
		return nil, err
	}

	// Create the http server if enabled
	s.srv = &http.Server{
		Addr:         s.conf.BindAddr,
		Handler:      s.router,
		ErrorLog:     nil,
		ReadTimeout:  20 * time.Second,
		WriteTimeout: 20 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return s, nil
}

// If the API is disabled, this middleware factory function returns a middleware that
// returns 529 Unavailable on all API requests. If the API is enabled, it returns nil
// and should not be used as a middleware function.
func (s *Server) APIEnabled() gin.HandlerFunc {
	if s.conf.APIEnabled {
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
	if s.conf.UIEnabled {
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

// Debug returns a server that uses the specified http server instead of creating one.
// This function is primarily used to create test servers easily.
func Debug(conf config.Config, store store.Store, network network.Network, srv *http.Server) (s *Server, err error) {
	if s, err = New(conf, store, network); err != nil {
		return nil, err
	}

	// Replace the http server with the one specified
	s.srv = nil
	s.srv = srv
	s.srv.Handler = s.router
	return s, nil
}
