package network

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"self-hosted-node/pkg/config"
	directory "self-hosted-node/pkg/trisa/gds"
	"self-hosted-node/pkg/trisa/keychain"
	"self-hosted-node/pkg/trisa/peers"

	"github.com/google/uuid"

	"github.com/rs/zerolog/log"
	members "github.com/trisacrypto/directory/pkg/gds/members/v1alpha1"
	api "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
	gds "github.com/trisacrypto/trisa/pkg/trisa/gds/api/v1beta1"
	"github.com/trisacrypto/trisa/pkg/trisa/keys"
	"github.com/trisacrypto/trisa/pkg/trust"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	grpcpeer "google.golang.org/grpc/peer"
)

// New returns a Network object which manages the entire TRISA network including remote
// peers, public and private key management, and interactions with the Directory Service.
func New(conf config.TRISAConfig) (_ Network, err error) {
	network := &TRISANetwork{
		conf:        conf,
		peers:       make(map[string]peers.Peer),
		directory:   directory.New(conf),
		constructor: peers.New,
	}

	if err = network.directory.Connect(); err != nil {
		return nil, fmt.Errorf("could not connect to GDS: %s", err)
	}

	if network.dialer, err = TRISADialer(network.conf); err != nil {
		return nil, err
	}

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

	if network.keyChain, err = keychain.New(keychain.WithDefaultKey(localKey), keychain.WithCacheDuration(conf.KeyExchangeCacheTTL)); err != nil {
		return nil, err
	}
	return network, nil
}

// TRISANetwork implements the Network interface managing TRISA peers.
type TRISANetwork struct {
	sync.RWMutex
	conf        config.TRISAConfig
	keyChain    keychain.KeyChain
	directory   directory.Directory
	dialer      PeerDialer
	constructor PeerConstructor
	peers       map[string]peers.Peer
}

//====================================================================================
// PeerManager Methods
//====================================================================================

// FromContext is used to fetch a resolved and connected Peer object from an incoming
// mTLS request by parsing the TLSInfo in the gRPC connection to get the common name of
// the counterparty making the request.
func (n *TRISANetwork) FromContext(ctx context.Context) (peers.Peer, error) {
	var (
		ok      bool
		remote  *grpcpeer.Peer
		tlsInfo credentials.TLSInfo
	)

	if remote, ok = grpcpeer.FromContext(ctx); !ok {
		return nil, ErrNoGRPCPeer
	}

	if tlsInfo, ok = remote.AuthInfo.(credentials.TLSInfo); !ok {
		return nil, fmt.Errorf("unexpected peer transport credentials type: %T", remote.AuthInfo)
	}

	if len(tlsInfo.State.VerifiedChains) == 0 || len(tlsInfo.State.VerifiedChains[0]) == 0 {
		return nil, ErrUnknownPeerCertificate
	}

	commonName := tlsInfo.State.VerifiedChains[0][0].Subject.CommonName
	if commonName == "" {
		return nil, ErrUnknownPeerSubject
	}

	// Now that we've extracted the common name, perform a lookup for the peer.
	return n.LookupPeer(ctx, commonName, "")
}

