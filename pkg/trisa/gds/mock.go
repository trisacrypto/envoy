package gds

import (
	"errors"

	"github.com/trisacrypto/envoy/pkg/bufconn"
	"github.com/trisacrypto/envoy/pkg/config"
	"github.com/trisacrypto/envoy/pkg/trisa/gds/mock"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// MockGDS implements all of the functionality of the GDS and implements the Directory
// interface because a GDS is embedded. However, instead of connecting to a live
// directory or requiring a TRISA config, it connects to a mock GDS server via bufconn
// to allow robust testing outside of the package.
//
// NOTE: the internal package tests test the GDS object directly, the MockGDS object
// should only be be used for tests outside of the package.
type MockGDS struct {
	GDS
	bufnet *bufconn.Listener
	mock   *mock.GDS
}

// Ensure that the MockGDS implements the Directory interface.
var _ Directory = &MockGDS{}

func NewMock(conf config.TRISAConfig) *MockGDS {
	gds := &MockGDS{
		GDS:    GDS{conf: conf},
		bufnet: bufconn.New(),
	}

	gds.mock = mock.New(gds.bufnet)
	return gds
}

func (m *MockGDS) Connect(opts ...grpc.DialOption) (err error) {
	return m.GDS.Connect(grpc.WithContextDialer(m.bufnet.Dialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
}

func (m *MockGDS) Close() (err error) {
	// Close the client connection to the universal mock
	if cerr := m.GDS.Close(); cerr != nil {
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

func (m *MockGDS) GetMock() *mock.GDS {
	return m.mock
}

func (m *MockGDS) GetBufnet() *bufconn.Listener {
	return m.bufnet
}
