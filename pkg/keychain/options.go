package keychain

import (
	"time"

	"self-hosted-node/pkg/keychain/keyerr"

	"github.com/trisacrypto/trisa/pkg/trisa/keys"
)

type CacheOption func(*Cache) error

// Specify the internal store to use with the key cache (which is a MemStore by default)
// Note that this option must come before WithSealingKeys or WithDefaultKey otherwise
// the default store will be replaced, overwriting the storage of sealing keys.
func WithInternalStore(store KeyStore) CacheOption {
	return func(cache *Cache) error {
		cache.internal = store
		return nil
	}
}

// Specify the external store to use with the key cache (which is a MemStore by default)
func WithExternalStore(store KeyStore) CacheOption {
	return func(cache *Cache) error {
		cache.external = store
		return nil
	}
}

// Store sealing keys when creating the cache. Note that this option must come after
// WithInternalStore if specifying a different internal store than the default store
// or it will have no effect (e.g. the sealing keys will not be stored).
func WithSealingKeys(keys ...keys.Key) CacheOption {
	return func(cache *Cache) error {
		if cache.internal == nil {
			return keyerr.KeyStoreUnavailable
		}

		for _, key := range keys {
			if err := cache.Store(key, nil); err != nil {
				return err
			}
		}
		return nil
	}
}

// Store default sealing key in the store. Note that this option must come after
// WithInternalStore if specifying a different internal store than the default store
// or it will have no effect (e.g. the sealing keys will not be stored).
func WithDefaultKey(key keys.Key) CacheOption {
	return func(cache *Cache) error {
		if cache.internal == nil {
			return keyerr.KeyStoreUnavailable
		}

		if err := cache.Store(key, &KeyOptions{IsDefault: true}); err != nil {
			return err
		}
		return nil
	}
}

// WithCacheDuration sets the TTL for cached public keys of remote peers.
func WithCacheDuration(ttl time.Duration) CacheOption {
	return func(cache *Cache) error {
		cache.cacheDuration = ttl
		return nil
	}
}