// LookupPeer by common name or vasp ID, returning a cached peer if one has already been
// resolved, otherwise performing a directory service lookup and creating the new peer,
// connecting it so that it's ready for use any time the peer is fetched. This is the
// primary entry point for all Peer lookups - it can be used directly to get a remote
// connection for an outgoing peer and is called FromContext to lookup an incoming peer.
//
// NOTE: registeredDirectory is currently unused and can be safely ignored, but is added
// here for future proofing for the possibility of a distributed directory service.
func (n *TRISANetwork) LookupPeer(ctx context.Context, commonNameOrID, registeredDirectory string) (peer peers.Peer, err error) {
	// Check if the peer is in the cache (thread-safe)
	// TODO: cache by vaspID/registeredDirectory in addition to common name
	var ok bool
	if peer, ok = n.fetch(commonNameOrID); ok {
		return peer, nil
	}

	// If the peer is not in the cache, look it up via the directory service.
	// NOTE: because this section of code is not guarded by the mutex it introduces the
	// possibility that multiple concurrent lookups will issue multiple requests to the
	// directory service. The cache will remain consistent - on the first result,
	// lookups will return the cached value, the last lookup that completes will be the
	// final value stored in the cache.
	req := &gds.LookupRequest{
		RegisteredDirectory: registeredDirectory,
	}

	if _, err := uuid.Parse(commonNameOrID); err == nil {
		req.Id = commonNameOrID
	} else {
		req.CommonName = commonNameOrID
	}

	var rep *gds.LookupReply
	if rep, err = n.directory.Lookup(ctx, req); err != nil {
		return nil, err
	}

	if rep.Error != nil {
		return nil, fmt.Errorf("[%d] %s", rep.Error.Code, rep.Error.Message)
	}

	info := &peers.Info{
		ID:                  rep.Id,
		RegisteredDirectory: rep.RegisteredDirectory,
		CommonName:          rep.CommonName,
		Endpoint:            rep.Endpoint,
		Name:                rep.Name,
		Country:             rep.Country,
	}

	if rep.VerifiedOn != "" {
		if info.VerifiedOn, err = time.Parse(time.RFC3339, rep.VerifiedOn); err != nil {
			log.Warn().Err(err).Msg("unable to parse verified on timestamp")
		}
	}

	return n.create(info)
}

// KeyExchange conducts a KeyExchange request with the remote peer and then caches the
// response in the keychain for future use. The key is returned if available.
func (n *TRISANetwork) KeyExchange(ctx context.Context, peer peers.Peer) (seal keys.Key, err error) {
	var local keys.PublicKey
	if local, err = n.keyChain.ExchangeKey(peer.Name()); err != nil {
		return nil, err
	}

	var out, remote *api.SigningKey
	if out, err = local.Proto(); err != nil {
		return nil, err
	}

	if remote, err = peer.KeyExchange(ctx, out); err != nil {
		return nil, err
	}

	if seal, err = keys.FromSigningKey(remote); err != nil {
		return nil, err
	}

	if err = n.keyChain.Cache(peer.Name(), seal, n.conf.KeyExchangeCacheTTL); err != nil {
		// If we cannot cache the key but we've successfully parsed it, warn but do not error
		log.Warn().Err(err).Str("peer", peer.Name()).Msg("could not cache remote public sealing keys")
	}
	return seal, nil
}

func (n *TRISANetwork) PeerDialer() PeerDialer {
	return n.dialer
}

// thread-safe read from the internal cache.
func (n *TRISANetwork) fetch(commonName string) (peer peers.Peer, ok bool) {
	n.RLock()
	peer, ok = n.peers[commonName]
	n.RUnlock()
	return peer, ok
}

// create, connect, and cache a peer.
func (n *TRISANetwork) create(info *peers.Info) (peer peers.Peer, err error) {
	if peer, err = n.constructor(info); err != nil {
		return nil, err
	}

	var opts []grpc.DialOption
	if opts, err = n.dialer(info.Endpoint); err != nil {
		return nil, err
	}

	if err = peer.Connect(opts...); err != nil {
		return nil, err
	}

	if err = n.cache(peer); err != nil {
		return nil, err
	}
	return peer, nil
}

// thread-safe write to the internal cache.
func (n *TRISANetwork) cache(peer peers.Peer) (err error) {
	var info *peers.Info
	if info, err = peer.Info(); err != nil {
		return err
	}

	// Ensure there is a common name, a triple check of validity for safety
	if err = info.Validate(); err != nil {
		return err
	}

	n.Lock()
	n.peers[info.CommonName] = peer
	n.Unlock()
	return nil
}

//====================================================================================
// KeyManager Methods
//====================================================================================

func (n *TRISANetwork) SealingKey(commonName string) (pubkey keys.PublicKey, err error) {
	return n.keyChain.SealingKey(commonName)
}

// Get the private unsealing key either by public key signature on the envelope or
// by common name from the mTLS certificates in the RPC to unseal an incoming secure
// envelope sealed by the remote.
func (n *TRISANetwork) UnsealingKey(signature, commonName string) (privkey keys.PrivateKey, err error) {
	return n.keyChain.UnsealingKey(signature, commonName)
}

