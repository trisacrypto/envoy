package network

import "errors"

var (
	ErrNoGRPCPeer             = errors.New("no grpc remote peer info found in context")
	ErrNoKeyChain             = errors.New("no key chain available on network")
	ErrNoDirectory            = errors.New("no directory configured on the network")
	ErrUnknownPeerCertificate = errors.New("could not verify peer certificate subject info")
	ErrUnknownPeerSubject     = errors.New("could not identify common name on certificate subject")
)
