package node

import (
	"errors"
	"os"
	"os/signal"
	"syscall"
	"time"

	"self-hosted-node/pkg/config"
	"self-hosted-node/pkg/logger"
	"self-hosted-node/pkg/store"
	"self-hosted-node/pkg/web"

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

// Create a new TRISA node from the global configuration, ready to serve.
func New(conf config.Config) (node *Node, err error) {
	// Load the default configuration from the environment if config is empty.
	if conf.IsZero() {
		if conf, err = config.New(); err != nil {
			return nil, err
		}
	}

	// Setup our logging config first thing
	zerolog.SetGlobalLevel(conf.GetLogLevel())
	if conf.ConsoleLog {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	// Set the gin mode for all gin servers
	gin.SetMode(conf.Mode)

	// Create the node and start to register its internal servers
	node = &Node{
		conf: conf,
		errc: make(chan error, 1),
	}

	// Connect to the database store
	if node.store, err = store.Open(conf.DatabaseURL); err != nil {
		return nil, err
	}

	// Create the web ui server if it is enabled
	if node.web, err = web.New(conf.Web, node.store); err != nil {
		return nil, err
	}

	return node, nil
}

// Node implements the complete TRISA Self Hosted Node including the TRISA gRPC server,
// the TRP API server, the web compliance and admin user interface, and the internal API
// server, along with kubernetes probes and metrics if required.
type Node struct {
	conf  config.Config
	web   *web.Server
	store store.Store
	errc  chan error
}

// Serve all enabled services based on configuration and block until shutdown or until
// an OS signal or error causes the server to shutdown.
func (s *Node) Serve() (err error) {
	// Handle OS Signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-quit
		s.errc <- s.Shutdown()
	}()

	// TODO: handle maintenance mode setup tasks

	// Start the web ui server if it is enabled
	if err = s.web.Serve(s.errc); err != nil {
		return err
	}

	// Block until an error occurs or shutdown happens
	log.Info().Msg("trisa node server has started")
	return <-s.errc
}

func (s *Node) Shutdown() (err error) {
	log.Info().Msg("gracefully shutting down trisa node services")

	// Shutdown web ui server if it is enabled.
	if serr := s.web.Shutdown(); serr != nil {
		err = errors.Join(err, serr)
	}

	log.Debug().Msg("trisa node shutdown")
	return err
}
