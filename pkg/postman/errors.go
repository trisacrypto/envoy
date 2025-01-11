package postman

import (
	"errors"
)

var (
	ErrNoCounterpartyInfo   = errors.New("no counterparty info available on packet")
	ErrNoUnsealingKey       = errors.New("cannot open incoming envelope without unsealing key")
	ErrNoSealingKey         = errors.New("cannot seal outgoing envelope without sealing key")
	ErrDatabaseNotReady     = errors.New("no database has been set on the packet")
	ErrCounterpartyNotReady = errors.New("no counterparty has been set on the packet")
	ErrTransactionNotReady  = errors.New("cannot resolve transaction from packet")
	ErrPacketNotReady       = errors.New("packet was not instantiated correctly")
)
