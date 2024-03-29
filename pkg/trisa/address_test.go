package trisa_test

import (
	"context"

	api "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
)

func (s *trisaTestSuite) TestConfirmAddress() {
	require := s.Require()

	rep, err := s.client.ConfirmAddress(context.Background(), &api.Address{})
	require.EqualError(err, "rpc error: code = Unimplemented desc = address confirmation is not part of the trisa v9 whitepaper spec")
	require.Nil(rep)
}
