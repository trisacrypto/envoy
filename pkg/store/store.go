package store

import (
	"fmt"
	"io"
	"self-hosted-node/pkg/store/dsn"
	"self-hosted-node/pkg/store/mock"
	"self-hosted-node/pkg/store/sqlite"
)

// Open a directory storage provider with the specified URI. Database URLs should either
// specify protocol+transport://user:pass@host/dbname?opt1=a&opt2=b for servers or
// protocol:///relative/path/to/file for embedded databases (for absolute paths, specify
// protocol:////absolute/path/to/file).
func Open(databaseURL string) (s Store, err error) {
	var uri *dsn.DSN
	if uri, err = dsn.Parse(databaseURL); err != nil {
		return nil, err
	}

	switch uri.Scheme {
	case dsn.Mock:
		return mock.Open(uri)
	case dsn.SQLite, dsn.SQLite3:
		return sqlite.Open(uri)
	default:
		return nil, fmt.Errorf("unhandled database scheme %q", uri.Scheme)
	}
}

type Store interface {
	io.Closer
}

// All Store implementations must implement the Store interface
var (
	_ Store = &mock.Store{}
	_ Store = &sqlite.Store{}
)
