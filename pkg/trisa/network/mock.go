package network

import (
	"time"

	"github.com/trisacrypto/envoy/pkg/bufconn"
	"github.com/trisacrypto/envoy/pkg/config"
	directory "github.com/trisacrypto/envoy/pkg/trisa/gds"
	"github.com/trisacrypto/envoy/pkg/trisa/keychain"
	"github.com/trisacrypto/envoy/pkg/trisa/peers"

	"github.com/trisacrypto/trisa/pkg/trisa/keys"
	"github.com/trisacrypto/trisa/pkg/trust"
)

// NewMocked returns a mocked network that is suitable both for testing network
// functionality as well as testing external packages that depend on the Network, e.g.
// this isn't a mock network object but rather a mocked network object. The underlying
// mocking is as follows: a test TRISA config is created, the internal directory is
// replaced with a MockGDS object and the PeerManager methods return a MockPeer object
// because the PeerConstructor method is replaced with peers.MockPeer. The KeyChain
// object and the PeerDialer using mTLS are not mocked and all TRISANetwork functions
// should be functional using the mocked network.
func NewMocked(conf *config.TRISAConfig) (_ Network, err error) {
	if conf == nil {
		conf = &config.TRISAConfig{
			MTLSConfig: config.MTLSConfig{
				Pool:  "testdata/pool.pem",
				Certs: "testdata/alice.pem",
			},
			KeyExchangeCacheTTL: 1 * time.Second,
			Directory: config.DirectoryConfig{
				Insecure:        true,
				Endpoint:        bufconn.Endpoint,
				MembersEndpoint: bufconn.Endpoint,
			},
		}
	}

	// Create a basic TRISANetwork object with a testing config and a mock peer constructor.
	network := &TRISANetwork{
		conf:        *conf,
		peers:       make(map[string]peers.Peer),
		constructor: peers.NewMock,
	}

	// Connect the network to a mock directory service.
	network.directory = directory.NewMock(network.conf)
	if err = network.directory.Connect(); err != nil {
		return nil, err
	}

	// Instantiate a real TRISA dialer to ensure that the certs are correctly loaded.
	// NOTE: the certs are ignored in MockPeer connect but the dialer can be used to
	// test creating an mTLS connection to a bufconn server, e.g. to test FromContext.
	if network.dialer, err = TRISADialer(network.conf); err != nil {
		return nil, err
	}

	// Using a regular KeyChain provider with an in-memory store for testing.
	var provider *trust.Provider
	if provider, err = network.conf.LoadCerts(); err != nil {
		return nil, err
	}

	var localKey keys.Key
	if localKey, err = keys.FromProvider(provider); err != nil {
		return nil, err
	}

	if network.keyChain, err = keychain.New(keychain.WithDefaultKey(localKey), keychain.WithCacheDuration(network.conf.KeyExchangeCacheTTL)); err != nil {
		return nil, err
	}
	return network, nil
}
