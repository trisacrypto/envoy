package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"self-hosted-node/pkg/store/dsn"
	"self-hosted-node/pkg/store/errors"
	"self-hosted-node/pkg/store/models"

	_ "github.com/mattn/go-sqlite3"
	"github.com/oklog/ulid/v2"
)

// Store implements the store.Store interface using SQLite3 as the storage backend.
type Store struct {
	readonly bool
	conn     *sql.DB
}

func Open(uri *dsn.DSN) (_ *Store, err error) {
	// Ensure that only SQLite3 connections can be opened.
	if uri.Scheme != dsn.SQLite && uri.Scheme != dsn.SQLite3 {
		return nil, errors.ErrUnknownScheme
	}

	// Require a path in order to open the database connection (no in-memory databases)
	if uri.Path == "" {
		return nil, errors.ErrPathRequired
	}

	// Check if the database file exists, if it doesn't exist it will be created and
	// all migrations will be applied to the database. Otherwise the code will attempt
	// to only apply migrations that have not yet been applied.
	empty := false
	if _, err := os.Stat(uri.Path); os.IsNotExist(err) {
		empty = true
	}

	// Connect to the database
	s := &Store{readonly: uri.ReadOnly}
	if s.conn, err = sql.Open("sqlite3", uri.Path); err != nil {
		return nil, err
	}

	// Ping the database to establish the connection
	if err = s.conn.Ping(); err != nil {
		return nil, err
	}

	// Ensure that foreign key support is turned on by executing a PRAGMA query.
	if _, err = s.conn.Exec("PRAGMA foreign_keys = on"); err != nil {
		return nil, fmt.Errorf("could not enable foreign key support: %w", err)
	}

	// Ensure the schema is initialized
	if err = s.InitializeSchema(empty); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Store) Close() error {
	return s.conn.Close()
}

func (s *Store) BeginTx(ctx context.Context, opts *sql.TxOptions) (tx *sql.Tx, err error) {
	// Ensure the options respect the read-only option specified by the user.
	if opts == nil {
		opts = &sql.TxOptions{ReadOnly: s.readonly}
	} else if s.readonly && !opts.ReadOnly {
		return nil, errors.ErrReadOnly
	}

	// Create a transaction with the specified context.
	return s.conn.BeginTx(ctx, opts)
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
