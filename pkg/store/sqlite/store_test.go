package sqlite_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/trisacrypto/envoy/pkg/audit"
	"github.com/trisacrypto/envoy/pkg/enum"
	"github.com/trisacrypto/envoy/pkg/store/dsn"
	dberr "github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/models"
	db "github.com/trisacrypto/envoy/pkg/store/sqlite"
	"github.com/trisacrypto/envoy/pkg/trisa/keychain"
	"go.rtnl.ai/ulid"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/trisacrypto/trisa/pkg/ivms101"
	trisa "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
	generic "github.com/trisacrypto/trisa/pkg/trisa/data/generic/v1beta1"
	"github.com/trisacrypto/trisa/pkg/trisa/keys"
	"github.com/trisacrypto/trisa/pkg/trust"

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

//===========================================================================
// Store Test Suite
//===========================================================================

type storeTestSuite struct {
	suite.Suite
	dbpath string
	store  *db.Store
}

func (s *storeTestSuite) SetupSuite() {
	s.CreateDB()
	loadAuditKeyChainFixture(s.T())
}

// Reset the DB before every test.
func (s *storeTestSuite) SetupTest() {
	s.ResetDB()
}

// Reset the DB before every subtest.
func (s *storeTestSuite) SetupSubtest() {
	s.ResetDB()
}

func (s *storeTestSuite) CreateDB() {
	var err error
	require := s.Require()

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

func loadAuditKeyChainFixture(t *testing.T) {
	// Load Certificate fixture with private keys
	sz, err := trust.NewSerializer(false)
	require.NoError(t, err, "could not create serializer to load fixture")

	provider, err := sz.ReadFile("testdata/certs.pem")
	require.NoError(t, err, "could not read test fixture")

	certs, err := keys.FromProvider(provider)
	require.NoError(t, err, "could not create Key from provider")
	require.True(t, certs.IsPrivate(), "expected test certs fixture to be private")

	// Setup a mock KeyChain
	kc, err := keychain.New(keychain.WithCacheDuration(1*time.Hour), keychain.WithDefaultKey(certs))
	require.NoError(t, err, "could not create a KeyChain")
	audit.UseKeyChain(kc)
}

// Returns a context.Background() with ActorID and ActorType context values for
// audit log testing. The ActorID is a fresh, random ULID and the type is
// an enum.ActorAPIKey.
func (s *storeTestSuite) ActorContext() context.Context {
	return audit.WithActor(context.Background(), ulid.MakeSecure().Bytes(), enum.ActorAPIKey)
}

// Counts the audit logs (by action and resource type) created recently (1
// hour) and compares those counts to the expected counts. The expected map
// takes a string from the function ActionResourceKey() for indexing.
func (s *storeTestSuite) AssertAuditLogCount(expected map[string]int) bool {
	// setup
	require := s.Require()
	ctx := s.ActorContext()
	pageInfo := &models.ComplianceAuditLogPageInfo{
		After:  time.Now().Add(-1 * time.Hour),
		Before: time.Now(),
	}

	// get logs
	logs, err := s.store.ListComplianceAuditLogs(ctx, pageInfo)
	require.NoError(err, "error getting logs")
	require.NotNil(logs, "logs was nil")

	// count logs
	actual := make(map[string]int, len(expected))
	for _, log := range logs.Logs {
		actual[ActionResourceKey(log.Action, log.ResourceType)]++
	}

	// compare
	require.Equal(expected, actual, "audit log count is off")
	return reflect.DeepEqual(expected, actual)
}

// Returns a string that can be used in the expected map for ExpectedAuditLogs().
func ActionResourceKey(a enum.Action, r enum.Resource) string {
	return a.String() + r.String()
}
