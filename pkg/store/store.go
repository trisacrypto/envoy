package store

import (
	"context"
	"fmt"
	"io"
	"self-hosted-node/pkg/store/dsn"
	"self-hosted-node/pkg/store/mock"
	"self-hosted-node/pkg/store/models"
	"self-hosted-node/pkg/store/sqlite"

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
	AccountStore
	CounterpartyStore
}

// All Store implementations must implement the Store interface
var (
	_ Store = &mock.Store{}
	_ Store = &sqlite.Store{}
)

// AccountStore provides CRUD interactions with Account models.
type AccountStore interface {
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
	ListCryptoAddresses(ctx context.Context, accountID ulid.ULID, page *models.PageInfo) (*models.CryptoAddressPage, error)
	CreateCryptoAddress(context.Context, *models.CryptoAddress) error
	RetrieveCryptoAddress(ctx context.Context, accountID, cryptoAddressID ulid.ULID) (*models.CryptoAddress, error)
	UpdateCryptoAddress(context.Context, *models.CryptoAddress) error
	DeleteCryptoAddress(ctx context.Context, accountID, cryptoAddressID ulid.ULID) error
}

// Counterparty store provides CRUD interactions with Counterparty models.
type CounterpartyStore interface {
	ListCounterparties(ctx context.Context, page *models.PageInfo) (*models.CounterpartyPage, error)
	CreateCounterparty(context.Context, *models.Counterparty) error
	RetrieveCounterparty(ctx context.Context, counterpartyID ulid.ULID) (*models.Counterparty, error)
	UpdateCounterparty(context.Context, *models.Counterparty) error
	DeleteCounterparty(ctx context.Context, counterpartyID ulid.ULID) error
}
