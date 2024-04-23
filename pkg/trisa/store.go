package trisa

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/trisacrypto/envoy/pkg/store/models"
)

func (s *Server) StoreIncoming(in *Incoming) (err error) {
	// STEP 1: get or create transaction
	tx := &models.Transaction{
		Counterparty: in.peer.Name(),
	}

	in.peer.Info()

	// TODO: return rejection instead of plain error
	if tx.ID, err = uuid.Parse(in.ID()); err != nil {
		return fmt.Errorf("envelope id must be a uuid: %w", err)
	}

	// STEP 2: save envelope

	// Source depends on if this is a new transfer or relates to an old transfer
	// Status depends on the status determined from the incoming envelope
	// Originator through amount requires decrypted information

	return nil
}

func (s *Server) StoreOutgoing() error {
	return nil
}
