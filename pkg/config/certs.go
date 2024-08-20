package config

import (
	"errors"
	"fmt"

	"github.com/trisacrypto/trisa/pkg/trust"
)

type CertsCacheLoader interface {
	Validate() error
	LoadCerts() (*trust.Provider, error)
	LoadPool() (trust.ProviderPool, error)
	Reset()
}

var (
	ErrMTLSPoolNotConfigured  = errors.New("invalid configuration: no certificate pool found")
	ErrMTLSCertsNotConfigured = errors.New("invalid configuration: no certificates found")
)

type MTLSConfig struct {
	Pool  string `required:"false" desc:"path to the x509 cert pool to use for mTLS connection authentication (optional)"`
	Certs string `required:"false" desc:"path to the x509 certificate and private key for mTLS authentication, with or without the certificate chain"`
	certs *trust.Provider
	pool  trust.ProviderPool
}

// LoadCerts returns the mtls trust provider for setting up an mTLS 1.3 config.
// NOTE: this method is not thread-safe, ensure it is not used from multiple go-routines
func (c *MTLSConfig) LoadCerts() (_ *trust.Provider, err error) {
	// Attempt to load the certificates from disk and cache them.
	if c.certs == nil {
		if err = c.load(); err != nil {
			return nil, err
		}
	}

	// If no certificates are available, return a configuration error
	if c.certs == nil {
		return nil, ErrMTLSCertsNotConfigured
	}
	return c.certs, nil
}

// LoadPool returns the mtls TRISA trust provider pool for creating an x509.Pool.
// NOTE: this method is not thread-safe, ensure it is not used from multiple go-routines
func (c *MTLSConfig) LoadPool() (_ trust.ProviderPool, err error) {
	if len(c.pool) == 0 && c.certs == nil {
		if err = c.load(); err != nil {
			return nil, err
		}
	}

	// Load either the configured certificate pool or use the certs chain specified.
	switch {
	case c.pool != nil:
		return c.pool, nil
	case c.certs != nil:
		return trust.NewPool(c.certs), nil
	default:
		return nil, ErrMTLSPoolNotConfigured
	}
}

// Load and cache the certificates and provider pool from disk.
func (c *MTLSConfig) load() (err error) {
	var sz *trust.Serializer
	if sz, err = trust.NewSerializer(false); err != nil {
		return err
	}

	if c.Certs != "" {
		if c.certs, err = sz.ReadFile(c.Certs); err != nil {
			return fmt.Errorf("could not parse certs: %w", err)
		}
	}

	if c.Pool != "" {
		if c.pool, err = sz.ReadPoolFile(c.Pool); err != nil {
			return fmt.Errorf("could not parse cert pool: %w", err)
		}
	}
	return nil
}

// Reset the certs cache to force load the pool and certs again
// NOTE: this method is not thread-safe, ensure it is not used from multiple go-routines
func (c *MTLSConfig) Reset() {
	c.pool = nil
	c.certs = nil
}
