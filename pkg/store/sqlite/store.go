package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/trisacrypto/envoy/pkg/store/dsn"
	"github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/models"

	_ "github.com/mattn/go-sqlite3"
)

// Store implements the store.Store interface using SQLite3 as the storage backend.
type Store struct {
	readonly bool
	conn     *sql.DB
	mkta     models.TravelAddressFactory
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

func (s *Store) UseTravelAddressFactory(f models.TravelAddressFactory) {
	s.mkta = f
}

func (s *Store) Stats() sql.DBStats {
	return s.conn.Stats()
}
