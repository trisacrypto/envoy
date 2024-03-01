package mock

import (
	"self-hosted-node/pkg/store/dsn"
	"self-hosted-node/pkg/store/models"

	"github.com/oklog/ulid/v2"
)

// Store implements the store.Store interface implemented as an in-memory mock
// interface for testing and development purposes.
type Store struct{}

func Open(uri *dsn.DSN) (*Store, error) {
	return nil, nil
}

func (s *Store) Close() error {
	return nil
}

func (s *Store) ListAccounts(page *models.PageInfo) (*models.AccountsPage, error) {
	return nil, nil
}

func (s *Store) CreateAccount(*models.Account) error {
	return nil
}

func (s *Store) RetrieveAccount(id ulid.ULID) (*models.Account, error) {
	return nil, nil
}

func (s *Store) UpdateAccount(*models.Account) error {
	return nil
}

func (s *Store) DeleteAccount(id ulid.ULID) error {
	return nil
}

func (s *Store) ListCryptoAddresses(page *models.PageInfo) (*models.CryptoAddressPage, error) {
	return nil, nil
}

func (s *Store) CreateCryptoAddress(*models.CryptoAddress) error {
	return nil
}

func (s *Store) RetrieveCryptoAddress(id ulid.ULID) (*models.CryptoAddress, error) {
	return nil, nil
}

func (s *Store) UpdateCryptoAddress(*models.CryptoAddress) error {
	return nil
}

func (s *Store) DeleteCryptoAddress(id ulid.ULID) error {
	return nil
}
