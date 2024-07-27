package postman

import (
	"errors"
)

var (
	ErrNoCounterpartyInfo = errors.New("no counterparty info available on packet")
	ErrNoUnsealingKey     = errors.New("cannot open incoming envelope without unsealing key")
	ErrNoSealingKey       = errors.New("cannot seal outgoing envelope without sealing key")
)
