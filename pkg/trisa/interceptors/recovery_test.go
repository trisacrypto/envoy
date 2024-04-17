package interceptors_test

import (
	"context"
	"testing"
	"time"

	"github.com/trisacrypto/envoy/pkg/bufconn"
	"github.com/trisacrypto/envoy/pkg/trisa/interceptors"
	"github.com/trisacrypto/envoy/pkg/trisa/mock"

	"github.com/stretchr/testify/require"
	api "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
)

func TestRecovery(t *testing.T) {
	opts := []grpc.ServerOption{
		grpc.StreamInterceptor(interceptors.StreamRecovery()),
		grpc.UnaryInterceptor(interceptors.UnaryRecovery()),
	}

	sock := bufconn.New()
	svc := mock.New(sock, opts...)
	defer svc.Shutdown()

	// Create a client to connect to the mock TRISA server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	creds := grpc.WithTransportCredentials(insecure.NewCredentials())
	cc, err := sock.Connect(ctx, creds)
	require.NoError(t, err, "could not connect to bufconn")

	client := api.NewTRISANetworkClient(cc)

	t.Run("UnaryPanic", func(t *testing.T) {
		t.Cleanup(svc.Reset)
		svc.OnTransfer = func(context.Context, *api.SecureEnvelope) (*api.SecureEnvelope, error) {
			panic("mayday mayday")
		}

		rep, err := client.Transfer(ctx, &api.SecureEnvelope{})
		require.EqualError(t, err, "rpc error: code = Internal desc = an unhandled exception occurred")
		require.Nil(t, rep, "expected a nil response after a panic")
	})

	t.Run("StreamPanic", func(t *testing.T) {
		t.Cleanup(svc.Reset)
		svc.OnTransferStream = func(stream api.TRISANetwork_TransferStreamServer) error {
			panic("mayday mayday")
		}

		stream, err := client.TransferStream(ctx)
		require.NoError(t, err, "expected no error initializing stream")

		rep, err := stream.Recv()
		require.EqualError(t, err, "rpc error: code = Internal desc = an unhandled exception occurred")
		require.Nil(t, rep, "expected a nil response after a panic")
	})

	t.Run("UnaryNoPanic", func(t *testing.T) {
		t.Cleanup(svc.Reset)
		svc.UseError(mock.TransferRPC, codes.NotFound, "not found")

		_, err := client.Transfer(ctx, &api.SecureEnvelope{})
		require.EqualError(t, err, "rpc error: code = NotFound desc = not found")
	})

	t.Run("StreamNoPanic", func(t *testing.T) {
		t.Cleanup(svc.Reset)
		svc.UseError(mock.TransferStreamRPC, codes.DataLoss, "data loss")

		stream, err := client.TransferStream(ctx)
		require.NoError(t, err, "expected no error initializing stream")

		_, err = stream.Recv()
		require.EqualError(t, err, "rpc error: code = DataLoss desc = data loss")
	})
}
