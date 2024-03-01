package store

import (
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
	CryptoAddressStore
}

// All Store implementations must implement the Store interface
var (
	_ Store = &mock.Store{}
	_ Store = &sqlite.Store{}
)

// AccountStore provides CRUD interactions with Account models.
type AccountStore interface {
	ListAccounts(page *models.PageInfo) (*models.AccountsPage, error)
	CreateAccount(*models.Account) error
	RetrieveAccount(id ulid.ULID) (*models.Account, error)
	UpdateAccount(*models.Account) error
	DeleteAccount(id ulid.ULID) error
}

// CryptoAddressStore provides CRUD interactions with CryptoAddress models.
type CryptoAddressStore interface {
	ListCryptoAddresses(page *models.PageInfo) (*models.CryptoAddressPage, error)
	CreateCryptoAddress(*models.CryptoAddress) error
	RetrieveCryptoAddress(id ulid.ULID) (*models.CryptoAddress, error)
	UpdateCryptoAddress(*models.CryptoAddress) error
	DeleteCryptoAddress(id ulid.ULID) error
}
