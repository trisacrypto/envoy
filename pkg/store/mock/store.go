package mock

import (
	"context"
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

func (s *Store) ListTransactions(context.Context, *models.PageInfo) (*models.TransactionPage, error) {
	return nil, nil
}

func (s *Store) CreateTransaction(context.Context, *models.Transaction) error {
	return nil
}

func (s *Store) RetrieveTransaction(context.Context, ulid.ULID) (*models.Transaction, error) {
	return nil, nil
}

func (s *Store) UpdateTransaction(context.Context, *models.Transaction) error {
	return nil
}

func (s *Store) DeleteTransaction(context.Context, ulid.ULID) error {
	return nil
}

func (s *Store) ListSecureEnvelopes(ctx context.Context, txID ulid.ULID, page *models.PageInfo) (*models.SecureEnvelopePage, error) {
	return nil, nil
}

func (s *Store) CreateSecureEnvelope(context.Context, *models.SecureEnvelope) error {
	return nil
}

func (s *Store) RetrieveSecureEnvelope(ctx context.Context, txID, envID ulid.ULID) (*models.SecureEnvelope, error) {
	return nil, nil
}

func (s *Store) UpdateSecureEnvelope(context.Context, *models.SecureEnvelope) error {
	return nil
}

func (s *Store) DeleteSecureEnvelope(ctx context.Context, txID, envID ulid.ULID) error {
	return nil
}

func (s *Store) ListAccounts(context.Context, *models.PageInfo) (*models.AccountsPage, error) {
	return nil, nil
}

func (s *Store) CreateAccount(context.Context, *models.Account) error {
	return nil
}

func (s *Store) RetrieveAccount(ctx context.Context, id ulid.ULID) (*models.Account, error) {
	return nil, nil
}

func (s *Store) UpdateAccount(context.Context, *models.Account) error {
	return nil
}

func (s *Store) DeleteAccount(ctx context.Context, id ulid.ULID) error {
	return nil
}

func (s *Store) ListCryptoAddresses(ctx context.Context, accountID ulid.ULID, page *models.PageInfo) (*models.CryptoAddressPage, error) {
	return nil, nil
}

func (s *Store) CreateCryptoAddress(context.Context, *models.CryptoAddress) error {
	return nil
}

func (s *Store) RetrieveCryptoAddress(ctx context.Context, accountID, cryptoAddressID ulid.ULID) (*models.CryptoAddress, error) {
	return nil, nil
}

func (s *Store) UpdateCryptoAddress(context.Context, *models.CryptoAddress) error {
	return nil
}

func (s *Store) DeleteCryptoAddress(ctx context.Context, accountID, cryptoAddressID ulid.ULID) error {
	return nil
}

func (s *Store) ListCounterparties(ctx context.Context, page *models.PageInfo) (*models.CounterpartyPage, error) {
	return nil, nil
}

func (s *Store) ListCounterpartySourceInfo(ctx context.Context, source string) ([]*models.CounterpartySourceInfo, error) {
	return nil, nil
}

func (s *Store) CreateCounterparty(context.Context, *models.Counterparty) error {
	return nil
}

func (s *Store) RetrieveCounterparty(ctx context.Context, counterpartyID ulid.ULID) (*models.Counterparty, error) {
	return nil, nil
}

func (s *Store) UpdateCounterparty(context.Context, *models.Counterparty) error {
	return nil
}

func (s *Store) DeleteCounterparty(ctx context.Context, counterpartyID ulid.ULID) error {
	return nil
}
