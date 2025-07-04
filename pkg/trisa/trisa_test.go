package trisa_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/trisacrypto/envoy/pkg/bufconn"
	"github.com/trisacrypto/envoy/pkg/config"
	"github.com/trisacrypto/envoy/pkg/logger"
	"github.com/trisacrypto/envoy/pkg/store"
	"github.com/trisacrypto/envoy/pkg/trisa"
	directory "github.com/trisacrypto/envoy/pkg/trisa/gds"
	gdsmock "github.com/trisacrypto/envoy/pkg/trisa/gds/mock"
	"github.com/trisacrypto/envoy/pkg/trisa/network"

	"github.com/stretchr/testify/suite"
	api "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
	"github.com/trisacrypto/trisa/pkg/trisa/mtls"
	"github.com/trisacrypto/trisa/pkg/trust"
	"google.golang.org/grpc"
	"google.golang.org/grpc/resolver"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func init() {
	// Required for buffconn testing.
	resolver.SetDefaultScheme("passthrough")
}

type trisaTestSuite struct {
	suite.Suite
	conf    config.TRISAConfig
	conn    *bufconn.Listener
	svc     *trisa.Server
	network network.Network
	client  api.TRISANetworkClient
	store   store.Store
	echan   chan error
}

func (s *trisaTestSuite) SetupSuite() {
	require := s.Require()
	logger.Discard()

	s.conn = bufconn.New()
	s.echan = make(chan error, 1)
	s.conf = config.TRISAConfig{
		Maintenance: false,
		Enabled:     true,
		BindAddr:    "bufnet",
		MTLSConfig: config.MTLSConfig{
			Certs: "testdata/certs/alice.vaspbot.com.pem",
			Pool:  "testdata/certs/trisatest.dev.pem",
		},
		KeyExchangeCacheTTL: 60 * time.Second,
		Directory: config.DirectoryConfig{
			Insecure:        true,
			Endpoint:        bufconn.Endpoint,
			MembersEndpoint: bufconn.Endpoint,
		},
	}

	var err error
	s.network, err = network.NewMocked(&s.conf)
	require.NoError(err, "could not create TRISA network manager")

	s.store, err = store.Open("mock:///")
	require.NoError(err, "could not create mock store")

	s.svc, err = trisa.New(s.conf, s.network, s.store, nil, s.echan)
	require.NoError(err, "could not create a new TRISA server")

	// Run the TRISA server on the bufconn for tests
	go s.svc.Run(s.conn.Sock())

	// Load client credentials
	creds, err := loadClientCredentials(bufconn.Endpoint, "testdata/certs/client.trisatest.dev.pem")
	require.NoError(err, "could not load client credentials")

	// Create the network client for testing (health client created seperately)
	var cc *grpc.ClientConn
	cc, err = s.conn.Connect(context.Background(), creds)
	require.NoError(err, "could not connect to bufconn client")

	s.client = api.NewTRISANetworkClient(cc)
}

func (s *trisaTestSuite) TearDownSuite() {
	require := s.Require()
	logger.ResetLogger()

	require.NoError(s.svc.Shutdown(), "could not shutdown the TRISA server")
	require.NoError(s.conn.Close(), "could not close the bufconn")
}

func (s *trisaTestSuite) BeforeTest(_, _ string) {
	require := s.Require()

	// Ensure the client is part of the directory service mock
	gds, err := s.network.Directory()
	require.NoError(err, "could not get mock directory from network")

	err = gds.(*directory.MockGDS).GetMock().UseFixture(gdsmock.LookupRPC, "testdata/gds/lookup.json")
	require.NoError(err, "could not configure directory mock to identify client")

}

func TestTRISA(t *testing.T) {
	suite.Run(t, new(trisaTestSuite))
}

func loadClientCredentials(endpoint, path string) (_ grpc.DialOption, err error) {
	var sz *trust.Serializer
	if sz, err = trust.NewSerializer(false); err != nil {
		return nil, err
	}

	var certs *trust.Provider
	if certs, err = sz.ReadFile(path); err != nil {
		return nil, fmt.Errorf("could not parse certs: %w", err)
	}

	var pool trust.ProviderPool
	if pool, err = sz.ReadPoolFile(path); err != nil {
		return nil, fmt.Errorf("could not parse cert pool %w", err)
	}

	return mtls.ClientCreds(endpoint, certs, pool)
}

func loadFixture(path string, obj proto.Message) (err error) {
	var data []byte
	if data, err = os.ReadFile(path); err != nil {
		return err
	}

	json := protojson.UnmarshalOptions{
		AllowPartial:   true,
		DiscardUnknown: true,
	}

	return json.Unmarshal(data, obj)
}

func loadSecureEnvelope(path string) (env *api.SecureEnvelope, err error) {
	env = &api.SecureEnvelope{}
	if err = loadFixture(path, env); err != nil {
		return nil, err
	}
	return env, nil
}
