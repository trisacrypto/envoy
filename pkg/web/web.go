package web

import (
	"net/http"
	"os"
	"self-hosted-node/pkg/config"
	"self-hosted-node/pkg/logger"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func init() {
	// Initializes zerolog with our default logging requirements
	zerolog.TimeFieldFormat = time.RFC3339
	zerolog.TimestampFieldName = logger.GCPFieldKeyTime
	zerolog.MessageFieldName = logger.GCPFieldKeyMsg

	// Add the severity hook for GCP logging
	var gcpHook logger.SeverityHook
	log.Logger = zerolog.New(os.Stdout).Hook(gcpHook).With().Timestamp().Logger()
}

func New(conf config.WebConfig) (s *Server, err error) {
	if err = conf.Validate(); err != nil {
		return nil, err
	}

	s = &Server{
		conf: conf,
	}

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
func Debug(conf config.WebConfig, srv *http.Server) (s *Server, err error) {
	if s, err = New(conf); err != nil {
		return nil, err
	}

	// Replace the http server with the one specified
	s.srv = nil
	s.srv = srv
	s.srv.Handler = s.router
	return s, nil
}
