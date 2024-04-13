package sqlite_test

import (
	"context"
	"path/filepath"
	"testing"

	"self-hosted-node/pkg/store/dsn"
	db "self-hosted-node/pkg/store/sqlite"
	"self-hosted-node/pkg/web/auth/permissions"

	"github.com/stretchr/testify/require"
)

func TestPermissions(t *testing.T) {
	uri, _ := dsn.Parse("sqlite3:///" + filepath.Join(t.TempDir(), "test.db"))

	store, err := db.Open(uri)
	require.NoError(t, err, "could not open connection to temporary sqlite database")
	t.Cleanup(func() { store.Close() })

	tx, err := store.BeginTx(context.Background(), nil)
	require.NoError(t, err, "could not begin transaction")
	t.Cleanup(func() { tx.Rollback() })

	rows, err := tx.Query("SELECT id, title FROM permissions")
	require.NoError(t, err, "could not execute query")
	t.Cleanup(func() { rows.Close() })

	for rows.Next() {
		// Validate that we can convert the ID and the title into a Permission and that
		// the string matches the constant in the permissions package.
		var (
			pk    int64
			title string
		)

		err := rows.Scan(&pk, &title)
		require.NoError(t, err, "could not scan pk and title")

		pkPermission, err := permissions.Parse(pk)
		require.NoError(t, err, "could not parse primary key %d", pk)
		require.Equal(t, title, pkPermission.String(), "string constant and database mismatch")

		titlePermission, err := permissions.Parse(title)
		require.NoError(t, err, "could not parse title %q", title)
		require.Equal(t, pkPermission, titlePermission, "mismatch pk and title permission parsing")
	}

	require.NoError(t, rows.Err(), "error iterating over rows")
}
