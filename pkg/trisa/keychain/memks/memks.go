package memks

import (
	"bytes"
	"sync"
	"time"

	"self-hosted-node/pkg/trisa/keychain/keyerr"

	"github.com/trisacrypto/trisa/pkg/trisa/keys"
)

// New creates a new MemStore that implements the keychain.KeyStore interface.
func New() (*MemStore, error) {
	return &MemStore{
		cache: make(map[string]*CachedKey),
	}, nil
}

// MemStore implements an in-memory KeyStore that caches keys in volatile memory. This
// key store can also be effectively used for testing without any on disk fixtures.
type MemStore struct {
	sync.RWMutex
	cache map[string]*CachedKey
}

// CachedKey is an internal data structure that holds a key pair and a timestamp.
type CachedKey struct {
	Key       keys.Key
	Timestamp time.Time
}

// Get a public sealing key and the timestamp it was cached by signature.
func (s *MemStore) Get(signature string) (keys.Key, time.Time, error) {
	s.RLock()
	defer s.RUnlock()
	if ckey, ok := s.cache[signature]; ok {
		return ckey.Key, ckey.Timestamp, nil
	}
	return nil, time.Time{}, keyerr.KeyNotFound
}

// Put a public sealing key into the cache. If a private key has already been added with
// the same signature, the cached key is updated with the corresponding public key. It
// is the caller's responsibility to ensure that the signature for both the private and
// the public key match.
func (s *MemStore) Put(key keys.Key) (err error) {
	var signature string
	if signature, err = key.PublicKeySignature(); err != nil {
		return err
	}

	s.Lock()
	defer s.Unlock()
	if cached, ok := s.cache[signature]; ok {
		// Do not allow overwriting of different keys
		var samekey bool
		if samekey, err = equalKeys(key, cached.Key); err != nil {
			return err
		}

		if !samekey {
			return keyerr.KeyOverwrite
		}

		// Update the timestamp of the cached value
		cached.Key = key
		cached.Timestamp = time.Now()
		return nil
	}

	s.cache[signature] = &CachedKey{
		Key:       key,
		Timestamp: time.Now(),
	}
	return nil
}

func (s *MemStore) Delete(signature string) error {
	s.Lock()
	defer s.Unlock()
	delete(s.cache, signature)
	return nil
}

// Prevent overwriting keys by comparing marshaled data
func equalKeys(a, b keys.Key) (_ bool, err error) {
	var adata []byte
	if adata, err = a.Marshal(); err != nil {
		return false, keyerr.KeysNotComparable
	}

	var bdata []byte
	if bdata, err = b.Marshal(); err != nil {
		return false, keyerr.KeysNotComparable
	}

	return bytes.Equal(adata, bdata), nil
}
