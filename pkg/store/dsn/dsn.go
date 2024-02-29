package dsn

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

var (
	ErrDSNParse      = errors.New("could not parse dsn")
	ErrInvalidDSN    = errors.New("could not parse DSN, critical component missing")
	ErrUnknownScheme = errors.New("database scheme not handled by this package")
	ErrPathRequired  = errors.New("a path is required for this database scheme")
)

const (
	SQLite  = "sqlite"
	SQLite3 = "sqlite3"
	LevelDB = "leveldb"
	Mock    = "mock"
)

// DSN (data source name)) represents the parsed components of an embedded database or
// database management service and is used to easily establish a connection to the db.
// TODO: add support for PostgreSQL and other server databases.
type DSN struct {
	Scheme  string
	Path    string
	Options map[string]string
}

// Parse a string based DSN into its constituent parts, creating a structured
// representation that can be used to make an actual database connection.
func Parse(uri string) (out *DSN, err error) {
	dsn, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDSNParse, err)
	}

	if dsn.Scheme == "" || dsn.Path == "" {
		return nil, ErrInvalidDSN
	}

	out = &DSN{
		Scheme: dsn.Scheme,
		Path:   strings.TrimPrefix(dsn.Path, "/"),
	}

	// Add any options represented as query paramaters.
	if params := dsn.Query(); len(params) > 0 {
		out.Options = make(map[string]string, len(params))
		for k, v := range params {
			out.Options[k] = v[0]
		}
	}

	return out, nil
}

func (d *DSN) String() string {
	u := &url.URL{
		Scheme: d.Scheme,
		Path:   "/" + d.Path,
	}
	return u.String()
}
