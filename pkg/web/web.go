package web

import (
	"net/http"
	"time"

	"github.com/trisacrypto/envoy/pkg/config"
	"github.com/trisacrypto/envoy/pkg/store"
	"github.com/trisacrypto/envoy/pkg/trisa/network"
	"github.com/trisacrypto/envoy/pkg/web/auth"

	"github.com/gin-gonic/gin"
)

// Create a new web server that serves the compliance and admin web user interface.
func New(conf config.WebConfig, store store.Store, network network.Network) (s *Server, err error) {
	if err = conf.Validate(); err != nil {
		return nil, err
	}

	s = &Server{
		conf:  conf,
		store: store,
		trisa: network,
	}

	// If not enabled, return just the server stub
	if !conf.Enabled {
		return s, nil
	}

	// Configure the token issuer if enabled
	if s.issuer, err = auth.NewIssuer(conf.Auth); err != nil {
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

// Debug returns a server that uses the specified http server instead of creating one.
// This function is primarily used to create test servers easily.
func Debug(conf config.WebConfig, store store.Store, network network.Network, srv *http.Server) (s *Server, err error) {
	if s, err = New(conf, store, network); err != nil {
		return nil, err
	}

	// Replace the http server with the one specified
	s.srv = nil
	s.srv = srv
	s.srv.Handler = s.router
	return s, nil
}
