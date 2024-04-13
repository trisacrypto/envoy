package web

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"self-hosted-node/pkg/config"
	"self-hosted-node/pkg/store"
	dberr "self-hosted-node/pkg/store/errors"
	"self-hosted-node/pkg/store/models"
	"self-hosted-node/pkg/web/auth"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/trisacrypto/trisa/pkg/openvasp/traddr"
)

// The Web Server implements the compliance and administrative user interfaces.
type Server struct {
	sync.RWMutex
	conf    config.WebConfig
	store   store.Store
	srv     *http.Server
	router  *gin.Engine
	issuer  *auth.ClaimsIssuer
	url     *url.URL
	vasp    *models.Counterparty
	started time.Time
	healthy bool
	ready   bool
}

// Serve the compliance and administrative user interfaces in its own go routine.
func (s *Server) Serve(errc chan<- error) (err error) {
	if !s.conf.Enabled {
		log.Warn().Bool("enabled", s.conf.Enabled).Msg("web ui is not enabled")
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

	log.Info().Str("url", s.URL()).Msg("compliance and admin web user interface started")
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
	if !s.conf.Enabled {
		return nil
	}

	log.Info().Msg("gracefully shutting down compliance and admin web user interface server")
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

// Localparty returns the VASP information for the current node.
func (s *Server) Localparty(ctx context.Context) (_ *models.Counterparty, err error) {
	if s.vasp == nil {
		// Parse TRISA endpoint
		var uri *traddr.URL
		if uri, err = traddr.Parse(s.conf.TRISAEndpoint); err != nil {
			return nil, fmt.Errorf("could not parse configured trisa endpoint: %w", err)
		}

		commonName := uri.Hostname()
		if commonName == "" {
			return nil, ErrNoLocalCommonName
		}

		// Lookup VASP information from counterparty database
		if s.vasp, err = s.store.LookupCounterparty(ctx, commonName); err != nil {
			log.Warn().Err(err).Msg("could not lookup local vasp information")
			if errors.Is(err, dberr.ErrNotFound) {
				return nil, ErrNoLocalparty
			}
			return nil, fmt.Errorf("could not lookup counterparty by common name: %w", err)
		}
	}
	return s.vasp, nil
}
