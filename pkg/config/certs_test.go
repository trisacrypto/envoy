package config_test

import (
	"encoding/pem"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/config"
)

// A certificate chain (leaf and CA) with the leaf's private key; see testdata/generate.sh
const (
	certsFixture    = "testdata/trisa.example.dev.pem"
	certsCommonName = "trisa.example.dev"
)

// All configurations that embed an MTLSConfig should be certificate cache loaders.
var (
	_ config.CertsCacheLoader = (*config.TRISAConfig)(nil)
	_ config.CertsCacheLoader = (*config.TRPConfig)(nil)
	_ config.CertsCacheLoader = (*config.WebhookConfig)(nil)
)

// Copies the certs fixture into a temporary directory that is cleaned up when the test
// completes and returns the path to the copy. If public is true then the private key
// is omitted from the copy.
func copyCertsFixture(t *testing.T, public bool) string {
	t.Helper()

	data, err := os.ReadFile(certsFixture)
	require.NoError(t, err, "could not read testdata certificate fixture")

	if public {
		var certs []byte
		for block, rest := pem.Decode(data); block != nil; block, rest = pem.Decode(rest) {
			if block.Type == "CERTIFICATE" {
				certs = append(certs, pem.EncodeToMemory(block)...)
			}
		}
		require.NotEmpty(t, certs, "no certificates found in testdata certificate fixture")
		data = certs
	}

	path := filepath.Join(t.TempDir(), "copy.example.dev.pem")
	require.NoError(t, os.WriteFile(path, data, 0600), "could not write temporary certificate fixture")
	return path
}

func TestMTLSConfigLoad(t *testing.T) {
	t.Run("CertsAndPool", func(t *testing.T) {
		conf := &config.MTLSConfig{Certs: certsFixture, Pool: certsFixture}

		certs, err := conf.LoadCerts()
		require.NoError(t, err, "was unable to load certs")
		require.True(t, certs.IsPrivate(), "certs do not contain private key")

		pool, err := conf.LoadPool()
		require.NoError(t, err, "was unable to load pool")
		require.Len(t, pool, 1, "unexpected cert pool length")
	})

	t.Run("PoolFromCerts", func(t *testing.T) {
		// If no pool is specified, it should be created from the certificate chain.
		conf := &config.MTLSConfig{Certs: certsFixture}

		pool, err := conf.LoadPool()
		require.NoError(t, err, "was unable to load pool from certs")
		require.Len(t, pool, 1, "unexpected cert pool length")
	})

	t.Run("PoolOnly", func(t *testing.T) {
		conf := &config.MTLSConfig{Pool: certsFixture}

		_, err := conf.LoadCerts()
		require.ErrorIs(t, err, config.ErrMTLSCertsNotConfigured)

		pool, err := conf.LoadPool()
		require.NoError(t, err, "was unable to load pool")
		require.Len(t, pool, 1, "unexpected cert pool length")
	})

	t.Run("NotConfigured", func(t *testing.T) {
		conf := &config.MTLSConfig{}

		_, err := conf.LoadCerts()
		require.ErrorIs(t, err, config.ErrMTLSCertsNotConfigured)

		_, err = conf.LoadPool()
		require.ErrorIs(t, err, config.ErrMTLSPoolNotConfigured)
	})

	t.Run("NotFound", func(t *testing.T) {
		missing := filepath.Join(t.TempDir(), "missing.pem")

		conf := &config.MTLSConfig{Certs: missing}
		_, err := conf.LoadCerts()
		require.ErrorIs(t, err, fs.ErrNotExist)
		require.ErrorContains(t, err, "could not parse certs")

		_, err = conf.LoadPool()
		require.ErrorIs(t, err, fs.ErrNotExist)

		conf = &config.MTLSConfig{Certs: certsFixture, Pool: missing}
		_, err = conf.LoadPool()
		require.ErrorIs(t, err, fs.ErrNotExist)
		require.ErrorContains(t, err, "could not parse cert pool")
	})
}

