package trisa_test

import (
	"context"
	"time"

	"github.com/trisacrypto/envoy/pkg/bufconn"
	api "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
)

func (s *trisaTestSuite) TestStatus() {
	require := s.Require()

	// Create a new TRISAHealth service client
	creds, err := loadClientCredentials(bufconn.Endpoint, "testdata/certs/client.trisatest.dev.pem")
	require.NoError(err, "could not load client credentiasls")

	cc, err := s.conn.Connect(context.Background(), creds)
	require.NoError(err, "could not connect tot he bufnet")
	defer cc.Close()

	healthClient := api.NewTRISAHealthClient(cc)

	// Execute a health check and verify the response
	out, err := healthClient.Status(context.Background(), &api.HealthCheck{})
	require.NoError(err, "could not make status request")
	require.NotNil(out, "unexpected nil reply from server")

	require.Equal(api.ServiceState_HEALTHY, out.Status)

	_, err = time.Parse(time.RFC3339, out.NotBefore)
	require.NoError(err, "could not parse not before as RFC3339 timestamp")

	_, err = time.Parse(time.RFC3339, out.NotAfter)
	require.NoError(err, "could not parse not after as RFC3339 timestamp")
}
