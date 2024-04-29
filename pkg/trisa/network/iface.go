package network

import (
	"context"
	"fmt"
	"io"

	directory "github.com/trisacrypto/envoy/pkg/trisa/gds"
	"github.com/trisacrypto/envoy/pkg/trisa/keychain"
	"github.com/trisacrypto/envoy/pkg/trisa/peers"

	"github.com/trisacrypto/trisa/pkg/trisa/keys"
)

// Network is a large scale interface that represents the TRISA network by embedding
// interactions with the TRISA Directory Service, a Peer Manager, and a Key Chain. Both
// incoming and outgoing TRISA interactions go through the TRISA Network interface.
type Network interface {
	DirectoryManager
	PeerManager
	KeyManager
	io.Closer
	fmt.Stringer
}

// PeerManager is an object that can create connections to remote TRISA peers either
// from an incoming request context or via unique lookup parameters. All PeerManager
// methods should return fully resolved (contains valid counterparty info) and
// connected (using mTLS) peers ready for TRISA network interactions.
type PeerManager interface {
	FromContext(context.Context) (peers.Peer, error)
	LookupPeer(ctx context.Context, commonNameOrID, registeredDirectory string) (peers.Peer, error)
	KeyExchange(context.Context, peers.Peer) (keys.Key, error)
	PeerDialer() PeerDialer
}

// KeyManager provides a high-level interface to key interactions based on polices and
// serves as a wrapper for a KeyChain object.
type KeyManager interface {
	SealingKey(commonName string) (pubkey keys.PublicKey, err error)
	UnsealingKey(signature, commonName string) (privkey keys.PrivateKey, err error)
	StorageKey(signature, commonName string) (pubkey keys.PublicKey, err error)
	ExchangeKey(commonName string) (pubkey keys.PublicKey, err error)
	Cache(commonName string, pubkey keys.Key) error
	KeyChain() (keychain.KeyChain, error)
}

// DirectoryManager provides a high-level interface to a specific directory service.
type DirectoryManager interface {
	Refresh() error
	Directory() (directory.Directory, error)
}
