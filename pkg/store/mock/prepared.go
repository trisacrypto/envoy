package mock

import "github.com/trisacrypto/envoy/pkg/store/models"

type PreparedTransaction struct{}

var _ models.PreparedTransaction = &PreparedTransaction{}

func (p *PreparedTransaction) Created() bool {
	return false
}

func (p *PreparedTransaction) Fetch() (*models.Transaction, error) {
	return nil, nil
}

func (p *PreparedTransaction) Update(*models.Transaction) error {
	return nil
}

func (p *PreparedTransaction) AddCounterparty(*models.Counterparty) error {
	return nil
}

func (p *PreparedTransaction) AddEnvelope(*models.SecureEnvelope) error {
	return nil
}

func (p *PreparedTransaction) CreateSunrise(*models.Sunrise) error { return nil }
func (p *PreparedTransaction) UpdateSunrise(*models.Sunrise) error { return nil }

func (p *PreparedTransaction) Rollback() error {
	return nil
}

func (p *PreparedTransaction) Commit() error {
	return nil
}
