package keychain

import (
	"time"

	"github.com/trisacrypto/trisa/pkg/trisa/keys"
	"github.com/trisacrypto/trisa/pkg/trust"
)

// The KeyChain user interface is used to manage sealing and unsealing keys both for the
// local node and any public key material required for remote nodes. The KeyChain
// indexes keys based on their key signature and common name. For outgoing messages, the
// keychain can return the public key of the recipient by common name to seal outgoing
// envelopes. For incoming messages, the signature of the key and optionally the common
// name can be used to retrieve the private key to open the envelope.
type KeyChain interface {
	// Get the cached *rsa.PublicKey associated with the remote Peer received during a
	// KeyExchange RPC or GDS lookup. An error is returned if no sealing key is
	// available (or the cache has expired), requiring a new KeyExchange or a key lookup
	// from the GDS.
	SealingKey(commonName string) (pubkey keys.PublicKey, err error)

	// Get the private unsealing key either by public key signature on the envelope or
	// by common name from the mTLS certificates in the RPC to unseal an incoming secure
	// envelope sealed by the remote.
	UnsealingKey(signature, commonName string) (privkey keys.PrivateKey, err error)

	// Get the storage key associated with the UnsealingKey (e.g. the public key
	// component of the private key). This key is typically the same key as the exchange
	// key but earlier versions can be retrieved via the signature.
	StorageKey(signature, commonName string) (pubkey keys.PublicKey, err error)

	// Get the local public seal key to send to the remote in a key exchange so that
	// the remote Peer can seal envelopes being sent to this node.
	ExchangeKey(commonName string) (pubkey keys.PublicKey, err error)

	// Cache a public key received from the remote Peer during a key exchange.
	Cache(commonName string, pubkey keys.Key, ttl time.Duration) error

	// Store a private key pair for use in unsealing incoming envelopes and to send
	// public keys in key exchange request with remote peers. Options deals with how
	// the sealing key is used during key exchanges.
	Store(keypair keys.Key, opts *KeyOptions) error
}

// KeyStore maps key signatures to serialized public sealing keys that can be stored in
// memory or on disk. The KeyStore must also manage the time of storage for cache busting.
type KeyStore interface {
	Get(signature string) (keys.Key, time.Time, error)
	Put(key keys.Key) error
	Delete(signature string) error
}

// KeyOptions defines how multiple private key pairs are used during key exchanges.
type KeyOptions struct {
	// Specify the key is the default key to use in key exchanges. If a default key
	// already exists on the store this key will replace that key.
	IsDefault bool `json:"is_default"`

	// Associate this key with the common names of specific counterparties. Used to
	// identify specific remote peers to use this key with. If counterparties are
	// specified, this key only be used in key exchanges with incoming mTLS connections
	// that have the specified common names, unless it is already marked as the default.
	Counterparties []string `json:"counterparties"`

	// Do not use the key after the specified expiration date. This is used for time
	// based keys. Note that this field ignores expiration on the certificates and if
	// no keys are available that aren't expired, errors will be returned during key
	// exchanges. If this field is empty then the key will never expire.
	ExpiresOn time.Time `json:"expires_on"`
}

// Defines the interface for the mTLS config and key loading.
type CertConfig interface {
	LoadCerts() (_ *trust.Provider, err error)
	LoadPool() (_ trust.ProviderPool, err error)
}

func Load(conf CertConfig, opts ...CacheOption) (_ KeyChain, err error) {
	// TODO: use policies to create different kinds of keychains.
	// TODO: allow configuration of different underlying key stores.
	// For now, the network creates a default key chain with in memory key stores and
	// uses the identity certificate as the default sealing key until multi-key
	// management is enabled both by configuration and the TRISA working group.
	var provider *trust.Provider
	if provider, err = conf.LoadCerts(); err != nil {
		return nil, err
	}

	var localKey keys.Key
	if localKey, err = keys.FromProvider(provider); err != nil {
		return nil, err
	}

	options := []CacheOption{WithDefaultKey(localKey)}
	options = append(options, opts...)
	return New(options...)
}
