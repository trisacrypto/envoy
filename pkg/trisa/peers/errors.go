package peers

import "errors"

var (
	ErrNoCommonName     = errors.New("a peer must have a unique common name")
	ErrNoEndpoint       = errors.New("peer does not have an endpoint to connect on")
	ErrAlreadyConnected = errors.New("already connected to remote peer, cannot overide dialer")
	ErrNotConnected     = errors.New("not connected to remote peer")
)
