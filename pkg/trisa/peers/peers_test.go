package peers_test

import (
	"context"
	"crypto/x509"
	"testing"

	"github.com/trisacrypto/envoy/pkg/bufconn"
	"github.com/trisacrypto/envoy/pkg/trisa/peers"
	"github.com/trisacrypto/envoy/pkg/trisa/peers/mock"

	"github.com/stretchr/testify/require"
	api "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/resolver"
)

func init() {
	// Required for buffconn testing.
	resolver.SetDefaultScheme("passthrough")
}

func TestPeers(t *testing.T) {
	_, err := peers.New(&peers.Info{})
	require.Error(t, err, "should not be able to create a peer with invalid info")

	info := &peers.Info{CommonName: "trisa.example.com", Endpoint: "trisa.example.com:443"}
	peer, err := peers.New(info)
	require.NoError(t, err, "should be able to create a peer with valid info")

	require.Equal(t, "trisa.example.com", peer.Name(), "expected peer name to be the common name")
	require.Equal(t, "trisa.example.com", peer.String(), "expected peer string to be the common name")

	// Ensure a copy of the info is made
	pinfo, err := peer.Info()
	require.NoError(t, err, "was not able to fetch peer info")
	require.NotSame(t, info, pinfo, "a copy of info should have been made")

	require.NoError(t, peer.Close(), "should be able to close an unconnected peer without error")

	// Should be able to connect to a bufconn
	bufnet := bufconn.New()
	defer bufnet.Close()

	// Should error if we try to connect without transport security
	err = peer.Connect(grpc.WithContextDialer(bufnet.Dialer))
	require.Error(t, err, "errors from the dialer should be passed up to caller (transport security required)")

	// Should be able to successfully connect
	err = peer.Connect(grpc.WithContextDialer(bufnet.Dialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err, "could not connect peer via bufconn")

	// Should not be able to connect to an already connected peer
	err = peer.Connect(grpc.WithContextDialer(bufnet.Dialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.ErrorIs(t, err, peers.ErrAlreadyConnected, "was able to connect to an already connected peer")

	// Should be able to close a client connection
	err = peer.Close()
	require.NoError(t, err, "could not close connection to peer")
}

func TestPeerTransfer(t *testing.T) {
	peer, err := peers.New(&peers.Info{CommonName: "bufnet", Endpoint: bufconn.Endpoint})
	require.NoError(t, err, "could not create new bufnet peer")

	_, err = peer.Transfer(context.TODO(), &api.SecureEnvelope{})
	require.ErrorIs(t, err, peers.ErrNotConnected, "should not be able to call transfer RPC when not connected")

	// Run a mock remote peer server to test the connection
	mrp := getMockRemote(t)
	err = peer.Connect(grpc.WithContextDialer(mrp.Channel().Dialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err, "could not connect peer via bufconn")

	// Should be able to make transfer request to bufconn remote
	rep, err := peer.Transfer(context.TODO(), &api.SecureEnvelope{})
	require.NoError(t, err, "could not make transfer request to remote peer")
	require.NotNil(t, rep, "received unexpected response from remote peer")
	require.Equal(t, 1, mrp.Calls[mock.TransferRPC], "mock remote peer should have been called")
}

func TestPeerTransferStream(t *testing.T) {
	peer, err := peers.New(&peers.Info{CommonName: "bufnet", Endpoint: bufconn.Endpoint})
	require.NoError(t, err, "could not create new bufnet peer")

	_, err = peer.TransferStream(context.TODO())
	require.ErrorIs(t, err, peers.ErrNotConnected, "should not be able to call transfer stream RPC when not connected")

	// Run a mock remote peer server to test the connection
	mrp := getMockRemote(t)
	err = peer.Connect(grpc.WithContextDialer(mrp.Channel().Dialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err, "could not connect peer via bufconn")

	// Should be able to make transfer stream request to bufconn remote
	stream, err := peer.TransferStream(context.TODO())
	require.NoError(t, err, "could not make transfer stream request to remote peer")
	require.NotNil(t, stream, "received unexpected response from remote peer")

	err = stream.Send(&api.SecureEnvelope{})
	require.NoError(t, err, "could not send message on stream")

	err = stream.CloseSend()
	require.NoError(t, err, "could not close stream")
}

func TestPeerKeyExchange(t *testing.T) {
	peer, err := peers.New(&peers.Info{CommonName: "bufnet", Endpoint: bufconn.Endpoint})
	require.NoError(t, err, "could not create new bufnet peer")

	_, err = peer.KeyExchange(context.TODO(), &api.SigningKey{})
	require.ErrorIs(t, err, peers.ErrNotConnected, "should not be able to call key exchange RPC when not connected")

	// Run a mock remote peer server to test the connection
	mrp := getMockRemote(t)
	err = peer.Connect(grpc.WithContextDialer(mrp.Channel().Dialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err, "could not connect peer via bufconn")

	// Should be able to make key exchange request to bufconn remote
	rep, err := peer.KeyExchange(context.TODO(), &api.SigningKey{})
	require.NoError(t, err, "could not make key exchange request to remote peer")
	require.NotNil(t, rep, "received unexpected response from remote peer")
	require.Equal(t, 1, mrp.Calls[mock.KeyExchangeRPC], "mock remote peer should have been called")
}

func TestPeerConfirmAddress(t *testing.T) {
	peer, err := peers.New(&peers.Info{CommonName: "bufnet", Endpoint: bufconn.Endpoint})
	require.NoError(t, err, "could not create new bufnet peer")

	_, err = peer.ConfirmAddress(context.TODO(), &api.Address{})
	require.ErrorIs(t, err, peers.ErrNotConnected, "should not be able to call transfer RPC when not connected")

	// Run a mock remote peer server to test the connection
	mrp := getMockRemote(t)
	err = peer.Connect(grpc.WithContextDialer(mrp.Channel().Dialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err, "could not connect peer via bufconn")

	// Should be able to make confirm address request to bufconn remote
	rep, err := peer.ConfirmAddress(context.TODO(), &api.Address{})
	require.NoError(t, err, "could not make confirm address request to remote peer")
	require.NotNil(t, rep, "received unexpected response from remote peer")
	require.Equal(t, 1, mrp.Calls[mock.ConfirmAddressRPC], "mock remote peer should have been called")
}

func TestPeerStatus(t *testing.T) {
	peer, err := peers.New(&peers.Info{CommonName: "bufnet", Endpoint: bufconn.Endpoint})
	require.NoError(t, err, "could not create new bufnet peer")

	_, err = peer.Status(context.TODO(), &api.HealthCheck{})
	require.ErrorIs(t, err, peers.ErrNotConnected, "should not be able to call status RPC when not connected")

	// Run a mock remote peer server to test the connection
	mrp := getMockRemote(t)
	err = peer.Connect(grpc.WithContextDialer(mrp.Channel().Dialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err, "could not connect peer via bufconn")

	// Should be able to make status request to bufconn remote
	rep, err := peer.Status(context.TODO(), &api.HealthCheck{})
	require.NoError(t, err, "could not make status request to remote peer")
	require.NotNil(t, rep, "received unexpected response from remote peer")
	require.Equal(t, 1, mrp.Calls[mock.StatusRPC], "mock remote peer should have been called")
}

func getMockRemote(t *testing.T) *mock.RemotePeer {
	// Create mock remote peer and shut it down when the tests clean up
	mrp := mock.New(nil)
	t.Cleanup(mrp.Shutdown)

	// Mock the RPC responses
	mrp.OnTransfer = func(ctx context.Context, in *api.SecureEnvelope) (*api.SecureEnvelope, error) {
		return &api.SecureEnvelope{
			Id: "0e55a3b9-0744-4db4-afd4-7bb2523cba42",
			Error: &api.Error{
				Code:    api.Error_BENEFICIARY_NAME_UNMATCHED,
				Message: "could not find specified beneficiary",
			},
			Timestamp:          "2022-04-16T14:19:44-05:00",
			Sealed:             false,
			PublicKeySignature: "",
		}, nil
	}

	mrp.OnTransferStream = func(stream api.TRISANetwork_TransferStreamServer) error {
		for {
			if _, err := stream.Recv(); err != nil {
				return err
			}

			msg := &api.SecureEnvelope{
				Id: "0e55a3b9-0744-4db4-afd4-7bb2523cba42",
				Error: &api.Error{
					Code:    api.Error_BENEFICIARY_NAME_UNMATCHED,
					Message: "could not find specified beneficiary",
				},
				Timestamp:          "2022-04-16T14:19:44-05:00",
				Sealed:             false,
				PublicKeySignature: "",
			}

			if err := stream.Send(msg); err != nil {
				return err
			}
		}
	}

	mrp.OnKeyExchange = func(ctx context.Context, in *api.SigningKey) (*api.SigningKey, error) {
		return &api.SigningKey{
			Version:            3,
			Signature:          []byte("signature"),
			SignatureAlgorithm: x509.SHA256WithRSA.String(),
			PublicKeyAlgorithm: x509.RSA.String(),
			NotBefore:          "2022-04-16T14:15:00Z",
			NotAfter:           "2023-04-17T22:15:00Z",
			Revoked:            false,
			Data:               []byte("key data"),
		}, nil
	}

	mrp.OnConfirmAddress = func(ctx context.Context, in *api.Address) (*api.AddressConfirmation, error) {
		return &api.AddressConfirmation{}, nil
	}

	mrp.OnStatus = func(ctx context.Context, in *api.HealthCheck) (*api.ServiceState, error) {
		return &api.ServiceState{
			Status:    api.ServiceState_HEALTHY,
			NotBefore: "2022-04-16T14:15:00Z",
			NotAfter:  "2022-04-16T22:15:00Z",
		}, nil
	}

	return mrp
}
