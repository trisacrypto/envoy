package mock

import (
	"context"
	"errors"
	"fmt"
	"os"
	"self-hosted-node/pkg/bufconn"

	api "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
)

// RPC names as defined by the grpc info method; used for tracking calls.
const (
	TransferRPC       = "trisa.api.v1beta1.TRISANetwork/Transfer"
	TransferStreamRPC = "trisa.api.v1beta1.TRISANetwork/TransferStream"
	KeyExchangeRPC    = "trisa.api.v1beta1.TRISANetwork/KeyExchange"
	ConfirmAddressRPC = "trisa.api.v1beta1.TRISANetwork/ConfirmAddress"
	StatusRPC         = "trisa.api.v1beta1.TRISAHealth/Status"
)

func New(bufnet *bufconn.Listener, opts ...grpc.ServerOption) *TRISA {
	if bufnet == nil {
		bufnet = bufconn.New()
	}

	remote := &TRISA{
		bufnet: bufnet,
		srv:    grpc.NewServer(opts...),
		Calls:  make(map[string]int),
	}

	api.RegisterTRISANetworkServer(remote.srv, remote)
	api.RegisterTRISAHealthServer(remote.srv, remote)
	go remote.srv.Serve(remote.bufnet.Sock())
	return remote
}

// TRISA implements the TRISANetwork and TRISAHealth services for mocking tests for
// clients that might call a remote server.
type TRISA struct {
	api.UnimplementedTRISAHealthServer
	api.UnimplementedTRISANetworkServer

	bufnet *bufconn.Listener
	srv    *grpc.Server
	Calls  map[string]int

	OnTransfer       func(context.Context, *api.SecureEnvelope) (*api.SecureEnvelope, error)
	OnTransferStream func(api.TRISANetwork_TransferStreamServer) error
	OnKeyExchange    func(context.Context, *api.SigningKey) (*api.SigningKey, error)
	OnConfirmAddress func(context.Context, *api.Address) (*api.AddressConfirmation, error)
	OnStatus         func(context.Context, *api.HealthCheck) (*api.ServiceState, error)
}

func (s *TRISA) Channel() *bufconn.Listener {
	return s.bufnet
}

func (s *TRISA) Shutdown() {
	s.srv.GracefulStop()
	s.bufnet.Close()
}

func (s *TRISA) Reset() {
	for key := range s.Calls {
		s.Calls[key] = 0
	}

	s.OnTransfer = nil
	s.OnTransferStream = nil
	s.OnKeyExchange = nil
	s.OnConfirmAddress = nil
	s.OnStatus = nil
}

// UseFixture loadsa a JSON fixture from disk (usually in a testdata folder) to use as
// the protocol buffer response to the specified RPC, simplifying handler mocking.
func (s *TRISA) UseFixture(rpc, path string) (err error) {
	var data []byte
	if data, err = os.ReadFile(path); err != nil {
		return fmt.Errorf("could not read fixture: %v", err)
	}

	jsonpb := &protojson.UnmarshalOptions{
		AllowPartial:   true,
		DiscardUnknown: true,
	}

	switch rpc {
	case TransferRPC:
		out := &api.SecureEnvelope{}
		if err = jsonpb.Unmarshal(data, out); err != nil {
			return fmt.Errorf("could not unmarshal json into %T: %v", out, err)
		}
		s.OnTransfer = func(context.Context, *api.SecureEnvelope) (*api.SecureEnvelope, error) {
			return out, nil
		}

	case TransferStreamRPC:
		return errors.New("cannot use fixture for a streaming RPC")

	case KeyExchangeRPC:
		out := &api.SigningKey{}
		if err = jsonpb.Unmarshal(data, out); err != nil {
			return fmt.Errorf("could not unmarshal json into %T: %v", out, err)
		}
		s.OnKeyExchange = func(context.Context, *api.SigningKey) (*api.SigningKey, error) {
			return out, nil
		}

	case ConfirmAddressRPC:
		out := &api.AddressConfirmation{}
		if err = jsonpb.Unmarshal(data, out); err != nil {
			return fmt.Errorf("could not unmarshal json into %T: %v", out, err)
		}
		s.OnConfirmAddress = func(context.Context, *api.Address) (*api.AddressConfirmation, error) {
			return out, nil
		}

	case StatusRPC:
		out := &api.ServiceState{}
		if err = jsonpb.Unmarshal(data, out); err != nil {
			return fmt.Errorf("could not unmarshal json into %T: %v", out, err)
		}
		s.OnStatus = func(context.Context, *api.HealthCheck) (*api.ServiceState, error) {
			return out, nil
		}

	default:
		return fmt.Errorf("unknown RPC %q", rpc)
	}

	return nil
}

// UseError allows you to specify a gRPC status error to return from the specified RPC.
func (s *TRISA) UseError(rpc string, code codes.Code, msg string) error {
	switch rpc {
	case TransferRPC:
		s.OnTransfer = func(context.Context, *api.SecureEnvelope) (*api.SecureEnvelope, error) {
			return nil, status.Error(code, msg)
		}

	case TransferStreamRPC:
		s.OnTransferStream = func(api.TRISANetwork_TransferStreamServer) error {
			return status.Error(code, msg)
		}

	case KeyExchangeRPC:
		s.OnKeyExchange = func(context.Context, *api.SigningKey) (*api.SigningKey, error) {
			return nil, status.Error(code, msg)
		}

	case ConfirmAddressRPC:
		s.OnConfirmAddress = func(context.Context, *api.Address) (*api.AddressConfirmation, error) {
			return nil, status.Error(code, msg)
		}

	case StatusRPC:
		s.OnStatus = func(context.Context, *api.HealthCheck) (*api.ServiceState, error) {
			return nil, status.Error(code, msg)
		}

	default:
		return fmt.Errorf("unknown RPC %q", rpc)
	}

	return nil
}

func (s *TRISA) Transfer(ctx context.Context, in *api.SecureEnvelope) (*api.SecureEnvelope, error) {
	s.Calls[TransferRPC]++
	if s.OnTransfer != nil {
		return s.OnTransfer(ctx, in)
	}
	return nil, status.Error(codes.Unavailable, "no mock function set for OnTransfer")
}

func (s *TRISA) TransferStream(stream api.TRISANetwork_TransferStreamServer) error {
	s.Calls[TransferStreamRPC]++
	if s.OnTransferStream != nil {
		return s.OnTransferStream(stream)
	}
	return status.Error(codes.Unavailable, "no mock function set for OnTransferStream")
}

func (s *TRISA) KeyExchange(ctx context.Context, in *api.SigningKey) (*api.SigningKey, error) {
	s.Calls[KeyExchangeRPC]++
	if s.OnKeyExchange != nil {
		return s.OnKeyExchange(ctx, in)
	}
	return nil, status.Error(codes.Unavailable, "no mock function set for OnKeyExchange")
}

func (s *TRISA) ConfirmAddress(ctx context.Context, in *api.Address) (*api.AddressConfirmation, error) {
	s.Calls[ConfirmAddressRPC]++
	if s.OnConfirmAddress != nil {
		return s.OnConfirmAddress(ctx, in)
	}
	return nil, status.Error(codes.Unavailable, "no mock function set for OnConfirmAddress")
}

func (s *TRISA) Status(ctx context.Context, in *api.HealthCheck) (*api.ServiceState, error) {
	s.Calls[StatusRPC]++
	if s.OnStatus != nil {
		return s.OnStatus(ctx, in)
	}
	return nil, status.Error(codes.Unavailable, "no mock function set for OnStatus")
}
