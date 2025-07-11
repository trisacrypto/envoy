package keychain

import (
	"fmt"
	"sync"
	"time"

	"github.com/trisacrypto/envoy/pkg/trisa/keychain/keyerr"
	"github.com/trisacrypto/envoy/pkg/trisa/keychain/memks"

	"github.com/trisacrypto/trisa/pkg/trisa/keys"
)

const (
	DefaultCacheDuration = 24 * time.Hour
)

// Cache implements the KeyChain interface and manages both the storage of private keys
// for unsealing envelopes and responding to key exchange requests as well as cacheing
// public keys from remote counterparties for sealing secure envelopes. By default the
// cache keeps keys from remote counter parties for 10 minutes before requiring another
// key exchange from the peer.
//
// The Cache stores public and private keys in separate key stores, by default using
// memks.MemStore as the default storage backend, but can be configured to use other
// key stores where necessary.
type Cache struct {
	sync.RWMutex
	internal      KeyStore             // stores private key pairs associated with the current node
	external      KeyStore             // stores public keys from key exchanges with counterparties
	names         SourceMap            // maps common names to key signatures
	defaultKey    string               // the default key to use if specified
	cacheDuration time.Duration        // amount of time external keys are cached for
	ttl           map[string]time.Time // the TTL of the cached keys for expiration purposes
}

// New returns a Cache KeyChain object configured and ready for use by the options.
func New(opts ...CacheOption) (_ KeyChain, err error) {
	// Create a chache with default in-memory stores (may be replaced by options).
	cache := &Cache{
		cacheDuration: DefaultCacheDuration,
		ttl:           make(map[string]time.Time),
		names:         NewSourceMap(),
	}

	if cache.internal, err = memks.New(); err != nil {
		return nil, err
	}

	if cache.external, err = memks.New(); err != nil {
		return nil, err
	}

	for _, opt := range opts {
		if err = opt(cache); err != nil {
			return nil, err
		}
	}

	return cache, nil
}

// Ensure Cache implements the KeyChain interface
var _ KeyChain = &Cache{}

// Get the public key (or other asymmetric public key) associated with the remote Peer
// via a key exchange using mTLS certificates with the given common name.
// An error is returned if no sealing key is available (or the cache has expired),
// requiring a KeyExchange or a key lookup from the GDS.
// This method operates on the external source (e.g. keys from external counterparties).
func (c *Cache) SealingKey(commonName, signature string) (pubkey keys.PublicKey, err error) {
	c.RLock()
	defer c.RUnlock()

	// Identify the signature from the common name, use default signature if not mapped.
	if signature == "" {
		if signature, err = c.lookup(commonName, ExternalSource); err != nil {
			return nil, err
		}
	}

	// Check if the key has expired (if there is no TTL, it is expired)
	if ttl, ok := c.ttl[signature]; !ok || time.Now().After(ttl) {
		// Return KeyNotFound instead of KeyExpired to represent cache misses
		return nil, keyerr.KeyNotFound
	}

	// Fetch the key from the key store if it's available
	if pubkey, _, err = c.external.Get(signature); err != nil {
		return nil, err
	}

	return pubkey, nil
}

// Get the private unsealing key either by public key signature on the envelope or
// by common name from the mTLS certificates in the RPC to unseal an incoming secure
// envelope sealed by the remote. If both a signature and a commonName are supplied, the
// commonName is ignored. If neither the signature nor the commonName are supplied then
// the default keys are returned if they are set on the cache.
// This method operates on the internal source (e.g. keys loaded for the local node).
func (c *Cache) UnsealingKey(signature, commonName string) (privkey keys.PrivateKey, err error) {
	c.RLock()
	defer c.RUnlock()

	// If a common name is supplied but not a signature look it up.
	if signature == "" {
		if signature, err = c.lookup(commonName, InternalSource); err != nil {
			return nil, err
		}
	}

	// If we don't have a signature at this point then there is no default key or it can't be looked up.
	if signature == "" {
		return nil, keyerr.KeyNotFound
	}

	// If the key has a TTL and it is after now then return expired (if no TTL, then key should not expire)
	// NOTE: if the key is the default key it should not expire and be returned.
	if signature != c.defaultKey {
		if ttl, ok := c.ttl[signature]; ok && time.Now().After(ttl) {
			return nil, keyerr.KeyExpired
		}
	}

	if privkey, _, err = c.internal.Get(signature); err != nil {
		return nil, err
	}

	return privkey, nil
}

func (c *Cache) StorageKey(signature, commonName string) (pubkey keys.PublicKey, err error) {
	c.RLock()
	defer c.RUnlock()

	// If a common name is supplied but not a signature look it up.
	if signature == "" {
		if signature, err = c.lookup(commonName, InternalSource); err != nil {
			return nil, err
		}
	}

	// If we don't have a signature at this point then there is no default key or it can't be looked up.
	if signature == "" {
		return nil, keyerr.KeyNotFound
	}

	// If the key has a TTL and it is after now then return expired (if no TTL, then key should not expire)
	// NOTE: if the key is the default key it should not expire and be returned.
	if signature != c.defaultKey {
		if ttl, ok := c.ttl[signature]; ok && time.Now().After(ttl) {
			return nil, keyerr.KeyExpired
		}
	}

	var key keys.Key
	if key, _, err = c.internal.Get(signature); err != nil {
		return nil, err
	}

	return key, nil
}

