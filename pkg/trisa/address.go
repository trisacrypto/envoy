package trisa

import (
	"context"

	api "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Address confirmation allows an originator VASP to establish that a beneficiary VASP
// has control of a crypto wallet address, prior to sending transaction information with
// sensitive PII data.
//
// NOTE: this RPC is currently undefined by the v9 whitepaper
func (s *Server) ConfirmAddress(ctx context.Context, in *api.Address) (*api.AddressConfirmation, error) {
	return nil, status.Error(codes.Unimplemented, "address confirmation is not part of the trisa v9 whitepaper spec")
}
