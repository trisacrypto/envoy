package sqlite_test

import (
	"path/filepath"
	"testing"

	"self-hosted-node/pkg/store/dsn"
	db "self-hosted-node/pkg/store/sqlite"

	"github.com/stretchr/testify/require"
)

func TestConnectClose(t *testing.T) {
	uri, _ := dsn.Parse("sqlite3:///" + filepath.Join(t.TempDir(), "test.db"))

	store, err := db.Open(uri)
	require.NoError(t, err, "could not open connection to temporary sqlite database")

	// _, err := db.BeginTx(context.Background(), nil)
	// require.ErrorIs(t, err, db.ErrNotConnected, "should not be able to open a transaction without connecting")

	err = store.Close()
	require.NoError(t, err, "should be able to close the db without error when not connected")
}
