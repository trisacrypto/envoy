package store

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/trisacrypto/envoy/pkg/store/dsn"
	"github.com/trisacrypto/envoy/pkg/store/mock"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/envoy/pkg/store/secrets"
	"github.com/trisacrypto/envoy/pkg/store/sqlite"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
)

// Open a directory storage provider with the specified URI. Database URLs should either
// specify protocol+transport://user:pass@host/dbname?opt1=a&opt2=b for servers or
// protocol:///relative/path/to/file for embedded databases (for absolute paths, specify
// protocol:////absolute/path/to/file).
func Open(databaseURL string) (s Store, err error) {
	var uri *dsn.DSN
	if uri, err = dsn.Parse(databaseURL); err != nil {
		return nil, err
	}

	switch uri.Scheme {
	case dsn.Mock:
		return mock.Open(uri)
	case dsn.SQLite, dsn.SQLite3:
		return sqlite.Open(uri)
	default:
		return nil, fmt.Errorf("unhandled database scheme %q", uri.Scheme)
	}
}

// Store is a generic storage interface allowing multiple storage backends such as
// SQLite or Postgres to be used based on the preference of the user.
type Store interface {
	io.Closer
	TransactionStore
	AccountStore
	CounterpartyStore
	UserStore
	APIKeyStore
}

// Secrets is a generic storage interface for storing secrets such as private key
// material for identity and sealing certificates. It is separate from the generic store
// since storage needs to be specialized for security.
type Secrets interface {
	io.Closer
	ListSecrets(ctx context.Context, namespace string) (secrets.Iterator, error)
	CreateSecret(context.Context, *secrets.Secret) error
	RetrieveSecret(context.Context, *secrets.Secret) error
	DeleteSecret(context.Context, *secrets.Secret) error
}

// All Store implementations must implement the Store interface
var (
	_ Store   = &mock.Store{}
	_ Store   = &sqlite.Store{}
	_ Secrets = &secrets.GCP{}
)

// TransactionStore stores some lightweight information about specific transactions
// stored in the database (most of which is not sensitive and is used for indexing).
// It also maintains an association with all secure envelopes sent and received as
// part of completing a travel rule exchange for the transaction.
type TransactionStore interface {
	SecureEnvelopeStore
	ListTransactions(context.Context, *models.PageInfo) (*models.TransactionPage, error)
	CreateTransaction(context.Context, *models.Transaction) error
	RetrieveTransaction(context.Context, uuid.UUID) (*models.Transaction, error)
	UpdateTransaction(context.Context, *models.Transaction) error
	DeleteTransaction(context.Context, uuid.UUID) error
	PrepareTransaction(context.Context, uuid.UUID) (models.PreparedTransaction, error)
	LatestSecureEnvelope(ctx context.Context, txID uuid.UUID, direction string) (*models.SecureEnvelope, error)
}

// SecureEnvelopes are associated with individual transactions.
type SecureEnvelopeStore interface {
	ListSecureEnvelopes(ctx context.Context, txID uuid.UUID, page *models.PageInfo) (*models.SecureEnvelopePage, error)
	CreateSecureEnvelope(context.Context, *models.SecureEnvelope) error
	RetrieveSecureEnvelope(ctx context.Context, txID uuid.UUID, envID ulid.ULID) (*models.SecureEnvelope, error)
	UpdateSecureEnvelope(context.Context, *models.SecureEnvelope) error
	DeleteSecureEnvelope(ctx context.Context, txID uuid.UUID, envID ulid.ULID) error
}

// AccountStore provides CRUD interactions with Account models.
type AccountStore interface {
	TravelAddressStore
	CryptoAddressStore
	ListAccounts(ctx context.Context, page *models.PageInfo) (*models.AccountsPage, error)
	CreateAccount(context.Context, *models.Account) error
	RetrieveAccount(ctx context.Context, id ulid.ULID) (*models.Account, error)
	UpdateAccount(context.Context, *models.Account) error
	DeleteAccount(ctx context.Context, id ulid.ULID) error
}

// CryptoAddressStore provides CRUD interactions with CryptoAddress models and their
// associated Account model.
type CryptoAddressStore interface {
	TravelAddressStore
	ListCryptoAddresses(ctx context.Context, accountID ulid.ULID, page *models.PageInfo) (*models.CryptoAddressPage, error)
	CreateCryptoAddress(context.Context, *models.CryptoAddress) error
	RetrieveCryptoAddress(ctx context.Context, accountID, cryptoAddressID ulid.ULID) (*models.CryptoAddress, error)
	UpdateCryptoAddress(context.Context, *models.CryptoAddress) error
	DeleteCryptoAddress(ctx context.Context, accountID, cryptoAddressID ulid.ULID) error
}

// Counterparty store provides CRUD interactions with Counterparty models.
type CounterpartyStore interface {
	ListCounterparties(ctx context.Context, page *models.PageInfo) (*models.CounterpartyPage, error)
	ListCounterpartySourceInfo(ctx context.Context, source string) ([]*models.CounterpartySourceInfo, error)
	CreateCounterparty(context.Context, *models.Counterparty) error
	RetrieveCounterparty(ctx context.Context, counterpartyID ulid.ULID) (*models.Counterparty, error)
	LookupCounterparty(ctx context.Context, commonName string) (*models.Counterparty, error)
	UpdateCounterparty(context.Context, *models.Counterparty) error
	DeleteCounterparty(ctx context.Context, counterpartyID ulid.ULID) error
}

type TravelAddressStore interface {
	UseTravelAddressFactory(models.TravelAddressFactory)
}

type UserStore interface {
	ListUsers(ctx context.Context, page *models.PageInfo) (*models.UserPage, error)
	CreateUser(context.Context, *models.User) error
	RetrieveUser(ctx context.Context, emailOrUserID any) (*models.User, error)
	UpdateUser(context.Context, *models.User) error
	SetUserPassword(ctx context.Context, userID ulid.ULID, password string) error
	SetUserLastLogin(ctx context.Context, userID ulid.ULID, lastLogin time.Time) error
	DeleteUser(ctx context.Context, userID ulid.ULID) error
	LookupRole(ctx context.Context, role string) (*models.Role, error)
}

type APIKeyStore interface {
	ListAPIKeys(context.Context, *models.PageInfo) (*models.APIKeyPage, error)
	CreateAPIKey(context.Context, *models.APIKey) error
	RetrieveAPIKey(ctx context.Context, clientIDOrKeyID any) (*models.APIKey, error)
	UpdateAPIKey(context.Context, *models.APIKey) error
	DeleteAPIKey(ctx context.Context, keyID ulid.ULID) error
}
