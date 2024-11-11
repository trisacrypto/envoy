package sqlite_test

import (
	"testing"

	db "github.com/trisacrypto/envoy/pkg/store/sqlite"

	"github.com/stretchr/testify/require"
)

func TestMigrations(t *testing.T) {
	migrations, err := db.Migrations()
	require.NoError(t, err, "should have been able to load migrations")
	require.GreaterOrEqual(t, len(migrations), 1, "wrong number of migrations, has a migration been added?")

	// The first three migrations should match our fixtures
	expected := []*db.Migration{
		{
			ID:   0,
			Name: "Migrations",
			Path: "0000_migrations.sql",
		},
		{
			ID:   1,
			Name: "Initial Schema",
			Path: "0001_initial_schema.sql",
		},
		{
			ID:   2,
			Name: "Authentication",
			Path: "0002_authentication.sql",
		},
		{
			ID:   3,
			Name: "Default Roles",
			Path: "0003_default_roles.sql",
		},
		{
			ID:   4,
			Name: "Secure Envelope Meta",
			Path: "0004_secure_envelope_meta.sql",
		},
		{
			ID:   5,
			Name: "Sunrise Tokens",
			Path: "0005_sunrise_tokens.sql",
		},
	}

	for i, migration := range migrations {
		if i > len(expected) {
			break
		}

		require.Equal(t, expected[i].ID, migration.ID)
		require.Equal(t, expected[i].Name, migration.Name)
		require.Equal(t, expected[i].Path, migration.Path)

		query, err := migration.SQL()
		require.NoError(t, err, "could not load SQL from the migration")
		require.NotEmpty(t, query, "no SQL was returned for the migration")
	}
}
