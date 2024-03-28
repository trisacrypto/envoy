package trisa

import (
	"net"
	"self-hosted-node/pkg/config"
	"self-hosted-node/pkg/trisa/network"

	"github.com/rs/zerolog/log"
	api "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
	"github.com/trisacrypto/trisa/pkg/trisa/mtls"
	"github.com/trisacrypto/trisa/pkg/trust"
	"google.golang.org/grpc"
)

// The TRISA server implements the TRISANetwork and TRISAHealth services defined by the
// TRISA protocol buffers in the github.com/trisacrypto/trisa repository. It can be run
// as a standalone service or can be embedded as a component in a larger service.
type Server struct {
	api.UnimplementedTRISAHealthServer
	api.UnimplementedTRISANetworkServer
	srv      *grpc.Server
	conf     config.TRISAConfig
	identity *trust.Provider
	certPool trust.ProviderPool
	network  network.Network
	echan    chan error
}

// Create a new TRISA server ready to handle gRPC requests.
func New(conf config.TRISAConfig, network network.Network) (s *Server, err error) {
	s = &Server{
		conf:    conf,
		network: network,
		echan:   make(chan error),
	}

	// Load the TRISA identity certificates and trust pool
	// TODO: load from the database if available
	if s.identity, err = conf.LoadCerts(); err != nil {
		return nil, err
	}

	if s.certPool, err = conf.LoadPool(); err != nil {
		return nil, err
	}

	// Create TRISA mTLS configuration
	var mtlsCreds grpc.ServerOption
	if mtlsCreds, err = mtls.ServerCreds(s.identity, s.certPool); err != nil {
		return nil, err
	}

	// Configure the gRPC server
	opts := make([]grpc.ServerOption, 0, 3)
	opts = append(opts, mtlsCreds)
	opts = append(opts, s.UnaryInterceptors())
	opts = append(opts, s.StreamInterceptors())

	// Create the gRPC server and register TRISA services
	s.srv = grpc.NewServer(opts...)
	api.RegisterTRISAHealthServer(s.srv, s)
	api.RegisterTRISANetworkServer(s.srv, s)
	return s, nil
}

func (s *Server) Serve() (err error) {
	var sock net.Listener
	if sock, err = net.Listen("tcp", s.conf.BindAddr); err != nil {
		return err
	}

	go s.Run(sock)
	log.Info().Str("addr", s.conf.BindAddr).Bool("maintenance", s.conf.Maintenance).Msg("TRISA server started")
	return nil
}

func (s *Server) Run(sock net.Listener) {
	defer sock.Close()
	if err := s.srv.Serve(sock); err != nil {
		s.echan <- err
	}
}

func (s *Server) Shutdown() error {
	log.Trace().Msg("gracefully shutting down the TRISA server")
	s.srv.GracefulStop()
	log.Debug().Msg("TRISA server stopped")
	return nil
}
