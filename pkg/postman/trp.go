package postman

import "github.com/trisacrypto/trisa/pkg/trisa/keys"

type TRP struct {
	Packet
	sealingKey   keys.PublicKey  // Optional: the public key used to seal the envelope sent to the remote peer
	unsealingKey keys.PrivateKey // Optional: the private key used to unseal the envelope received from the remote peer
}

func (p *TRP) SealingKey() keys.PublicKey {
	return p.sealingKey
}

func (p *TRP) UnsealingKey() keys.PrivateKey {
	return p.unsealingKey
}
