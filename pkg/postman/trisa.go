package postman

import (
	"github.com/trisacrypto/envoy/pkg/trisa/peers"
	"github.com/trisacrypto/trisa/pkg/trisa/keys"
)

type TRISA struct {
	Packet
	Peer         peers.Peer      // The remote peer the transfer is being conducted with
	PeerInfo     *peers.Info     // The peer info for finding the counterparty
	SealingKey   keys.PublicKey  // The public key used to seal the envelope sent to the remote peer
	UnsealingKey keys.PrivateKey // The private key used to unseal the envelope received from the remote peer
}
