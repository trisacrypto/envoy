package trisa_test

import (
	"context"
	"testing"
	"time"

	"self-hosted-node/pkg/bufconn"
	"self-hosted-node/pkg/config"
	"self-hosted-node/pkg/trisa"

	"github.com/stretchr/testify/require"
	api "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestAvailableInterceptor(t *testing.T) {
	// Create a maintenance mode TRISA server
	svc, err := trisa.New(config.TRISAConfig{Maintenance: true}, nil)
	require.NoError(t, err, "could not create maintenance mode TRISA server")

	sock := bufconn.New()
	go svc.Run(sock.Sock())
	defer svc.Shutdown()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cc, err := sock.Connect(ctx, grpc.WithTransportCredentials(insecure.NewCredentials()))
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
