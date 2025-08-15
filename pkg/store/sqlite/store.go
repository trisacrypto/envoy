package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/trisacrypto/envoy/pkg/audit"
	"github.com/trisacrypto/envoy/pkg/enum"
	"github.com/trisacrypto/envoy/pkg/store/dsn"
	"github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/envoy/pkg/store/txn"

	_ "github.com/mattn/go-sqlite3"
)

// Store implements the store.Store interface using SQLite3 as the storage backend.
type Store struct {
	readonly bool
	conn     *sql.DB
	mkta     models.TravelAddressFactory
}

// Tx implements the store.Tx interface using SQLite3 as the storage backend.
type Tx struct {
	tx   *sql.Tx
	opts *sql.TxOptions
	mkta models.TravelAddressFactory

	// Compliance audit log actor metadata
	actorID   []byte
	actorType enum.Actor
}

//===========================================================================
// Store methods
//===========================================================================

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

func (s *Store) Begin(ctx context.Context, opts *sql.TxOptions) (txn.Txn, error) {
	return s.BeginTx(ctx, opts)
}

func (s *Store) BeginTx(ctx context.Context, opts *sql.TxOptions) (_ *Tx, err error) {
	// Ensure the options respect the read-only option specified by the user.
	if opts == nil {
		opts = &sql.TxOptions{ReadOnly: s.readonly}
	} else if s.readonly && !opts.ReadOnly {
		return nil, errors.ErrReadOnly
	}

	var tx *sql.Tx
	if tx, err = s.conn.BeginTx(ctx, opts); err != nil {
		return nil, err
	}

	// Get the actor's ID and type. By default, use an "unknown" actor so that
	// database transactions do not fail for this reason.
	var (
		actorID   []byte
		actorType enum.Actor
		ok        bool
	)
	if actorID, ok = audit.ActorID(ctx); !ok {
		actorID = []byte("unknown")
	}
	if actorType, ok = audit.ActorType(ctx); !ok {
		actorType = enum.ActorUnknown
	}

	return &Tx{
		tx:        tx,
		opts:      opts,
		mkta:      s.mkta,
		actorID:   actorID,
		actorType: actorType,
	}, nil
}

func (s *Store) UseTravelAddressFactory(f models.TravelAddressFactory) {
	s.mkta = f
}

func (s *Store) Stats() sql.DBStats {
	return s.conn.Stats()
}

//===========================================================================
// Tx methods
//===========================================================================

func (t *Tx) Commit() error {
	return t.tx.Commit()
}

func (t *Tx) Rollback() error {
	return t.tx.Rollback()
}

// Sets the actor metadata to be returned by GetActor().
func (t *Tx) SetActor(actorID []byte, actorType enum.Actor) {
	t.actorID = actorID
	t.actorType = actorType
}

// Returns the actor metadata set by SetActor().
func (t *Tx) GetActor() ([]byte, enum.Actor) {
	return t.actorID, t.actorType
}

func (t *Tx) Query(query string, args ...any) (*sql.Rows, error) {
	return t.tx.Query(query, args...)
}

func (t *Tx) QueryRow(query string, args ...any) *sql.Row {
	return t.tx.QueryRow(query, args...)
}

func (t *Tx) Exec(query string, args ...any) (sql.Result, error) {
	return t.tx.Exec(query, args...)
}
