package store

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"time"

	"github.com/trisacrypto/envoy/pkg/enum"
	"github.com/trisacrypto/envoy/pkg/store/dsn"
	"github.com/trisacrypto/envoy/pkg/store/mock"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/envoy/pkg/store/secrets"
	"github.com/trisacrypto/envoy/pkg/store/sqlite"

	"github.com/google/uuid"
	"go.rtnl.ai/ulid"
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
	ContactStore
	SunriseStore
	UserStore
	APIKeyStore
	ResetPasswordLinkStore
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

// The Stats interface exposes database statistics if it is available from the backend.
type Stats interface {
	Stats() sql.DBStats
}

// TransactionStore stores some lightweight information about specific transactions
// stored in the database (most of which is not sensitive and is used for indexing).
// It also maintains an association with all secure envelopes sent and received as
// part of completing a travel rule exchange for the transaction.
type TransactionStore interface {
	SecureEnvelopeStore
	ListTransactions(context.Context, *models.TransactionPageInfo) (*models.TransactionPage, error)
	CreateTransaction(context.Context, *models.Transaction) error
	RetrieveTransaction(context.Context, uuid.UUID) (*models.Transaction, error)
	UpdateTransaction(context.Context, *models.Transaction) error
	DeleteTransaction(context.Context, uuid.UUID) error
	ArchiveTransaction(context.Context, uuid.UUID) error
	UnarchiveTransaction(context.Context, uuid.UUID) error
	PrepareTransaction(context.Context, uuid.UUID) (models.PreparedTransaction, error)
	CountTransactions(context.Context) (*models.TransactionCounts, error)
	TransactionState(context.Context, uuid.UUID) (archived bool, status enum.Status, err error)
}

// SecureEnvelopes are associated with individual transactions.
type SecureEnvelopeStore interface {
	ListSecureEnvelopes(ctx context.Context, txID uuid.UUID, page *models.PageInfo) (*models.SecureEnvelopePage, error)
	CreateSecureEnvelope(context.Context, *models.SecureEnvelope) error
	RetrieveSecureEnvelope(ctx context.Context, txID uuid.UUID, envID ulid.ULID) (*models.SecureEnvelope, error)
	UpdateSecureEnvelope(context.Context, *models.SecureEnvelope) error
	DeleteSecureEnvelope(ctx context.Context, txID uuid.UUID, envID ulid.ULID) error
	LatestSecureEnvelope(ctx context.Context, txID uuid.UUID, direction enum.Direction) (*models.SecureEnvelope, error)
	LatestPayloadEnvelope(ctx context.Context, txID uuid.UUID, direction enum.Direction) (*models.SecureEnvelope, error)
}

// AccountStore provides CRUD interactions with Account models.
type AccountStore interface {
	TravelAddressStore
	CryptoAddressStore
	ListAccounts(ctx context.Context, page *models.PageInfo) (*models.AccountsPage, error)
	CreateAccount(context.Context, *models.Account) error
	LookupAccount(ctx context.Context, cryptoAddress string) (*models.Account, error)
	RetrieveAccount(ctx context.Context, id ulid.ULID) (*models.Account, error)
	UpdateAccount(context.Context, *models.Account) error
	DeleteAccount(ctx context.Context, id ulid.ULID) error
	ListAccountTransactions(ctx context.Context, accountID ulid.ULID, page *models.TransactionPageInfo) (*models.TransactionPage, error)
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
	SearchCounterparties(ctx context.Context, query *models.SearchQuery) (*models.CounterpartyPage, error)
	ListCounterparties(ctx context.Context, page *models.CounterpartyPageInfo) (*models.CounterpartyPage, error)
	ListCounterpartySourceInfo(ctx context.Context, source enum.Source) ([]*models.CounterpartySourceInfo, error)
	CreateCounterparty(context.Context, *models.Counterparty) error
	RetrieveCounterparty(ctx context.Context, counterpartyID ulid.ULID) (*models.Counterparty, error)
	LookupCounterparty(ctx context.Context, field, value string) (*models.Counterparty, error)
	UpdateCounterparty(context.Context, *models.Counterparty) error
	DeleteCounterparty(ctx context.Context, counterpartyID ulid.ULID) error
}

type ContactStore interface {
	ListContacts(ctx context.Context, counterparty any, page *models.PageInfo) (*models.ContactsPage, error)
	CreateContact(context.Context, *models.Contact) error
	RetrieveContact(ctx context.Context, contactID, counterpartyID any) (*models.Contact, error)
	UpdateContact(context.Context, *models.Contact) error
	DeleteContact(ctx context.Context, contactID, counterpartyID any) error
}

type TravelAddressStore interface {
	UseTravelAddressFactory(models.TravelAddressFactory)
}

// Sunrise store manages both contacts and counterparties.
type SunriseStore interface {
	ListSunrise(context.Context, *models.PageInfo) (*models.SunrisePage, error)
	CreateSunrise(context.Context, *models.Sunrise) error
	RetrieveSunrise(context.Context, ulid.ULID) (*models.Sunrise, error)
	UpdateSunrise(context.Context, *models.Sunrise) error
	UpdateSunriseStatus(context.Context, uuid.UUID, enum.Status) error
	DeleteSunrise(context.Context, ulid.ULID) error
	GetOrCreateSunriseCounterparty(ctx context.Context, email, name string) (*models.Counterparty, error)
}

type UserStore interface {
	ListUsers(ctx context.Context, page *models.UserPageInfo) (*models.UserPage, error)
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

type ResetPasswordLinkStore interface {
	ListResetPasswordLinks(context.Context, *models.PageInfo) (*models.ResetPasswordLinkPage, error)
	CreateResetPasswordLink(context.Context, *models.ResetPasswordLink) error
	RetrieveResetPasswordLink(context.Context, ulid.ULID) (*models.ResetPasswordLink, error)
	UpdateResetPasswordLink(context.Context, *models.ResetPasswordLink) error
	DeleteResetPasswordLink(context.Context, ulid.ULID) error
}

// Methods required for managing Daybreak records in the database. This interface allows
// us to have a single transaction open for a daybreak operation so that with respect
// to a single counterparty we completely create the record or rollback on failure.
//
// NOTE: this is not part of the Store interface since it is not required for the
// server to function but is useful for Daybreak-specific operations.
type DaybreakStore interface {
	ListDaybreak(context.Context) (map[string]*models.CounterpartySourceInfo, error)
	CreateDaybreak(context.Context, *models.Counterparty) error
	UpdateDaybreak(context.Context, *models.Counterparty) error
}
