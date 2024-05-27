package peers

import (
	"context"
	"fmt"
	"sync"

	api "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
	"google.golang.org/grpc"
)

// Peer objects should wrap (or mock) the TRISANetworkClient and TRISAHealthClient
// interfaces from the TRISA protocol buffers. Additionally, the Peer should provide
// an Info() method to return its identifying details as fetched from the directory
// service and a Name() method that returns the common name of the peer. A Peer is the
// primary mechanism for network interaction between TRISA nodes. A TRISA node should be
// able to identify a peer from an incoming request via the mTLS certificates used for
// authentication, or to select a remote Peer via a directory service lookup to make
// outgoing transfers. To facilitate remote connections, a Peer should be able to Connect
// using gRPC dial options. Mock peers can implement a bufconn connection, whereas
// actual peer-to-peer connections require mTLS credentials for connecting.
type Peer interface {
	api.TRISANetworkClient
	api.TRISAHealthClient
	fmt.Stringer
	Name() string
	Info() (*Info, error)
	Connect(opts ...grpc.DialOption) error
	Close() error
}

// New creates a new peer object ready to connect to the remote peer. This method is
// primarily used by the directory to instantiate peers to return to the user. An error
// is returned if the infor object does not have a common name or endpoint.
func New(info *Info) (Peer, error) {
	if err := info.Validate(); err != nil {
		return nil, err
	}
	return &TRISAPeer{info: *info}, nil
}

// TRISAPeer implements the Peer interface and is used to directly connect to a remote
// TRISA node by wrapping a gRPC dial connection to the endpoint.
type TRISAPeer struct {
	sync.RWMutex
	info   Info
	conn   *grpc.ClientConn
	client api.TRISANetworkClient
	health api.TRISAHealthClient
}

// Ensure TRISAPeer implements the Peer interface
var _ Peer = &TRISAPeer{}

// Info returns a copy of the peer counterparty information.
func (p *TRISAPeer) Info() (*Info, error) {
	if err := p.info.Validate(); err != nil {
		return nil, err
	}

	// Return a copy of the peer info so that callers cannot modify a peer's internal state
	info := p.info
	return &info, nil
}

// Name returns the common name of the peer.
func (p *TRISAPeer) Name() string {
	if err := p.info.Validate(); err != nil {
		return ""
	}
	return p.info.CommonName
}

// String returns the common name of the peer and can be used for hashing purposes.
func (p *TRISAPeer) String() string {
	return p.Name()
}

// Connect to the remote peer by dialing the grpc endpoint with the specified options
// and credentials. Connect validates that the peer is correctly configured and must be
// called before any gRPC methods can be called on the peer.
func (p *TRISAPeer) Connect(opts ...grpc.DialOption) (err error) {
	p.Lock()
	defer p.Unlock()
	if p.conn != nil {
		return ErrAlreadyConnected
	}

	if err = p.info.Validate(); err != nil {
		return err
	}

	if p.conn, err = grpc.NewClient(p.info.Endpoint, opts...); err != nil {
		return err
	}

	p.client = api.NewTRISANetworkClient(p.conn)
	p.health = api.NewTRISAHealthClient(p.conn)
	return nil
}

// Close the connection to the remote peer. Once called the peer must be reconnected
// before any gRPC methods can be called on the peer.
func (p *TRISAPeer) Close() (err error) {
	p.Lock()
	defer p.Unlock()
	if p.conn != nil {
		err = p.conn.Close()
	}

	p.client = nil
	p.health = nil
	p.conn = nil
	return err
}

func (p *TRISAPeer) Transfer(ctx context.Context, in *api.SecureEnvelope, opts ...grpc.CallOption) (*api.SecureEnvelope, error) {
	p.RLock()
	defer p.RUnlock()
	if p.client == nil {
		return nil, ErrNotConnected
	}
	return p.client.Transfer(ctx, in, opts...)
}

func (p *TRISAPeer) TransferStream(ctx context.Context, opts ...grpc.CallOption) (api.TRISANetwork_TransferStreamClient, error) {
	p.RLock()
	defer p.RUnlock()
	if p.client == nil {
		return nil, ErrNotConnected
	}
	return p.client.TransferStream(ctx, opts...)
}

func (p *TRISAPeer) KeyExchange(ctx context.Context, in *api.SigningKey, opts ...grpc.CallOption) (*api.SigningKey, error) {
	p.RLock()
	defer p.RUnlock()
	if p.client == nil {
		return nil, ErrNotConnected
	}
	return p.client.KeyExchange(ctx, in, opts...)
}

func (p *TRISAPeer) ConfirmAddress(ctx context.Context, in *api.Address, opts ...grpc.CallOption) (*api.AddressConfirmation, error) {
	p.RLock()
	defer p.RUnlock()
	if p.client == nil {
		return nil, ErrNotConnected
	}
	return p.client.ConfirmAddress(ctx, in, opts...)
}

// Status performs a health check against the remote TRISA node.
func (p *TRISAPeer) Status(ctx context.Context, in *api.HealthCheck, opts ...grpc.CallOption) (*api.ServiceState, error) {
	p.RLock()
	defer p.RUnlock()
	if p.health == nil {
		return nil, ErrNotConnected
	}
	return p.health.Status(ctx, in, opts...)
}
