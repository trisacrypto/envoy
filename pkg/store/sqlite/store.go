package sqlite

import (
	"context"
	"crypto/rsa"
	"database/sql"
	"fmt"
	"os"

	"github.com/trisacrypto/envoy/pkg/store/dsn"
	"github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/envoy/pkg/store/txn"
	"github.com/trisacrypto/envoy/pkg/trisa/keychain"
	"github.com/trisacrypto/trisa/pkg/trisa/crypto/rsaoeap"
	"github.com/trisacrypto/trisa/pkg/trisa/keys"

	_ "github.com/mattn/go-sqlite3"
)

// Store implements the store.Store interface using SQLite3 as the storage backend.
type Store struct {
	readonly bool
	conn     *sql.DB
	mkta     models.TravelAddressFactory
	kc       *keychain.KeyChain
}

// Tx implements the store.Tx interface using SQLite3 as the storage backend.
type Tx struct {
	tx   *sql.Tx
	opts *sql.TxOptions
	mkta models.TravelAddressFactory
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

	return &Tx{tx: tx, opts: opts, mkta: s.mkta}, nil
}

func (s *Store) UseTravelAddressFactory(f models.TravelAddressFactory) {
	s.mkta = f
}

func (s *Store) Stats() sql.DBStats {
	return s.conn.Stats()
}

//===========================================================================
// KeyChain and related functions
//===========================================================================

// The store will use the given keychain.KeyChain for signing and verification
// of ComplianceAuditLog entries.
func (s *Store) UseKeyChain(kc *keychain.KeyChain) {
	s.kc = kc
}

// Returns a cryptographic signature for the input data.
func (s *Store) Sign(data []byte) (signature []byte, err error) {
	var signer *rsaoeap.RSA
	if signer, err = s.getSigner(); err != nil {
		return nil, err
	}
	return signer.Sign(data)
}

// Returns the signature of the current verification key.
func (s *Store) VerificationKeySignature() (signature string, err error) {
	var verifier *rsaoeap.RSA
	if verifier, err = s.getVerifier(""); err != nil {
		return "", err
	}
	return verifier.PublicKeySignature()
}

// Returns the algorithm details as a string for the current data-signing
// algorithm.
func (s *Store) SignatureAlgorithm() string {
	// NOTE: in order to not have to return an error, we're bypassing the
	// SignatureAlgorithm() function so we don't have to init an RSA.
	return rsaoeap.SignerAlgorithm
}

// Verifies the input data versus the input dataSignature using the verification
// key identified by keySignature. If keySignature is the empty string, the
// default local node's verification key will be used. The data is valid if no
// error is returned.
func (s *Store) Verify(data, dataSignature []byte, keySignature string) (err error) {
	var verifier *rsaoeap.RSA
	if verifier, err = s.getVerifier(keySignature); err != nil {
		return err
	}
	return verifier.Verify(data, dataSignature)
}

// Returns an rsaoeap.RSA that can be used to sign data.
func (s *Store) getSigner() (rsaSigner *rsaoeap.RSA, err error) {
	var (
		privkey      keys.PrivateKey
		interfacekey any
		rsaPrivkey   *rsa.PrivateKey
		ok           bool
	)

	// Get the singing key
	if privkey, err = (*s.kc).SigningKey(); err != nil {
		return nil, err
	}

	// Get the unsealing key interface key
	if interfacekey, err = privkey.UnsealingKey(); err != nil {
		return nil, err
	}

	// Assert type of key is an RSA private key
	if rsaPrivkey, ok = interfacekey.(*rsa.PrivateKey); !ok {
		return nil, errors.ErrSigningKeyMissing
	}

	// Get the RSA interface to sign with
	if rsaSigner, err = rsaoeap.New(rsaPrivkey); err != nil {
		return nil, err
	}

	return rsaSigner, nil
}

// Returns an rsaoeap.RSA that can be used to verify signatures for a specific
// verification key. If signature input is the empty string, will use the local
// node's current verification key.
func (s *Store) getVerifier(signature string) (rsaVerifier *rsaoeap.RSA, err error) {
	var (
		pubkey       keys.PublicKey
		interfacekey any
		rsaPubkey    *rsa.PublicKey
		ok           bool
	)

	// Get the singing key
	if pubkey, err = (*s.kc).VerificationKey(signature); err != nil {
		return nil, err
	}

	// Get the unsealing key interface key
	if interfacekey, err = pubkey.SealingKey(); err != nil {
		return nil, err
	}

	// Assert type of key is an RSA private key
	if rsaPubkey, ok = interfacekey.(*rsa.PublicKey); !ok {
		return nil, errors.ErrVerificationKeyMissing
	}

	// Get the RSA interface to sign with
	if rsaVerifier, err = rsaoeap.New(rsaPubkey); err != nil {
		return nil, err
	}

	return rsaVerifier, nil
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

func (t *Tx) Query(query string, args ...any) (*sql.Rows, error) {
	return t.tx.Query(query, args...)
}

func (t *Tx) QueryRow(query string, args ...any) *sql.Row {
	return t.tx.QueryRow(query, args...)
}

func (t *Tx) Exec(query string, args ...any) (sql.Result, error) {
	return t.tx.Exec(query, args...)
}