func TestMTLSConfigCache(t *testing.T) {
	path := copyCertsFixture(t, false)
	conf := &config.MTLSConfig{Certs: path, Pool: path}

	requireLoaded := func(t *testing.T, conf *config.MTLSConfig, msg string) {
		t.Helper()
		_, err := conf.LoadCerts()
		require.NoError(t, err, msg)
		_, err = conf.LoadPool()
		require.NoError(t, err, msg)
	}

	requireLoaded(t, conf, "could not load certs and pool from the temporary fixture")

	// Delete the fixture so that the certs and pool can only be loaded from the cache.
	require.NoError(t, os.Remove(path), "could not delete the temporary fixture")
	requireLoaded(t, conf, "expected certs and pool to be cached after the first load")

	// The cache should be retained when the config is copied, e.g. by config.Get()
	copied := *conf
	requireLoaded(t, &copied, "expected certs and pool cache to be copied with the config")

	// Once the cache is reset the certs and pool must be loaded from disk again.
	conf.Reset()
	_, err := conf.LoadCerts()
	require.ErrorIs(t, err, fs.ErrNotExist, "expected certs to be loaded from disk after reset")
	_, err = conf.LoadPool()
	require.ErrorIs(t, err, fs.ErrNotExist, "expected pool to be loaded from disk after reset")

	// Resetting the config should not reset any of its copies.
	requireLoaded(t, &copied, "expected the copied cache to be independent of the original")
}

func TestMTLSConfigClientTLS(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		tests := map[string]*config.MTLSConfig{
			"CertsAndPool": {Certs: certsFixture, Pool: certsFixture},
			"CertsOnly":    {Certs: certsFixture},
		}

		for name, conf := range tests {
			tlsConf, err := conf.ClientTLSConfig()
			require.NoError(t, err, "could not create client tls config for %s", name)
			require.Len(t, tlsConf.Certificates, 1, "expected a client certificate for %s", name)
			require.NotNil(t, tlsConf.Certificates[0].PrivateKey, "expected a client private key for %s", name)
			require.NotNil(t, tlsConf.RootCAs, "expected a root ca pool for %s", name)
		}
	})

	t.Run("NotConfigured", func(t *testing.T) {
		tests := map[string]*config.MTLSConfig{
			"Empty":    {},
			"PoolOnly": {Pool: certsFixture},
		}

		for name, conf := range tests {
			tlsConf, err := conf.ClientTLSConfig()
			require.EqualError(t, err, "could not configure mTLS client certs: invalid configuration: no certificates found", name)
			require.Nil(t, tlsConf, name)
		}
	})

	t.Run("NotPrivate", func(t *testing.T) {
		conf := &config.MTLSConfig{Certs: copyCertsFixture(t, true)}

		tlsConf, err := conf.ClientTLSConfig()
		require.ErrorIs(t, err, config.ErrMTLSCertsNotPrivate)
		require.Nil(t, tlsConf)
	})

	t.Run("PoolNotFound", func(t *testing.T) {
		// Certs must be loaded first for the pool path to be read from disk.
		conf := &config.MTLSConfig{Certs: certsFixture, Pool: filepath.Join(t.TempDir(), "missing.pem")}

		tlsConf, err := conf.ClientTLSConfig()
		require.ErrorContains(t, err, "could not configure mTLS client certs: could not parse cert pool")
		require.Nil(t, tlsConf)
	})
}

func TestMTLSConfigLeafCertificate(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		conf := &config.MTLSConfig{Certs: certsFixture}

		leaf, err := conf.GetLeafCertificate()
		require.NoError(t, err, "could not get leaf certificate")
		require.Equal(t, certsCommonName, leaf.Subject.CommonName)

		require.Equal(t, certsCommonName, conf.CommonName())
		require.Equal(t, leaf.NotBefore, conf.IssuedAt())
		require.Equal(t, leaf.NotAfter, conf.Expires())
		require.True(t, conf.Expires().After(conf.IssuedAt()), "expected certs to expire after they are issued")
	})

	t.Run("NotFound", func(t *testing.T) {
		// Certificate info should be zero valued if the certs cannot be loaded.
		conf := &config.MTLSConfig{Certs: filepath.Join(t.TempDir(), "missing.pem")}

		_, err := conf.GetLeafCertificate()
		require.ErrorIs(t, err, fs.ErrNotExist)

		require.Empty(t, conf.CommonName())
		require.Zero(t, conf.IssuedAt())
		require.Zero(t, conf.Expires())
	})

	t.Run("NotConfigured", func(t *testing.T) {
		// If no certs are specified an error should be returned rather than a panic
		// and the certificate info should be zero valued.
		tests := map[string]*config.MTLSConfig{
			"Empty":    {},
			"PoolOnly": {Pool: certsFixture},
		}

		for name, conf := range tests {
			leaf, err := conf.GetLeafCertificate()
			require.ErrorIs(t, err, config.ErrMTLSCertsNotConfigured, name)
			require.Nil(t, leaf, name)

			require.Empty(t, conf.CommonName(), name)
			require.Zero(t, conf.IssuedAt(), name)
			require.Zero(t, conf.Expires(), name)
		}
	})
}
