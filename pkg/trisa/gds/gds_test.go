package gds_test

import (
	"context"
	"testing"

	"github.com/trisacrypto/envoy/pkg/config"

	"github.com/trisacrypto/envoy/pkg/bufconn"
	"github.com/trisacrypto/envoy/pkg/trisa/gds"
	mockgds "github.com/trisacrypto/envoy/pkg/trisa/gds/mock"

	"github.com/stretchr/testify/require"
	members "github.com/trisacrypto/directory/pkg/gds/members/v1alpha1"
	api "github.com/trisacrypto/trisa/pkg/trisa/gds/api/v1beta1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
)

func TestGDSString(t *testing.T) {
	testCases := []struct {
		endpoint string
		expected string
	}{
		{"", ""},
		{"api.trisatest.net:443", "trisatest.net"},
		{"api.test.vaspdirectory.net:3000", "vaspdirectory.net"},
		{"localhost:2226", "localhost"},
		{"trisatest.net:443", "trisatest.net"},
		{"api.trisatest.net", "trisatest.net"},
		{"trisatest.net", "trisatest.net"},
	}

	for i, tc := range testCases {
		conf := config.TRISAConfig{Directory: config.DirectoryConfig{Endpoint: tc.endpoint}}
		gds := gds.New(conf)
		require.Equal(t, tc.expected, gds.String(), "test case %d", i)
	}
}

