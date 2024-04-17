package network_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/trisacrypto/envoy/pkg/bufconn"
	"github.com/trisacrypto/envoy/pkg/config"
	directory "github.com/trisacrypto/envoy/pkg/trisa/gds"
	"github.com/trisacrypto/envoy/pkg/trisa/network"

	dmock "github.com/trisacrypto/envoy/pkg/trisa/gds/mock"
	pmock "github.com/trisacrypto/envoy/pkg/trisa/peers/mock"

	"github.com/stretchr/testify/require"
	api "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
	gds "github.com/trisacrypto/trisa/pkg/trisa/gds/api/v1beta1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestFromContext tests the happy case where a correct mTLS connection is established
// and an actual gRPC context exists to parse the mTLS credentials off of. It cannot
// test any of the unhappy and error paths that are checked for in the code base.
func TestFromContext(t *testing.T) {
	trisa, err := network.NewMocked(nil)
	require.NoError(t, err, "could not create mocked trisa network")
	defer trisa.Close()

	// Set up a mock directory service response for from context lookup
	ds, err := trisa.Directory()
	require.NoError(t, err, "could not get directory")
	mgds, ok := ds.(*directory.MockGDS)
	require.True(t, ok, "expected a mocked directory servce")

	// Make assertions about what is being looked up in the GDS
	mgds.GetMock().OnLookup = func(ctx context.Context, in *gds.LookupRequest) (out *gds.LookupReply, err error) {
		// Assert that the expected common name is being looked up
		require.Equal(t, "alice.vaspbot.net", in.CommonName, "unexpected common name in lookup request")
		require.Empty(t, in.Id, "unexpected id in lookup request")
		require.Empty(t, in.RegisteredDirectory, "unexpected registered directory in lookup request")

		return &gds.LookupReply{
			Id:                  "bd6bf155-86a6-41c9-90e5-1d6a4797b160",
			RegisteredDirectory: "trisatest.dev",
			CommonName:          "alice.vaspbot.net",
			Endpoint:            "alice.vaspbot.net:443",
			Name:                "Alice VASP",
			Country:             "US",
			VerifiedOn:          "2024-03-10T08:23:02Z",
		}, nil
	}

	// Create an mTLS connection to test the context over bufconn
	opts, err := trisa.PeerDialer()("trisa.example.com")
	require.NoError(t, err, "could not create mtls dial credentials")
	require.Len(t, opts, 1, "dial options contains unexpected number of things")

	// Create an mTLS RemotePeer gRPC server for testing
	conf := config.TRISAConfig{Certs: "testdata/alice.pem", Pool: "testdata/pool.pem"}
	bufnet := bufconn.New()
	remote, err := pmock.NewAuth(bufnet, conf)
	require.NoError(t, err, "could not create authenticated remote peer mock")

	// Connect a TRISANetwork client to the authenticated mock
	opts = append(opts, grpc.WithContextDialer(bufnet.Dialer))
	cc, err := grpc.Dial("alice.vaspbot.net", opts...)
	require.NoError(t, err, "could not dial authenticated remote peer mock")

	// Setup to get the context from the remote dialer
	client := api.NewTRISANetworkClient(cc)
	remote.OnTransfer = func(ctx context.Context, in *api.SecureEnvelope) (*api.SecureEnvelope, error) {
		// Ok, after all that work above we finally have an actual gRPC context with mTLS info
		peer, err := trisa.FromContext(ctx)
		if err != nil {
			return nil, errors.Join(errors.New("could not lookup peer from context"), err)
		}

		info, err := peer.Info()
		if err != nil {
			return nil, errors.Join(errors.New("could not get peer info"), err)
		}

		if info.CommonName != "alice.vaspbot.net" {
			return nil, fmt.Errorf("unknown common name %q expected alice.vaspbot.net", info.CommonName)
		}

		// Don't return anything
		return &api.SecureEnvelope{}, nil
	}

	// Make the request with the client to finish the tests
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = client.Transfer(ctx, &api.SecureEnvelope{})
	require.NoError(t, err, "could not make transfer to initiate from context tests")
}

