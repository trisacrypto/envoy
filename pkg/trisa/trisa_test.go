package trisa_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"self-hosted-node/pkg/bufconn"
	"self-hosted-node/pkg/config"
	"self-hosted-node/pkg/logger"
	"self-hosted-node/pkg/trisa"
	directory "self-hosted-node/pkg/trisa/gds"
	gdsmock "self-hosted-node/pkg/trisa/gds/mock"
	"self-hosted-node/pkg/trisa/network"

	"github.com/stretchr/testify/suite"
	api "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
	"github.com/trisacrypto/trisa/pkg/trisa/mtls"
	"github.com/trisacrypto/trisa/pkg/trust"
	"google.golang.org/grpc"
)

type trisaTestSuite struct {
	suite.Suite
	conf    config.TRISAConfig
	conn    *bufconn.Listener
	svc     *trisa.Server
	network network.Network
	client  api.TRISANetworkClient
	echan   chan error
}

func (s *trisaTestSuite) SetupSuite() {
	assert := s.Assert()
	logger.Discard()

	s.conn = bufconn.New()
	s.echan = make(chan error, 1)
	s.conf = config.TRISAConfig{
		Maintenance:         false,
		BindAddr:            "bufnet",
		Certs:               "testdata/certs/alice.vaspbot.net.pem",
		Pool:                "testdata/certs/trisatest.dev.pem",
		KeyExchangeCacheTTL: 60 * time.Second,
		Directory: config.DirectoryConfig{
			Insecure:        true,
			Endpoint:        "bufnet",
			MembersEndpoint: "bufnet",
		},
	}

	var err error
	s.network, err = network.NewMocked(&s.conf)
	assert.NoError(err, "could not create TRISA network manager")

	s.svc, err = trisa.New(s.conf, s.network, s.echan)
	assert.NoError(err, "could not create a new TRISA server")

	// Run the TRISA server on the bufconn for tests
	go s.svc.Run(s.conn.Sock())

	// Load client credentials
	creds, err := loadClientCredentials("bufnet", "testdata/certs/client.trisatest.dev.pem")
	assert.NoError(err, "could not load client credentials")

	// Create the network client for testing (health client created seperately)
	var cc *grpc.ClientConn
	cc, err = s.conn.Connect(context.Background(), creds)
	assert.NoError(err, "could not connect to bufconn client")

	s.client = api.NewTRISANetworkClient(cc)
}

func (s *trisaTestSuite) TearDownSuite() {
	assert := s.Assert()
	logger.ResetLogger()

	assert.NoError(s.svc.Shutdown(), "could not shutdown the TRISA server")
	assert.NoError(s.conn.Close(), "could not close the bufconn")
}

func (s *trisaTestSuite) BeforeTest(_, _ string) {
	assert := s.Assert()

	// Ensure the client is part of the directory service mock
	gds, err := s.network.Directory()
	assert.NoError(err, "could not get mock directory from network")

	err = gds.(*directory.MockGDS).GetMock().UseFixture(gdsmock.LookupRPC, "testdata/gds/lookup.json")
	assert.NoError(err, "could not configure directory mock to identify client")

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
