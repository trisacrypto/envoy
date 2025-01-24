package main

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/trisacrypto/envoy/pkg/web/api/v1"
)

func testTRISAWorkflow_Approve() (err error) {
	log.Debug().Msg("testing complete TRISA workflow with approval")
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	// Prepare a transaction to send
	// TODO: get travel address from command line
	prepare := &api.Prepare{}
	if err = unmarshalJSONFixture("integration/approve/prepare.json", prepare); err != nil {
		return fmt.Errorf("could not load prepare.json: %w", err)
	}

	var prepared *api.Prepared
	if prepared, err = envoyClient.Prepare(ctx, prepare); err != nil {
		return fmt.Errorf("could not prepare transaction: %w", err)
	}

	var transaction *api.Transaction
	if transaction, err = envoyClient.SendPrepared(ctx, prepared); err != nil {
		return fmt.Errorf("could not send prepared transaction: %w", err)
	}

	log.Debug().Str("envelope_id", transaction.ID.String()).Msg("trisa message sent to counterparty")

	// Accept the transaction without any changes to the payload.
	var preview *api.Envelope
	if preview, err = counterpartyClient.AcceptPreview(ctx, transaction.ID); err != nil {
		return fmt.Errorf("could not retrieve accept preview from counterparty: %w", err)
	}

	preview.TransferState = "accepted"
	if _, err = counterpartyClient.Accept(ctx, transaction.ID, preview); err != nil {
		return fmt.Errorf("could not accept transaction: %w", err)
	}

	return nil
}