func TestLookupPeer(t *testing.T) {
	trisa, err := network.NewMocked(nil)
	require.NoError(t, err, "could not create mocked trisa network")
	defer trisa.Close()

	trisaActual, ok := trisa.(*network.TRISANetwork)
	require.True(t, ok, "trisa should be a TRISANetwork")

	// Set up a mock directory service response for from context lookup
	ds, err := trisa.Directory()
	require.NoError(t, err, "could not get directory")
	mgds, ok := ds.(*directory.MockGDS)
	require.True(t, ok, "expected a mocked directory servce")

	// Test uncached lookup by common name for alpha peer
	err = mgds.GetMock().UseFixture(dmock.LookupRPC, "testdata/gds/alice.json")
	require.NoError(t, err, "could not setup GDS mock with fixture")

	require.Equal(t, 0, trisaActual.NPeers(), "peers cache is not empty")
	peer, err := trisa.LookupPeer(context.TODO(), "alice.vaspbot.net", "")
	require.NoError(t, err, "could not lookup peer")
	require.Equal(t, "alice.vaspbot.net", peer.Name(), "unexpected peer returned")
	require.Equal(t, 1, trisaActual.NPeers(), "peers cache should contain one item")
	require.Equal(t, 1, mgds.GetMock().Calls[dmock.LookupRPC], "unexpected number of GDS lookup calls")

	// Test cached lookup by common name for alpha peer
	peer, err = trisa.LookupPeer(context.TODO(), "alice.vaspbot.net", "")
	require.NoError(t, err, "could not lookup peer")
	require.Equal(t, "alice.vaspbot.net", peer.Name(), "unexpected peer returned")
	require.Equal(t, 1, trisaActual.NPeers(), "peers cache should not have increased on next call")
	require.Equal(t, 1, mgds.GetMock().Calls[dmock.LookupRPC], "cache should have prevented GDS Lookup")

	// Test uncached lookup by vasp ID for bravo peer
	err = mgds.GetMock().UseFixture(dmock.LookupRPC, "testdata/gds/bob.json")
	require.NoError(t, err, "could not setup GDS mock with fixture")

	peer, err = trisa.LookupPeer(context.TODO(), "d5699a22-e1f5-4952-afb0-81c447749e6e", "")
	require.NoError(t, err, "could not lookup peer")
	require.Equal(t, "bob.vaspbot.net", peer.Name(), "unexpected peer returned")
	require.Equal(t, 2, trisaActual.NPeers(), "peers cache should contain two item")
	require.Equal(t, 2, mgds.GetMock().Calls[dmock.LookupRPC], "unexpected number of GDS lookup calls")

	// Test cached lookup by common name for bravo peer
	// NOTE: currently, executing another request by UUID would trigger another GDS lookup
	peer, err = trisa.LookupPeer(context.TODO(), "bob.vaspbot.net", "")
	require.NoError(t, err, "could not lookup peer")
	require.Equal(t, "bob.vaspbot.net", peer.Name(), "unexpected peer returned")
	require.Equal(t, 2, trisaActual.NPeers(), "peers cache should not have increased on next call")
	require.Equal(t, 2, mgds.GetMock().Calls[dmock.LookupRPC], "cache should have prevented GDS Lookup")

	// Check cache at end of testing
	require.True(t, trisaActual.Contains("alice.vaspbot.net"), "missing alpha")
	require.True(t, trisaActual.Contains("bob.vaspbot.net"), "missing bravo")
}

func TestLookupPeerInput(t *testing.T) {
	trisa, err := network.NewMocked(nil)
	require.NoError(t, err, "could not create mocked trisa network")
	defer trisa.Close()

	// Set up a mock directory service response for from context lookup
	ds, err := trisa.Directory()
	require.NoError(t, err, "could not get directory")
	mgds, ok := ds.(*directory.MockGDS)
	require.True(t, ok, "expected a mocked directory servce")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Assert lookups by comon name have correct request
	mgds.GetMock().OnLookup = func(ctx context.Context, in *gds.LookupRequest) (*gds.LookupReply, error) {
		require.Equal(t, in.RegisteredDirectory, "foo.io", "expected directory to be passed through")
		require.Equal(t, in.CommonName, "test.example.com", "unexpected common name in request")
		require.Empty(t, in.Id, "expected empty id in request")
		return nil, status.Error(codes.Canceled, "stop the test")
	}

	_, err = trisa.LookupPeer(ctx, "test.example.com", "foo.io")
	require.Error(t, err, "expected error from mock GDS lookup")

	// Assert lookups by vasp ID have correct request
	mgds.GetMock().OnLookup = func(ctx context.Context, in *gds.LookupRequest) (*gds.LookupReply, error) {
		require.Equal(t, in.RegisteredDirectory, "hello.world", "expected directory to be passed through")
		require.Equal(t, in.Id, "ed40acc8-60b3-4d0d-9dc2-15dc84790853", "unexpected vasp ID in request")
		require.Empty(t, in.CommonName, "expected empty common name in request")
		return nil, status.Error(codes.Canceled, "stop the test")
	}

	_, err = trisa.LookupPeer(ctx, "ed40acc8-60b3-4d0d-9dc2-15dc84790853", "hello.world")
	require.Error(t, err, "expected error from mock GDS lookup")
}

