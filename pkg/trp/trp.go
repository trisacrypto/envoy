/*
 * Package trp implements a JSON web server for the Travel Rule Protocol that was
 * designed and developed by OpenVASP. This is a separate server from the rest of the
 * envoy services so that it can be enabled, authenticated, and managed independently.
 */
package trp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/trisacrypto/envoy/pkg/config"
	"github.com/trisacrypto/envoy/pkg/store"
	"github.com/trisacrypto/envoy/pkg/trisa/network"
	"github.com/trisacrypto/trisa/pkg/trisa/mtls"
	"github.com/trisacrypto/trisa/pkg/trust"
)

type Server struct {
	sync.RWMutex
	conf    config.Config
	store   store.Store
	srv     *http.Server
	router  *gin.Engine
	url     *url.URL
	trisa   network.Network
	started time.Time
	healthy bool
	ready   bool
}

func New(conf config.Config, store store.Store, network network.Network) (s *Server, err error) {
	if err = conf.TRP.Validate(); err != nil {
		return nil, err
	}

	s = &Server{
		conf:  conf,
		store: store,
		trisa: network,
	}

	// If not enabled, return just the server stub
	if !s.conf.TRP.Enabled {
		return s, nil
	}

	// Use TRISA certs if no TRP specific certs are passed in
	if conf.TRP.Certs == "" {
		conf.TRP.Certs = conf.Node.Certs
		conf.TRP.Pool = conf.Node.Pool
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
		Addr:              s.conf.TRP.BindAddr,
		Handler:           s.router,
		ErrorLog:          nil,
		ReadHeaderTimeout: 20 * time.Second,
		WriteTimeout:      20 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// Configure mTLS if enabled
	if s.conf.TRP.UseMTLS {
		var identity *trust.Provider
		if identity, err = conf.TRP.LoadCerts(); err != nil {
			return nil, fmt.Errorf("could not load mtls certs: %w", err)
		}

		var pool trust.ProviderPool
		if pool, err = conf.TRP.LoadPool(); err != nil {
			return nil, fmt.Errorf("could not load mtls pool: %w", err)
		}

		if s.srv.TLSConfig, err = mtls.Config(identity, pool); err != nil {
			return nil, fmt.Errorf("could not configure mtls: %w", err)
		}
	}

	return s, nil
}

// Serve the TRP API server
func (s *Server) Serve(errc chan<- error) (err error) {
	if !s.conf.TRP.Enabled {
		log.Warn().Bool("enabled", s.conf.TRP.Enabled).Msg("openvasp/trp server is not enabled")
		return nil
	}

	// Create a socket to listen on and infer the final URL.
	// NOTE: if the bindaddr is 127.0.0.1:0 for testing, a random port will be assigned,
	// manually creating the listener will allow us to determine which port.
	// When we start listening all incoming requests will be buffered until the server
	// actually starts up in its own go routine below.
	var sock net.Listener
	if sock, err = net.Listen("tcp", s.srv.Addr); err != nil {
		return fmt.Errorf("could not listen on bind addr %s: %s", s.srv.Addr, err)
	}

	s.setURL(sock.Addr())
	s.SetStatus(true, true)
	s.started = time.Now()

	// Listen for HTTP requests and handle them.
	go func() {
		// Make sure we don't use the external err to avoid data races.
		if serr := s.serve(sock); !errors.Is(serr, http.ErrServerClosed) {
			errc <- serr
		}
	}()

	log.Info().Str("url", s.URL()).Msg("openvasp/trp api server started")
	return nil
}

// ServeTLS if a tls configuration is provided, otherwise Serve.
func (s *Server) serve(sock net.Listener) error {
	if s.srv.TLSConfig != nil {
		return s.srv.ServeTLS(sock, "", "")
	}
	return s.srv.Serve(sock)
}

// Shutdown the web server gracefully.
func (s *Server) Shutdown() (err error) {
	// If the server is not enabled, skip shutdown.
	if !s.conf.TRP.Enabled {
		return nil
	}

	log.Info().Msg("gracefully shutting down openvasp/trp server")
	s.SetStatus(false, false)

	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	defer cancel()

	s.srv.SetKeepAlivesEnabled(false)
	if err = s.srv.Shutdown(ctx); err != nil {
		return err
	}

	return nil
}

// SetStatus sets the health and ready status on the server, modifying the behavior of
// the kubernetes probe responses.
func (s *Server) SetStatus(health, ready bool) {
	s.Lock()
	s.healthy = health
	s.ready = ready
	s.Unlock()
	log.Debug().Bool("health", health).Bool("ready", ready).Msg("server status set")
}

// URL returns the endpoint of the server as determined by the configuration and the
// socket address and port (if specified).
func (s *Server) URL() string {
	s.RLock()
	defer s.RUnlock()
	return s.url.String()
}

func (s *Server) setURL(addr net.Addr) {
	s.Lock()
	defer s.Unlock()

	s.url = &url.URL{
		Scheme: "http",
		Host:   addr.String(),
	}

	if s.srv.TLSConfig != nil {
		s.url.Scheme = "https"
	}

	if tcp, ok := addr.(*net.TCPAddr); ok && tcp.IP.IsUnspecified() {
		s.url.Host = fmt.Sprintf("127.0.0.1:%d", tcp.Port)
	}
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
