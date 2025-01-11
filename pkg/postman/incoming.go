package postman

import "github.com/trisacrypto/trisa/pkg/trisa/envelope"

type Incoming struct {
	Envelope *envelope.Envelope
	packet   *Packet
}
