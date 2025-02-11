package mock

import (
	"context"
	"time"

	"github.com/trisacrypto/envoy/pkg/store/dsn"
	"github.com/trisacrypto/envoy/pkg/store/models"

	"github.com/google/uuid"
	"go.rtnl.ai/ulid"
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

func (s *Store) RetrieveTransaction(context.Context, uuid.UUID) (*models.Transaction, error) {
	return nil, nil
}

func (s *Store) UpdateTransaction(context.Context, *models.Transaction) error {
	return nil
}

func (s *Store) DeleteTransaction(context.Context, uuid.UUID) error {
	return nil
}

func (s *Store) ArchiveTransaction(context.Context, uuid.UUID) error {
	return nil
}

func (s *Store) PrepareTransaction(context.Context, uuid.UUID) (models.PreparedTransaction, error) {
	return nil, nil
}

func (s *Store) ListSecureEnvelopes(ctx context.Context, txID uuid.UUID, page *models.PageInfo) (*models.SecureEnvelopePage, error) {
	return nil, nil
}

func (s *Store) CreateSecureEnvelope(context.Context, *models.SecureEnvelope) error {
	return nil
}

func (s *Store) RetrieveSecureEnvelope(ctx context.Context, txID uuid.UUID, envID ulid.ULID) (*models.SecureEnvelope, error) {
	return nil, nil
}

func (s *Store) UpdateSecureEnvelope(context.Context, *models.SecureEnvelope) error {
	return nil
}

func (s *Store) DeleteSecureEnvelope(ctx context.Context, txID uuid.UUID, envID ulid.ULID) error {
	return nil
}

func (s *Store) LatestSecureEnvelope(ctx context.Context, txID uuid.UUID, direction string) (*models.SecureEnvelope, error) {
	return nil, nil
}

func (s *Store) LatestPayloadEnvelope(ctx context.Context, txID uuid.UUID, direction string) (*models.SecureEnvelope, error) {
	return nil, nil
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

func (s *Store) SearchCounterparties(ctx context.Context, query *models.SearchQuery) (out *models.CounterpartyPage, err error) {
	return out, nil
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

func (s *Store) LookupCounterparty(ctx context.Context, field, value string) (*models.Counterparty, error) {
	return nil, nil
}

func (s *Store) UpdateCounterparty(context.Context, *models.Counterparty) error {
	return nil
}

func (s *Store) DeleteCounterparty(ctx context.Context, counterpartyID ulid.ULID) error {
	return nil
}

func (s *Store) ListContacts(ctx context.Context, counterpartyID ulid.ULID, page *models.PageInfo) (*models.ContactsPage, error) {
	return nil, nil
}

func (s *Store) CreateContact(context.Context, *models.Contact) error {
	return nil
}

func (s *Store) RetrieveContact(ctx context.Context, contactID, counterpartyID ulid.ULID) (*models.Contact, error) {
	return nil, nil
}

func (s *Store) UpdateContact(context.Context, *models.Contact) error {
	return nil
}

func (s *Store) DeleteContact(ctx context.Context, contactID, counterpartyID ulid.ULID) error {
	return nil
}

func (s *Store) UseTravelAddressFactory(models.TravelAddressFactory) {
}

func (s *Store) ListSunrise(ctx context.Context, page *models.PageInfo) (out *models.SunrisePage, err error) {
	return nil, nil
}

func (s *Store) CreateSunrise(ctx context.Context, msg *models.Sunrise) (err error) {
	return nil
}

func (s *Store) RetrieveSunrise(ctx context.Context, id ulid.ULID) (msg *models.Sunrise, err error) {
	return nil, nil
}

func (s *Store) UpdateSunrise(ctx context.Context, msg *models.Sunrise) (err error) {
	return nil
}

func (s *Store) DeleteSunrise(ctx context.Context, id ulid.ULID) (err error) {
	return nil
}

func (s *Store) GetOrCreateSunriseCounterparty(ctx context.Context, email, name string) (*models.Counterparty, error) {
	return nil, nil
}

func (s *Store) ListUsers(ctx context.Context, page *models.PageInfo) (*models.UserPage, error) {
	return nil, nil
}

func (s *Store) CreateUser(context.Context, *models.User) error {
	return nil
}

func (s *Store) RetrieveUser(ctx context.Context, emailOrUserID any) (*models.User, error) {
	return nil, nil
}

func (s *Store) UpdateUser(context.Context, *models.User) error {
	return nil
}

func (s *Store) SetUserPassword(ctx context.Context, userID ulid.ULID, password string) (err error) {
	return nil
}

func (s *Store) SetUserLastLogin(ctx context.Context, userID ulid.ULID, lastLogin time.Time) (err error) {
	return nil
}

func (s *Store) DeleteUser(ctx context.Context, userID ulid.ULID) error {
	return nil
}

func (s *Store) LookupRole(ctx context.Context, role string) (*models.Role, error) {
	return nil, nil
}

func (s *Store) ListAPIKeys(context.Context, *models.PageInfo) (*models.APIKeyPage, error) {
	return nil, nil
}

func (s *Store) CreateAPIKey(context.Context, *models.APIKey) error {
	return nil
}

func (s *Store) RetrieveAPIKey(ctx context.Context, clientIDOrKeyID any) (*models.APIKey, error) {
	return nil, nil
}

func (s *Store) UpdateAPIKey(context.Context, *models.APIKey) error {
	return nil
}

func (s *Store) DeleteAPIKey(ctx context.Context, keyID ulid.ULID) error {
	return nil
}