// Get the local public seal key to send to the remote in a key exchange so that
// the remote Peer can seal envelopes being sent to this node.
func (n *TRISANetwork) ExchangeKey(commonName string) (pubkey keys.PublicKey, err error) {
	return n.keyChain.ExchangeKey(commonName)
}

// Cache a public key received from the remote Peer during a key exchange.
func (n *TRISANetwork) Cache(commonName string, pubkey keys.Key) error {
	return n.keyChain.Cache(commonName, pubkey, n.conf.KeyExchangeCacheTTL)
}

func (n *TRISANetwork) KeyChain() (keychain.KeyChain, error) {
	if n.keyChain == nil {
		return nil, ErrNoKeyChain
	}
	return n.keyChain, nil
}

//====================================================================================
// DirectoryManager Methods
//====================================================================================

// Refresh the cached peers by performing a directory members listing and populating the
// internal cache with connected peers. Routinely refreshing the network listing
// improves the performance of lookups by preventing per-RPC GDS queries.
func (n *TRISANetwork) Refresh() (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// Initiate the list request without pagination
	req := &members.ListRequest{PageSize: 100}
	for {
		var rep *members.ListReply
		if rep, err = n.directory.List(ctx, req); err != nil {
			return err
		}

		// Handle VASP records from list reply
		for _, vasp := range rep.Vasps {
			info := &peers.Info{
				ID:                  vasp.Id,
				RegisteredDirectory: vasp.RegisteredDirectory,
				CommonName:          vasp.CommonName,
				Endpoint:            vasp.Endpoint,
				Name:                vasp.Name,
				Country:             vasp.Country,
			}

			if vasp.VerifiedOn != "" {
				if info.VerifiedOn, err = time.Parse(time.RFC3339, vasp.VerifiedOn); err != nil {
					log.Warn().Err(err).Msg("unable to parse verified on timestamp")
				}
			}

			if _, err = n.create(info); err != nil {
				log.Warn().Err(err).
					Str("common_name", vasp.CommonName).
					Str("vasp_id", vasp.Id).
					Msg("could not create peer")
			}
		}

		// Pagination complete when there is no next page token
		if rep.NextPageToken == "" {
			return nil
		}

		// Fetch the next page on the next loop
		req.PageToken = rep.NextPageToken
	}
}

func (n *TRISANetwork) Directory() (directory.Directory, error) {
	if n.directory == nil {
		return nil, ErrNoDirectory
	}
	return n.directory, nil
}

//====================================================================================
// Other Network Methods
//====================================================================================

// Close connections to directory service, all peer connections, and cleanup. The
// network is unusable after it is closed and could panic if calls are made to it.
func (n *TRISANetwork) Close() (err error) {
	if cerr := n.directory.Close(); cerr != nil {
		err = errors.Join(err, cerr)
	}

	for name, peer := range n.peers {
		if cerr := peer.Close(); cerr != nil {
			err = errors.Join(err, fmt.Errorf("could not close %s: %s", name, cerr))
		}
	}

	// Cleanup closures and cache
	n.peers = nil
	n.constructor = nil
	n.dialer = nil
	return err
}

// String returns the last part of the configured endpoint usually returning
// vaspdirectory.net or trisatest.net depending on the configuration.
func (n *TRISANetwork) String() string {
	return n.conf.Directory.Network()
}

// NPeers returns the number of peers in the cache, used primarily for testing.
func (n *TRISANetwork) NPeers() int {
	n.RLock()
	defer n.RUnlock()
	return len(n.peers)
}

// Contains returns true if the common name is in the cache, used primarily for testing.
func (n *TRISANetwork) Contains(commonName string) (ok bool) {
	n.RLock()
	defer n.RUnlock()
	_, ok = n.peers[commonName]
	return ok
}

// Reset and empty the cache of peers
func (n *TRISANetwork) Reset() {
	n.Lock()
	defer n.Unlock()
	for name := range n.peers {
		delete(n.peers, name)
	}
}
