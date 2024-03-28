package network

import (
	"self-hosted-node/pkg/bufconn"
	"self-hosted-node/pkg/config"
	"self-hosted-node/pkg/trisa/peers"

	"github.com/trisacrypto/trisa/pkg/trisa/mtls"
	"github.com/trisacrypto/trisa/pkg/trust"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// PeerConstructor is used to determine what type of Peer to make. In normal operations
// a TRISAPeer is created to connect over a real grpc connection to a remote server. In
// tests, a MockPeer is created to connect to a RemotePeer universal mock object via a
// bufconn to ensure that the network is correctly creating peers.
type PeerConstructor func(info *peers.Info) (peers.Peer, error)

// PeerDialer is used by networks to specify how to connect the Peer object. In normal
// operation the PeerDialer establishes an mTLS connection using the configuration on
// the network. For tests, the PeerDialer connects to a RemotePeer universal mock object
// via a bufconn to ensure the network code is correctly being called.
type PeerDialer func(endpoint string) ([]grpc.DialOption, error)

// TRISADialer returns a closure that is able to dial arbitrary endpoints using mTLS
// authentication loaded via the certs and pool in the TRISA config. Using a factory
// method to create the dialer allows us to mock the dialer for testing purposes.
// NOTE: if the certs change during runtime, the dialer will have to be recreated since
// the mTLS authority stays on the stack of the closure and is not accessible elsewhere.
func TRISADialer(conf config.TRISAConfig) (_ PeerDialer, err error) {
	var certs *trust.Provider
	if certs, err = conf.LoadCerts(); err != nil {
		return nil, err
	}

	var pool trust.ProviderPool
	if pool, err = conf.LoadPool(); err != nil {
		return nil, err
	}

	return func(endpoint string) (opts []grpc.DialOption, err error) {
		opts = make([]grpc.DialOption, 0, 1)

		var creds grpc.DialOption
		if creds, err = mtls.ClientCreds(endpoint, certs, pool); err != nil {
			return nil, err
		}
		opts = append(opts, creds)

		return opts, nil
	}, nil
}

// BufnetDialer returns a closure that is able to connect the peer to via the bufconn
// socket. This method is currently unused and is kept for documentation purposes.
func BufnetDialer(bufnet *bufconn.Listener) (_ PeerDialer, err error) {
	return func(endpoint string) (opts []grpc.DialOption, err error) {
		opts = make([]grpc.DialOption, 0, 2)
		opts = append(opts, grpc.WithContextDialer(bufnet.Dialer))
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
		return opts, nil
	}, nil
}
