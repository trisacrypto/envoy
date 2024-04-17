package peers

import (
	"errors"

	"github.com/trisacrypto/envoy/pkg/bufconn"
	"github.com/trisacrypto/envoy/pkg/trisa/peers/mock"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// MockPeer implements all of the functionality of a TRISAPeer and implements the Peer
// interface because a TRISAPeer is embedded. However, instead of connecting to a live
// remote TRISA node or requiring a TRISA config, it connects to a mock RemotePeer via
// bufconn to allow for robust testing outside of the package.
//
// NOTE: the internal package tests test the TRISAPeer object directly, the MockPeer
// should only be used for tests outside of the package.
type MockPeer struct {
	TRISAPeer
	bufnet *bufconn.Listener
	mock   *mock.RemotePeer
}

// Ensure that the MockPeer implements the Peer interface
var _ Peer = &MockPeer{}

func NewMock(info *Info) (Peer, error) {
	if err := info.Validate(); err != nil {
		return nil, err
	}

	peer := &MockPeer{
		TRISAPeer: TRISAPeer{info: *info},
		bufnet:    bufconn.New(),
	}

	peer.mock = mock.New(peer.bufnet)
	return peer, nil
}

func (m *MockPeer) Connect(opts ...grpc.DialOption) (err error) {
	return m.TRISAPeer.Connect(grpc.WithContextDialer(m.bufnet.Dialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
}

func (m *MockPeer) Close() (err error) {
	// Close the client connection to the universal mock
	if cerr := m.TRISAPeer.Close(); cerr != nil {
		err = errors.Join(err, cerr)
	}

	// Shutdown the universal mock server
	m.mock.Shutdown()

	// Close the bufnet socket
	if cerr := m.bufnet.Close(); cerr != nil {
		err = errors.Join(err, cerr)
	}

	// Cleanup
	m.bufnet = nil
	m.mock = nil
	return err
}

func (m *MockPeer) GetMock() *mock.RemotePeer {
	return m.mock
}

func (m *MockPeer) GetBufnet() *bufconn.Listener {
	return m.bufnet
}
