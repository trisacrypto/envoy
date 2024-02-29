package mock

import "self-hosted-node/pkg/store/dsn"

// Store implements the store.Store interface implemented as an in-memory mock
// interface for testing and development purposes.
type Store struct{}

func Open(uri *dsn.DSN) (*Store, error) {
	return nil, nil
}

func (s *Store) Close() error {
	return nil
}
