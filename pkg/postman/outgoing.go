package postman

import "github.com/trisacrypto/trisa/pkg/trisa/envelope"

type Outgoing struct {
	Envelope *envelope.Envelope
	packet   *Packet
}
