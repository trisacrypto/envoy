package dsn_test

import (
	"self-hosted-node/pkg/store/dsn"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	testCases := []struct {
		input    string
		expected *dsn.DSN
		err      error
	}{
		{
			"sqlite3:////data/app.db",
			&dsn.DSN{Scheme: "sqlite3", Path: "/data/app.db"},
			nil,
		},
		{
			"sqlite3:///fixtures/app.db",
			&dsn.DSN{Scheme: "sqlite3", Path: "fixtures/app.db"},
			nil,
		},
		{
			"leveldb:////data/db/",
			&dsn.DSN{Scheme: "leveldb", Path: "/data/db/"},
			nil,
		},
		{
			"leveldb:///fixtures/db/",
			&dsn.DSN{Scheme: "leveldb", Path: "fixtures/db/"},
			nil,
		},
		{
			"sqlite3:////data/app.db?foreign_keys=on",
			&dsn.DSN{Scheme: "sqlite3", Path: "/data/app.db", Options: map[string]string{"foreign_keys": "on"}},
			nil,
		},
		{
			"foo", nil, dsn.ErrInvalidDSN,
		},
		{
			"foo://", nil, dsn.ErrInvalidDSN,
		},
		{
			"cache_object:foo/bar", nil, dsn.ErrDSNParse,
		},
	}

	for i, tc := range testCases {
		actual, err := dsn.Parse(tc.input)

		if tc.err != nil {
			require.ErrorIs(t, err, tc.err, "expected error match on test case %d", i)
			require.Zero(t, actual, "expected empty uri returned on test case %d", i)
		} else {
			require.NoError(t, err, "expected no error on test case %d", i)
			require.Equal(t, tc.expected, actual, "incorrect parse on test case %d", i)
		}
	}
}

func TestString(t *testing.T) {
	testCases := []struct {
		dsn      *dsn.DSN
		expected string
	}{
		{
			&dsn.DSN{Scheme: "sqlite3", Path: "fixtures/app.db"},
			"sqlite3:///fixtures/app.db",
		},
		{
			&dsn.DSN{Scheme: "sqlite3", Path: "/data/app.db"},
			"sqlite3:////data/app.db",
		},
	}

	for i, tc := range testCases {
		require.Equal(t, tc.expected, tc.dsn.String(), "test case %d failed", i)
	}

}