// Get the local public seal key to send to the remote in a key exchange so that
// the remote Peer can seal envelopes being sent to this node. If there is no keys
// specified for the common name, the default keys are returned.
// This method operates on the internal source (e.g. keys loaded for the local node).
func (c *Cache) ExchangeKey(commonName string) (pubkey keys.PublicKey, err error) {
	c.RLock()
	defer c.RUnlock()

	var signature string
	if signature, err = c.lookup(commonName, InternalSource); err != nil {
		return nil, err
	}

	// This should already be returned from c.lookup; this is just an extra guard.
	if signature == "" {
		return nil, keyerr.NoDefaultKeys
	}

	// If the key has a TTL and it is after now then return expired (if no TTL, then key should not expire)
	// NOTE: if the key is the default key it should not expire and be returned.
	if signature != c.defaultKey {
		if ttl, ok := c.ttl[signature]; ok && time.Now().After(ttl) {
			return nil, keyerr.KeyExpired
		}
	}

	if pubkey, _, err = c.internal.Get(signature); err != nil {
		return nil, err
	}

	return pubkey, nil
}

// Returns the default node keys.PrivateKey to be used for signing.
func (c *Cache) SigningKey() (privkey keys.PrivateKey, err error) {
	return c.UnsealingKey("", "")
}

// Returns the keys.PublicKey with the given signature for signature
// verification. If signature is the empty string, then the default local node
// signature verification key will be returned.
func (c *Cache) VerificationKey(signature string) (privkey keys.PublicKey, err error) {
	return c.SealingKey("", signature)
}

// Cache a public key received from the remote Peer during a key exchange.
// If ttl is less than or equal to 0 the default cache time is used.
// This method operates on the external source (e.g. keys from external counterparties).
func (c *Cache) Cache(commonName string, pubkey keys.Key, ttl time.Duration) (err error) {
	if pubkey.IsPrivate() {
		return keyerr.NoCachePrivateKeys
	}

	var signature string
	if signature, err = pubkey.PublicKeySignature(); err != nil {
		return err
	}

	var expires time.Time
	if ttl > 0 {
		expires = time.Now().Add(ttl)
	} else {
		expires = time.Now().Add(c.cacheDuration)
	}

	// Critical section
	c.Lock()
	defer c.Unlock()
	if err = c.external.Put(pubkey); err != nil {
		return err
	}

	c.ttl[signature] = expires
	c.names[ExternalSource][commonName] = signature
	return nil
}

// Store a private key pair for use in unsealing incoming envelopes and to send
// public keys in key exchange request with remote peers. Options deals with how
// the sealing key is used during key exchanges.
// This method operates on the internal source (e.g. keys loaded for the local node).
func (c *Cache) Store(keypair keys.Key, opts *KeyOptions) (err error) {
	if !keypair.IsPrivate() {
		return keyerr.NoStorePublicKeys
	}

	var signature string
	if signature, err = keypair.PublicKeySignature(); err != nil {
		return err
	}

	// Critical section
	c.Lock()
	defer c.Unlock()
	if err = c.internal.Put(keypair); err != nil {
		return err
	}

	// Manage key chain options
	if opts != nil {
		if opts.IsDefault {
			c.defaultKey = signature
		}

		for _, commonName := range opts.Counterparties {
			c.names[InternalSource][commonName] = signature
		}

		if !opts.ExpiresOn.IsZero() {
			c.ttl[signature] = opts.ExpiresOn
		}
	}
	return nil
}

// Lookup the signature for the specified common name. If the source is internal and
// there is a default key, then the default key signature is returned. Returns an error
// if no signature and no default key is available.
func (c *Cache) Lookup(commonName string, source Source) (signature string, err error) {
	c.RLock()
	defer c.RUnlock()
	return c.lookup(commonName, source)
}

// non-threadsafe lookup of signature for common name.
func (c *Cache) lookup(commonName string, source Source) (signature string, err error) {
	var ok bool
	if commonName != "" {
		if signature, ok = c.names[source][commonName]; ok {
			// TODO: if the signature is empty should we return an error instead of the default key?
			if signature != "" {
				return signature, nil
			}
		}
	}

	// Return default key for internal source
	if source == InternalSource && c.defaultKey != "" {
		return c.defaultKey, nil
	}

	// Return error based on source
	switch source {
	case InternalSource:
		return "", keyerr.NoDefaultKeys
	case ExternalSource:
		return "", keyerr.KeyNotMatched
	default:
		return "", keyerr.InvalidSource
	}
}

func (c *Cache) InternalStore() KeyStore {
	return c.internal
}

func (c *Cache) ExternalStore() KeyStore {
	return c.external
}

func (c *Cache) ClearCache() error {
	c.Lock()
	defer c.Unlock()
	for name, signature := range c.names[ExternalSource] {
		if err := c.external.Delete(signature); err != nil {
			return fmt.Errorf("could not delete key for %s with signature %s: %s", name, signature, err)
		}
	}
	return nil
}

// Source type provides cache directionality information. Internal source refers to keys
// stored for the local (e.g. internal node) usually containing private key pairs.
// External source refers to keys stored from incoming key exchanges, usually only
// public keys that are used to seal envelopes.
type Source uint8

const (
	UnknownSource Source = iota
	InternalSource
	ExternalSource
)

func (s Source) String() string {
	switch s {
	case UnknownSource:
		return "unknown"
	case InternalSource:
		return "internal"
	case ExternalSource:
		return "external"
	default:
		panic("invalid source type")
	}
}

// SourceMap is a dictionary that maps names (e.g. common names or signatures) to a
// map of source to interface, allowing the tree storage of data from different sources.
type SourceMap map[Source]map[string]string

func NewSourceMap() SourceMap {
	return SourceMap{
		ExternalSource: make(map[string]string),
		InternalSource: make(map[string]string),
	}
}
