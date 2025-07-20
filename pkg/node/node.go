package node

import (
	"errors"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/trisacrypto/envoy/pkg/audit"
	"github.com/trisacrypto/envoy/pkg/config"
	"github.com/trisacrypto/envoy/pkg/directory"
	"github.com/trisacrypto/envoy/pkg/emails"
	"github.com/trisacrypto/envoy/pkg/logger"
	"github.com/trisacrypto/envoy/pkg/metrics"
	"github.com/trisacrypto/envoy/pkg/store"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/envoy/pkg/store/sqlite"
	"github.com/trisacrypto/envoy/pkg/trisa"
	"github.com/trisacrypto/envoy/pkg/trisa/keychain"
	"github.com/trisacrypto/envoy/pkg/trisa/network"
	"github.com/trisacrypto/envoy/pkg/trp"
	"github.com/trisacrypto/envoy/pkg/web"
	"github.com/trisacrypto/envoy/pkg/webhook"

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

	// Register the prometheus metrics
	if err = metrics.Setup(); err != nil {
		return nil, err
	}

	// Create the node and start to register its internal servers
	node = &Node{
		conf: conf,
		errc: make(chan error, 1),
	}

	// Connect to the database store
	if node.store, err = store.Open(conf.DatabaseURL); err != nil {
		return nil, err
	}

	// Set the search threshold for fuzzy searching
	sqlite.SetThreshold(conf.SearchThreshold)

	// Configure the store with travel address generators
	if factory, err := models.NewTravelAddressFactory(conf.Node.Endpoint, "trisa"); err != nil {
		log.Warn().Err(err).Msg("could not configure travel address factory")
	} else {
		node.store.UseTravelAddressFactory(factory)
	}

	// Configure the webhook if it's enabled
	if conf.Webhook.Enabled() {
		if node.webhook, err = webhook.New(conf.Webhook); err != nil {
			return nil, err
		}
	}

	// Configure email if it's available
	if err = emails.Configure(conf.Email); err != nil {
		return nil, err
	}

	// Create the TRISA management system
	if node.network, err = network.New(conf.Node); err != nil {
		return nil, err
	}
	log.Debug().
		Str("endpoint", conf.Node.Endpoint).
		Str("gds", conf.Node.Directory.Endpoint).
		Str("members", conf.Node.Directory.MembersEndpoint).
		Msg("trisa initialized")

	// Add the node's keychain.KeyChain (created in the network) to the audit package
	// for ComlianceAuditLog signatures and verification. NOTE: ComplianceAuditLogs
	// are currently required, so it is required that we have a keychain.KeyChain
	// at this step.
	var kc keychain.KeyChain
	if kc, err = node.network.KeyChain(); err != nil {
		return nil, err
	}
	if kc == nil {
		return nil, errors.New("keychain must be configured for audit logging")
	}
	audit.UseKeyChain(&kc)

	// Create the admin web ui server if it is enabled
	if node.admin, err = web.New(conf, node.store, node.network); err != nil {
		return nil, err
	}

	// Create the TRISA API server
	if node.trisa, err = trisa.New(conf.Node, node.network, node.store, node.webhook, node.errc); err != nil {
		return nil, err
	}

	// Create the TRP server
	if node.trp, err = trp.New(conf, node.store, node.network); err != nil {
		return nil, err
	}

	// Create the directory sync background routine
	if node.syncd, err = directory.New(conf.DirectorySync, node.network, node.store, node.errc); err != nil {
		return nil, err
	}

	return node, nil
}

// Node implements the complete TRISA Self Hosted Node including the TRISA gRPC server,
// the TRP API server, the web compliance and admin user interface, and the internal API
// server, along with kubernetes probes and metrics if required.
type Node struct {
	conf    config.Config
	admin   *web.Server
	trisa   *trisa.Server
	trp     *trp.Server
	syncd   *directory.Sync
	store   store.Store
	network network.Network
	webhook webhook.Handler
	errc    chan error
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

	// Run services that should not be run in maintenance mode
	if !s.conf.Maintenance {
		// Run the directory sync service
		if err = s.syncd.Run(); err != nil {
			return err
		}
	}

	// Start the web ui server if it is enabled
	if err = s.admin.Serve(s.errc); err != nil {
		return err
	}

	// Start the TRISA node server if it is enabled
	if err = s.trisa.Serve(); err != nil {
		return err
	}

	// Start the TRP server if it is enabled
	if err = s.trp.Serve(s.errc); err != nil {
		return err
	}

	// Block until an error occurs or shutdown happens
	log.Info().Msg("envoy node has started")
	if err := <-s.errc; err != nil {
		log.WithLevel(zerolog.FatalLevel).Err(err).Msg("envoy node has crashed")
		return err
	}
	return nil
}

func (s *Node) Shutdown() (err error) {
	log.Info().Msg("gracefully shutting down envoy node services")

	// Stop services that only run when not in maintenance mode
	if !s.conf.Maintenance {
		if serr := s.syncd.Stop(); serr != nil {
			err = errors.Join(err, serr)
		}
	}

	// Shutdown web ui server if it is enabled.
	if serr := s.admin.Shutdown(); serr != nil {
		err = errors.Join(err, serr)
	}

	// Shutdown the TRISA server
	if terr := s.trisa.Shutdown(); terr != nil {
		err = errors.Join(err, terr)
	}

	// Shutdown the TRP server
	if terr := s.trp.Shutdown(); terr != nil {
		err = errors.Join(err, terr)
	}

	log.Debug().Msg("envoy node has shutdown")
	return err
}
