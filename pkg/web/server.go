package web

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"self-hosted-node/pkg/config"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// TODO: Replace with service and load static files.
// var views = jet.NewSet(jet.NewOSFileSystemLoader("templates"))

// func main() {
// 	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
// 		view, err := views.GetTemplate("/partials/test.jet")
// 		if err != nil {
// 			log.Println("Unexpected template error:", err.Error())
// 		}
// 		view.Execute(w, nil, nil)
// 	})

// 	http.ListenAndServe(":8080", nil)
// }

// The Web Server implements the compliance and administrative user interfaces.
type Server struct {
	sync.RWMutex
	conf   config.WebConfig
	srv    *http.Server
	router *gin.Engine
	url    *url.URL
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
	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	defer cancel()

	s.srv.SetKeepAlivesEnabled(false)
	if err = s.srv.Shutdown(ctx); err != nil {
		return err
	}

	return nil
}

// URL returns the endpoint of the server as determined by the configuration and the
// socket address and port (if specified).
func (s *Server) URL() string {
	return s.url.String()
}

func (s *Server) setURL(addr net.Addr) {
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
