package postman

import (
	"fmt"

	"github.com/rs/zerolog"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/envoy/pkg/trisa/peers"
)

// A functional option for setting packet fields.
type Option func(packet any)

func WithLogger(log zerolog.Logger) Option {
	return func(packet any) {
		switch packet := packet.(type) {
		case *Packet:
			packet.log = log
		case *TRISA:
			packet.Packet.log = log
		case TRP:
			packet.Packet.log = log
		case Sunrise:
			packet.Packet.log = log
		default:
			panic(fmt.Errorf("cannot set logger on type %T", packet))
		}
	}
}

func WithDatabase(db models.PreparedTransaction) Option {
	return func(packet any) {
		switch packet := packet.(type) {
		case *Packet:
			packet.db = db
		case *TRISA:
			packet.Packet.db = db
		case TRP:
			packet.Packet.db = db
		case Sunrise:
			packet.Packet.db = db
		default:
			panic(fmt.Errorf("cannot set database on type %T", packet))
		}
	}
}

func WithTransaction(txn *models.Transaction) Option {
	return func(packet any) {
		switch packet := packet.(type) {
		case *Packet:
			packet.txn = txn
		case *TRISA:
			packet.Packet.txn = txn
		case TRP:
			packet.Packet.txn = txn
		case Sunrise:
			packet.Packet.txn = txn
		default:
			panic(fmt.Errorf("cannot set transaction on type %T", packet))
		}
	}
}

func WithCounterparty(counterparty *models.Counterparty) Option {
	return func(packet any) {
		switch packet := packet.(type) {
		case *Packet:
			packet.counterparty = counterparty
		case *TRISA:
			packet.Packet.counterparty = counterparty
		case TRP:
			packet.Packet.counterparty = counterparty
		case Sunrise:
			packet.Packet.counterparty = counterparty
		default:
			panic(fmt.Errorf("cannot set counterparty on type %T", packet))
		}
	}
}

func WithPeer(peer peers.Peer) Option {
	return func(packet any) {
		switch packet := packet.(type) {
		case *TRISA:
			packet.peer = peer
			packet.info, _ = peer.Info()
		default:
			panic(fmt.Errorf("cannot set peer on type %T", packet))
		}
	}
}
