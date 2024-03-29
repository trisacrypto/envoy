package interceptors_test

import (
	"context"
	"testing"
	"time"

	"self-hosted-node/pkg/bufconn"
	"self-hosted-node/pkg/config"
	"self-hosted-node/pkg/trisa/interceptors"
	"self-hosted-node/pkg/trisa/mock"

	"github.com/stretchr/testify/require"
	api "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestMaintenanceInterceptor(t *testing.T) {
	// Create a mock maintenance mode TRISA server
	conf := config.TRISAConfig{
		Maintenance: true,
		Certs:       "testdata/certs/alice.vaspbot.net.pem",
		Pool:        "testdata/certs/trisatest.dev.pem",
	}

	opts := []grpc.ServerOption{
		grpc.StreamInterceptor(interceptors.StreamAvailable(conf)),
		grpc.UnaryInterceptor(interceptors.UnaryAvailable(conf)),
	}

	sock := bufconn.New()
	svc := mock.New(sock, opts...)
	defer svc.Shutdown()

	// Mock the OnStatus method
	svc.OnStatus = func(context.Context, *api.HealthCheck) (*api.ServiceState, error) {
		return &api.ServiceState{
			Status: api.ServiceState_MAINTENANCE,
		}, nil
	}

	// Create a client to connect to the mock TRISA server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	creds := grpc.WithTransportCredentials(insecure.NewCredentials())
	cc, err := sock.Connect(ctx, creds)
	require.NoError(t, err, "could not connect to bufconn")

	healthClient := api.NewTRISAHealthClient(cc)
	trisaClient := api.NewTRISANetworkClient(cc)

	t.Run("Unary", func(t *testing.T) {
		// Should receive unavailable error for transfers
		_, err := trisaClient.Transfer(ctx, &api.SecureEnvelope{})
		require.EqualError(t, err, "rpc error: code = Unavailable desc = conducting temporary maintenance", "expected interceptor to return unavailable")

		// Should get a response back from the status endpoint
		out, err := healthClient.Status(ctx, &api.HealthCheck{})
		require.NoError(t, err, "expected ok response from health check")
		require.Equal(t, api.ServiceState_MAINTENANCE, out.Status)
	})

	t.Run("Stream", func(t *testing.T) {
		// Should receive unavailable errors for streaming transfers
		stream, err := trisaClient.TransferStream(ctx)
		require.NoError(t, err)

		_, err = stream.Recv()
		require.EqualError(t, err, "rpc error: code = Unavailable desc = conducting temporary maintenance", "expected interceptor to return unavailable error")
	})
}
