package interceptors_test

import (
	"context"
	"testing"
	"time"

	"github.com/trisacrypto/envoy/pkg/bufconn"
	"github.com/trisacrypto/envoy/pkg/logger"
	"github.com/trisacrypto/envoy/pkg/trisa/interceptors"
	"github.com/trisacrypto/envoy/pkg/trisa/mock"

	"github.com/stretchr/testify/require"
	api "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
	"go.rtnl.ai/ulid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func TestMonitor(t *testing.T) {
	opts := []grpc.ServerOption{
		grpc.StreamInterceptor(interceptors.StreamMonitoring()),
		grpc.UnaryInterceptor(interceptors.UnaryMonitoring()),
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

	t.Run("Unary", func(t *testing.T) {
		t.Cleanup(svc.Reset)
		svc.OnTransfer = func(ctx context.Context, _ *api.SecureEnvelope) (*api.SecureEnvelope, error) {
			requestID, ok := logger.RequestID(ctx)
			if !ok || requestID == "" {
				return nil, status.Error(codes.FailedPrecondition, "no request id in context")
			}

			if _, err := ulid.Parse(requestID); err != nil {
				return nil, status.Errorf(codes.FailedPrecondition, "could not parse request id: %s", err)
			}

			return &api.SecureEnvelope{}, nil
		}

		_, err := client.Transfer(context.Background(), &api.SecureEnvelope{})
		require.NoError(t, err)
	})

	t.Run("Stream", func(t *testing.T) {
		t.Cleanup(svc.Reset)
		svc.OnTransferStream = func(t api.TRISANetwork_TransferStreamServer) error {
			requestID, ok := logger.RequestID(t.Context())
			if !ok || requestID == "" {
				return status.Error(codes.FailedPrecondition, "no request id in context")
			}

			if _, err := ulid.Parse(requestID); err != nil {
				return status.Errorf(codes.FailedPrecondition, "could not parse request id: %s", err)
			}

			t.Send(&api.SecureEnvelope{})
			return nil
		}

		stream, err := client.TransferStream(context.Background())
		require.NoError(t, err, "expected no error on stream initialization")

		_, err = stream.Recv()
		require.NoError(t, err)
	})
}

func TestParseMethod(t *testing.T) {
	tests := []struct {
		FullMethod string
		service    string
		rpc        string
	}{
		{mock.TransferRPC, "trisa.api.v1beta1.TRISANetwork", "Transfer"},
		{mock.TransferStreamRPC, "trisa.api.v1beta1.TRISANetwork", "TransferStream"},
		{mock.ConfirmAddressRPC, "trisa.api.v1beta1.TRISANetwork", "ConfirmAddress"},
		{mock.KeyExchangeRPC, "trisa.api.v1beta1.TRISANetwork", "KeyExchange"},
		{mock.StatusRPC, "trisa.api.v1beta1.TRISAHealth", "Status"},
		{"foo", "unknown", "unknown"},
	}

	for _, tc := range tests {
		service, rpc := interceptors.ParseMethod(tc.FullMethod)
		require.Equal(t, tc.service, service, "unexpected service parsed from %q", tc.FullMethod)
		require.Equal(t, tc.rpc, rpc, "unexpected rpc parsed from %q", tc.FullMethod)
	}
}
