package sqlite_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/trisacrypto/envoy/pkg/store/dsn"
	dberr "github.com/trisacrypto/envoy/pkg/store/errors"
	db "github.com/trisacrypto/envoy/pkg/store/sqlite"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/trisacrypto/trisa/pkg/ivms101"
	trisa "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
	generic "github.com/trisacrypto/trisa/pkg/trisa/data/generic/v1beta1"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestConnectClose(t *testing.T) {
	t.Run("ReadWrite", func(t *testing.T) {
		uri, _ := dsn.Parse("sqlite3:///" + filepath.Join(t.TempDir(), "test.db"))

		store, err := db.Open(uri)
		require.NoError(t, err, "could not open connection to temporary sqlite database")

		tx, err := store.BeginTx(context.Background(), nil)
		require.NoError(t, err, "could not create write transaction")
		tx.Rollback()

		tx, err = store.BeginTx(context.Background(), &sql.TxOptions{ReadOnly: true})
		require.NoError(t, err, "could not create readonly transaction")
		tx.Rollback()

		err = store.Close()
		require.NoError(t, err, "should be able to close the db without error when not connected")
	})

	t.Run("ReadOnly", func(t *testing.T) {
		uri, _ := dsn.Parse("sqlite3:///" + filepath.Join(t.TempDir(), "test.db") + "?readonly=true")

		store, err := db.Open(uri)
		require.NoError(t, err, "could not open connection to temporary sqlite database")

		tx, err := store.BeginTx(context.Background(), &sql.TxOptions{ReadOnly: false})
		require.ErrorIs(t, err, dberr.ErrReadOnly, "created write transaction in readonly mode")
		require.Nil(t, tx, "expected no transaction to be returned")

		tx, err = store.BeginTx(context.Background(), &sql.TxOptions{ReadOnly: true})
		require.NoError(t, err, "could not create readonly transaction")
		tx.Rollback()

		err = store.Close()
		require.NoError(t, err, "should be able to close the db without error when not connected")
	})

	t.Run("Failures", func(t *testing.T) {
		tests := []struct {
			uri *dsn.DSN
			err error
		}{
			{
				&dsn.DSN{Scheme: "leveldb"},
				dberr.ErrUnknownScheme,
			},
			{
				&dsn.DSN{Scheme: "sqlite3"},
				dberr.ErrPathRequired,
			},
		}

		for i, tc := range tests {
			_, err := db.Open(tc.uri)
			require.ErrorIs(t, err, tc.err, "test case %d failed", i)
		}
	})
}

type storeTestSuite struct {
	suite.Suite
	dbpath string
	store  *db.Store
}

func (s *storeTestSuite) SetupSuite() {
	s.CreateDB()
}

func (s *storeTestSuite) CreateDB() {
	var err error
	require := s.Assert()

	// Only create the database path on the first call to CreateDB. Otherwise the call
	// to TempDir() will be prefixed with the name of the subtest, which will cause an
	// "attempt to write a read-only database" for subsequent tests because the directory
	// will be deleted when the subtest is complete.
	if s.dbpath == "" {
		s.dbpath = filepath.Join(s.T().TempDir(), "envoytests.db")
	}

	uri, _ := dsn.Parse("sqlite3:///" + s.dbpath)
	s.store, err = db.Open(uri)
	require.NoError(err, "could not open store in temporary location")

	// Execute any SQL files in the testdata directory
	paths, err := filepath.Glob("testdata/*.sql")
	require.NoError(err, "could not list testdata directory")

	tx, err := s.store.BeginTx(context.Background(), nil)
	require.NoError(err, "could not open transaction")
	defer tx.Rollback()

	for _, path := range paths {
		stmt, err := os.ReadFile(path)
		require.NoError(err, "could not read query from file")

		_, err = tx.Exec(string(stmt))
		require.NoError(err, "could not execute sql query from fixture %s", path)
	}

	require.NoError(tx.Commit(), "could not commit transaction")
}

func (s *storeTestSuite) ResetDB() {
	require := s.Require()
	require.NoError(s.store.Close(), "could not close connection to db")
	require.NoError(os.Remove(s.dbpath), "could not delete old database")
	s.CreateDB()
}

func TestStore(t *testing.T) {
	suite.Run(t, new(storeTestSuite))
}

//===========================================================================
// Helper Functions
//===========================================================================

func loadPayload(identityPath, transactionPath string) (payload *trisa.Payload, err error) {
	payload = &trisa.Payload{}

	payload.Identity, _ = anypb.New(&ivms101.IdentityPayload{})
	if err = loadFixture(identityPath, payload.Identity); err != nil {
		return nil, err
	}

	payload.Transaction, _ = anypb.New(&generic.Transaction{})
	if err = loadFixture(transactionPath, payload.Transaction); err != nil {
		return nil, err
	}

	payload.SentAt = time.Now().UTC().Format(time.RFC3339)
	return payload, nil
}

func loadFixture(path string, obj proto.Message) (err error) {
	var data []byte
	if data, err = os.ReadFile(path); err != nil {
		return err
	}

	json := protojson.UnmarshalOptions{
		AllowPartial:   true,
		DiscardUnknown: true,
	}

	return json.Unmarshal(data, obj)
}