func TestGDSConnect(t *testing.T) {
	conf := config.TRISAConfig{
		Directory: config.DirectoryConfig{
			Insecure:        true,
			Endpoint:        bufconn.Endpoint,
			MembersEndpoint: bufconn.Endpoint,
		},
	}

	directory := gds.New(conf)
	bufnet := bufconn.New()
	defer bufnet.Close()

	// Should error if we try to connect without transport security
	err := directory.Connect(grpc.WithContextDialer(bufnet.Dialer))
	require.Error(t, err, "errors from the dialer should be passed up to caller (transport security required)")

	// Should be able to successfully connect
	err = directory.Connect(grpc.WithContextDialer(bufnet.Dialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err, "could not connect gds via bufconn")

	// Should not be able to connect to an already connected directory
	err = directory.Connect(grpc.WithContextDialer(bufnet.Dialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.ErrorIs(t, err, gds.ErrAlreadyConnected, "was able to connect to an already connected directory")

	// Should be able to close and reconnect to the directory
	err = directory.Close()
	require.NoError(t, err, "could not close connection to directory")

	err = directory.Connect(grpc.WithContextDialer(bufnet.Dialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err, "could not connect gds via bufconn")
}

func TestGDSDefaultConnect(t *testing.T) {
	// Test secure connections with mTLS and TLS certificates
	conf := config.TRISAConfig{
		Directory: config.DirectoryConfig{
			Insecure:        false,
			Endpoint:        bufconn.Endpoint,
			MembersEndpoint: bufconn.Endpoint,
		},
	}

	directory := gds.New(conf)
	bufnet := bufconn.New()
	defer bufnet.Close()

	// Should error if we try to connect without transport security
	err := directory.Connect()
	require.Error(t, err, "errors from the dialer should be passed up to caller (transport security required)")

	// Should be able to successfully connect
	err = directory.Connect(grpc.WithContextDialer(bufnet.Dialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err, "could not connect gds via bufconn")

	// Should not be able to connect to an already connected directory
	err = directory.Connect(grpc.WithContextDialer(bufnet.Dialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.ErrorIs(t, err, gds.ErrAlreadyConnected, "was able to connect to an already connected directory")

	// Should be able to close and reconnect to the directory
	err = directory.Close()
	require.NoError(t, err, "could not close connection to directory")

	err = directory.Connect(grpc.WithContextDialer(bufnet.Dialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err, "could not connect gds via bufconn")
}

func TestGDSLookup(t *testing.T) {
	mock, directory := createMockGDS(t)

	_, err := directory.Lookup(context.TODO(), &api.LookupRequest{CommonName: "trisa.example.com"})
	require.ErrorIs(t, err, gds.ErrNotConnected, "should not be able to call Lookup RPC when not connected")

	err = directory.Connect(grpc.WithContextDialer(mock.Channel().Dialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err, "could not connect to mock GDS via bucfonn")
	defer directory.Close()

	mock.OnLookup = func(context.Context, *api.LookupRequest) (*api.LookupReply, error) {
		return &api.LookupReply{Id: "96f51748-f72a-4ae2-b52d-55d9fb7b897a"}, nil
	}

	// Should be able to make a lookup request
	rep, err := directory.Lookup(context.TODO(), &api.LookupRequest{CommonName: "trisa.example.com"})
	require.NoError(t, err, "could not make lookup request to GDS")
	require.NotNil(t, rep, "received unexpected response from GDS")
	require.Equal(t, "96f51748-f72a-4ae2-b52d-55d9fb7b897a", rep.Id, "received unexpected response from GDS")
	require.Equal(t, 1, mock.Calls[mockgds.LookupRPC])

	// Should return status errors from GDS
	mock.UseError(mockgds.LookupRPC, codes.NotFound, "vasp not found")
	_, err = directory.Lookup(context.TODO(), &api.LookupRequest{CommonName: "trisa.example.com"})
	require.Error(t, err, "error not passed through from GDS")
	require.Equal(t, 2, mock.Calls[mockgds.LookupRPC])
}

func TestGDSSearch(t *testing.T) {
	mock, directory := createMockGDS(t)

	_, err := directory.Search(context.TODO(), &api.SearchRequest{Name: []string{"Alice VASP"}})
	require.ErrorIs(t, err, gds.ErrNotConnected, "should not be able to call Lookup RPC when not connected")

	err = directory.Connect(grpc.WithContextDialer(mock.Channel().Dialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err, "could not connect to mock GDS via bucfonn")
	defer directory.Close()

	mock.OnSearch = func(context.Context, *api.SearchRequest) (*api.SearchReply, error) {
		return &api.SearchReply{Results: []*api.SearchReply_Result{{Id: "96f51748-f72a-4ae2-b52d-55d9fb7b897a"}}}, nil
	}

	// Should be able to make a search request
	rep, err := directory.Search(context.TODO(), &api.SearchRequest{Name: []string{"Alice VASP"}})
	require.NoError(t, err, "could not make lookup request to GDS")
	require.NotNil(t, rep, "received unexpected response from GDS")
	require.Len(t, rep.Results, 1, "unexpected search results returned")
	require.Equal(t, "96f51748-f72a-4ae2-b52d-55d9fb7b897a", rep.Results[0].Id, "received unexpected response from GDS")
	require.Equal(t, 1, mock.Calls[mockgds.SearchRPC])

	// Should return status errors from GDS
	mock.UseError(mockgds.SearchRPC, codes.NotFound, "no results found")
	_, err = directory.Search(context.TODO(), &api.SearchRequest{Name: []string{"Alice VASP"}})
	require.Error(t, err, "error not passed through from GDS")
	require.Equal(t, 2, mock.Calls[mockgds.SearchRPC])
}

func TestGDSList(t *testing.T) {
	mock, directory := createMockGDS(t)

	_, err := directory.List(context.TODO(), &members.ListRequest{})
	require.ErrorIs(t, err, gds.ErrNotConnected, "should not be able to call Lookup RPC when not connected")

	err = directory.Connect(grpc.WithContextDialer(mock.Channel().Dialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err, "could not connect to mock GDS via bucfonn")
	defer directory.Close()

	mock.OnList = func(context.Context, *members.ListRequest) (*members.ListReply, error) {
		return &members.ListReply{Vasps: []*members.VASPMember{{Id: "96f51748-f72a-4ae2-b52d-55d9fb7b897a"}}}, nil
	}

	// Should be able to make a lookup request
	rep, err := directory.List(context.TODO(), &members.ListRequest{})
	require.NoError(t, err, "could not make lookup request to GDS")
	require.NotNil(t, rep, "received unexpected response from GDS")
	require.Len(t, rep.Vasps, 1, "unexpected search results returned")
	require.Equal(t, "96f51748-f72a-4ae2-b52d-55d9fb7b897a", rep.Vasps[0].Id, "received unexpected response from GDS")
	require.Equal(t, 1, mock.Calls[mockgds.ListRPC])

	// Should return status errors from GDS
	mock.UseError(mockgds.ListRPC, codes.InvalidArgument, "invalid page token")
	_, err = directory.List(context.TODO(), &members.ListRequest{PageToken: "foo"})
	require.Error(t, err, "error not passed through from GDS")
	require.Equal(t, 2, mock.Calls[mockgds.ListRPC])
}

func TestGDSStatus(t *testing.T) {
	mock, directory := createMockGDS(t)

	_, err := directory.Status(context.TODO(), &api.HealthCheck{})
	require.ErrorIs(t, err, gds.ErrNotConnected, "should not be able to call Lookup RPC when not connected")

	err = directory.Connect(grpc.WithContextDialer(mock.Channel().Dialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err, "could not connect to mock GDS via bucfonn")
	defer directory.Close()

	mock.OnStatus = func(context.Context, *api.HealthCheck) (*api.ServiceState, error) {
		return &api.ServiceState{Status: api.ServiceState_HEALTHY}, nil
	}

	// Should be able to make a lookup request
	rep, err := directory.Status(context.TODO(), &api.HealthCheck{})
	require.NoError(t, err, "could not make lookup request to GDS")
	require.NotNil(t, rep, "received unexpected response from GDS")
	require.Equal(t, api.ServiceState_HEALTHY, rep.Status, "received unexpected response from GDS")
	require.Equal(t, 1, mock.Calls[mockgds.StatusRPC])

	// Should return status errors from GDS
	mock.UseError(mockgds.StatusRPC, codes.Unavailable, "unavailable")
	_, err = directory.Status(context.TODO(), &api.HealthCheck{})
	require.Error(t, err, "error not passed through from GDS")
	require.Equal(t, 2, mock.Calls[mockgds.StatusRPC])
}

func createMockGDS(t *testing.T) (mgds *mockgds.GDS, directory *gds.GDS) {
	mgds = mockgds.New(nil)
	t.Cleanup(mgds.Shutdown)

	conf := config.TRISAConfig{
		Directory: config.DirectoryConfig{
			Insecure:        true,
			Endpoint:        bufconn.Endpoint,
			MembersEndpoint: bufconn.Endpoint,
		},
	}

	directory = gds.New(conf)
	return mgds, directory
}
