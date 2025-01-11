package postman

import "github.com/trisacrypto/envoy/pkg/trisa/peers"

type TRISA struct {
	Packet
	peer peers.Peer  // The remote peer the transfer is being conducted with
	info *peers.Info // The peer info for finding the counterparty
}

func (p *TRISA) ResolveCounterparty() (err error) {
	if p.counterparty == nil {
		if p.info == nil {
			if p.peer == nil {
				return ErrNoCounterpartyInfo
			}

			if p.info, err = p.peer.Info(); err != nil {
				return err
			}
		}

		p.counterparty = p.info.Model()
	}
	return nil
}
