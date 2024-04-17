package network_test

import (
	"testing"

	"github.com/trisacrypto/envoy/pkg/bufconn"
	"github.com/trisacrypto/envoy/pkg/config"
	"github.com/trisacrypto/envoy/pkg/trisa/network"

	"github.com/stretchr/testify/require"
)

func TestDialers(t *testing.T) {
	conf := config.TRISAConfig{
		Certs: "testdata/notreal.pem",
		Pool:  "testdata/notreal.pem",
	}

	_, err := network.TRISADialer(conf)
	require.EqualError(t, err, "could not parse certs: open testdata/notreal.pem: no such file or directory", "should not be able to create a dialer without certs")

	conf.Certs = "testdata/alice.pem"
	_, err = network.TRISADialer(conf)
	require.EqualError(t, err, "could not parse cert pool: open testdata/notreal.pem: no such file or directory", "should not be able to create a dialer without certs")

	conf.Pool = "testdata/pool.pem"
	dialer, err := network.TRISADialer(conf)
	require.NoError(t, err, "could not create a dialer with valid certs and pool")

	opts, err := dialer("example.com:443")
	require.NoError(t, err, "could not create mTLS credentials for dialer")
	require.Len(t, opts, 1, "incorrect number of dial options returned")

	bufnet := bufconn.New()
	dialer, err = network.BufnetDialer(bufnet)
	require.NoError(t, err, "could not create bufnet dialer")

	opts, err = dialer("example.com:443")
	require.NoError(t, err, "could not dial bufnet")
	require.Len(t, opts, 2, "incorrect number of dial options returned")
}
