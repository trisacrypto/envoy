package trisa_test

import "context"

func (s *trisaTestSuite) TestTransfer() {
	require := s.Require()

	req, err := loadSecureEnvelope("testdata/fixtures/secenv_transaction.pb.json")
	require.NoError(err, "could not load secenv_transaction fixture")

	_, err = s.client.Transfer(context.Background(), req)
	require.NoError(err, "unable to make transfer rpc request")
}
