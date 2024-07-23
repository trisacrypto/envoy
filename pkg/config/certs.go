package config

import "github.com/trisacrypto/trisa/pkg/trust"

type CertsCacheLoader interface {
	Validate() error
	LoadCerts() (*trust.Provider, error)
	LoadPool() (trust.ProviderPool, error)
	Reset()
}
