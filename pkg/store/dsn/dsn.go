package dsn

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"self-hosted-node/pkg/store/errors"
)

// Supported schemes by this package.
const (
	SQLite  = "sqlite"
	SQLite3 = "sqlite3"
	LevelDB = "leveldb"
	Mock    = "mock"
)

// Supported options by this package.
const (
	ReadOnly = "readonly"
)

// DSN (data source name) represents the parsed components of an embedded database or
// database management service and is used to easily establish a connection to the db.
// TODO: add support for PostgreSQL and other server databases.
type DSN struct {
	Scheme   string
	Path     string
	ReadOnly bool
	Options  map[string]string
}

// Parse a string based DSN into its constituent parts, creating a structured
// representation that can be used to make an actual database connection.
func Parse(uri string) (out *DSN, err error) {
	dsn, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errors.ErrDSNParse, err)
	}

	if dsn.Scheme == "" || dsn.Path == "" {
		return nil, errors.ErrInvalidDSN
	}

	out = &DSN{
		Scheme: dsn.Scheme,
		Path:   strings.TrimPrefix(dsn.Path, "/"),
	}

	// Add any options represented as query parameters.
	if params := dsn.Query(); len(params) > 0 {
		out.Options = make(map[string]string, len(params))
		for k, v := range params {
			switch strings.ToLower(k) {
			case ReadOnly:
				if out.ReadOnly, err = strconv.ParseBool(v[0]); err != nil {
					return nil, fmt.Errorf("could not parse read only option: %w", err)
				}
			default:
				out.Options[k] = v[0]
			}
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