func TestLookupPeerErrors(t *testing.T) {
	trisa, err := network.NewMocked(nil)
	require.NoError(t, err, "could not create mocked trisa network")
	defer trisa.Close()

	// Set up a mock directory service response for from context lookup
	ds, err := trisa.Directory()
	require.NoError(t, err, "could not get directory")
	mgds, ok := ds.(*directory.MockGDS)
	require.True(t, ok, "expected a mocked directory servce")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Test error return from directory service
	err = mgds.GetMock().UseError(dmock.LookupRPC, codes.NotFound, "vasp not found")
	require.NoError(t, err, "could not setup GDS mock with error")

	_, err = trisa.LookupPeer(ctx, "bob.vaspbot.net", "")
	require.EqualError(t, err, "rpc error: code = NotFound desc = vasp not found", "unexpected error returned")

	// Test error in response from directory
	mgds.GetMock().OnLookup = func(ctx context.Context, in *gds.LookupRequest) (*gds.LookupReply, error) {
		return &gds.LookupReply{
			Error: &gds.Error{Code: 42, Message: "whoopsie"},
		}, nil
	}

	_, err = trisa.LookupPeer(ctx, "bob.vaspbot.net", "")
	require.EqualError(t, err, "[42] whoopsie", "unexpected error returned")

	// Test bad info returned, cannot create peer
	mgds.GetMock().OnLookup = func(ctx context.Context, in *gds.LookupRequest) (*gds.LookupReply, error) {
		return &gds.LookupReply{
			CommonName: "bob.vaspbot.net",
			VerifiedOn: "October 4, 2022",
		}, nil
	}

	_, err = trisa.LookupPeer(ctx, "bob.vaspbot.net", "")
	require.EqualError(t, err, "peer does not have an endpoint to connect on", "unexpected error returned")
}

func TestRefresh(t *testing.T) {
	trisa, err := network.NewMocked(nil)
	require.NoError(t, err, "could not create mocked trisa network")
	defer trisa.Close()

	trisaActual, ok := trisa.(*network.TRISANetwork)
	require.True(t, ok, "trisa should be a TRISANetwork")

	// Set up a mock directory service response for from context lookup
	ds, err := trisa.Directory()
	require.NoError(t, err, "could not get directory")
	mgds, ok := ds.(*directory.MockGDS)
	require.True(t, ok, "expected a mocked directory servce")

	err = mgds.GetMock().UseFixture(dmock.ListRPC, "testdata/gds/list.json")
	require.NoError(t, err, "could not setup GDS mock with fixture")
	require.Equal(t, 0, trisaActual.NPeers(), "peers cache is not empty")

	err = trisa.Refresh()
	require.NoError(t, err, "expected no error during refresh")

	require.Equal(t, 2, trisaActual.NPeers(), "peers cache is not empty")
	require.Equal(t, 1, mgds.GetMock().Calls[dmock.ListRPC], "unexpected number of GDS list calls")
	require.True(t, trisaActual.Contains("alice.vaspbot.net"), "missing alpha")
	require.True(t, trisaActual.Contains("bob.vaspbot.net"), "missing bravo")
}

func TestString(t *testing.T) {
	trisa, err := network.NewMocked(nil)
	require.NoError(t, err, "could not create mocked trisa network")
	defer trisa.Close()

	require.Equal(t, trisa.String(), "bufnet", "stringer should return conf.Directory.Network()")
}
