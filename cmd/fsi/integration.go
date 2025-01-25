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
	prepare := makePrepare("ta2fFeKgcLirnGbYFL9YnkqWr8kQu1gW7PWhxHqqcDErjSZLTeeqYWGKwbNT")

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

	// Complete the transaction with a txid
	if preview, err = envoyClient.AcceptPreview(ctx, transaction.ID); err != nil {
		return fmt.Errorf("could not retrieve accept preview from envoy: %w", err)
	}

	// TODO: the accept preview should probably do this
	preview.Transaction = preview.Pending.Transaction
	preview.Transaction.Txid = "b657e22827039461a9493ede7bdf55b01579254c1630b0bfc9185ec564fc05ab"
	preview.TransferState = "completed"
	preview.Pending = nil

	if _, err = envoyClient.SendEnvelope(ctx, transaction.ID, preview); err != nil {
		return fmt.Errorf("could not complete transaction: %w", err)
	}

	return nil
}

func testTRISAWorkflow_Reject() (err error) {
	log.Debug().Msg("testing complete TRISA workflow with rejection")
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	// Prepare a transaction to send
	// TODO: get travel address from command line
	prepare := makePrepare("ta2fFeKgcLirnGbYFL9YnkqWr8kQu1gW7PWhxHqqcDErjSZLTeeqYWGKwbNT")

	var prepared *api.Prepared
	if prepared, err = envoyClient.Prepare(ctx, prepare); err != nil {
		return fmt.Errorf("could not prepare transaction: %w", err)
	}

	var transaction *api.Transaction
	if transaction, err = envoyClient.SendPrepared(ctx, prepared); err != nil {
		return fmt.Errorf("could not send prepared transaction: %w", err)
	}

	log.Debug().Str("envelope_id", transaction.ID.String()).Msg("trisa message sent to counterparty")

	// Reject the transaction without any changes to the payload.
	if _, err = counterpartyClient.AcceptPreview(ctx, transaction.ID); err != nil {
		return fmt.Errorf("could not retrieve accept preview from counterparty: %w", err)
	}

	rejection := &api.Rejection{
		Code:    "HIGH_RISK",
		Message: "specified beneficiary is not authorized to receive foreign transfers",
		Retry:   false,
	}

	if _, err = counterpartyClient.Reject(ctx, transaction.ID, rejection); err != nil {
		return fmt.Errorf("could not reject transaction: %w", err)
	}

	return nil
}

func testTRISAWorkflow_Repair() (err error) {
	log.Debug().Msg("testing complete TRISA workflow with approval")
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	// Prepare a transaction to send
	// TODO: get travel address from command line
	prepare := makePrepare("ta2fFeKgcLirnGbYFL9YnkqWr8kQu1gW7PWhxHqqcDErjSZLTeeqYWGKwbNT")

	var prepared *api.Prepared
	if prepared, err = envoyClient.Prepare(ctx, prepare); err != nil {
		return fmt.Errorf("could not prepare transaction: %w", err)
	}

	var transaction *api.Transaction
	if transaction, err = envoyClient.SendPrepared(ctx, prepared); err != nil {
		return fmt.Errorf("could not send prepared transaction: %w", err)
	}

	log.Debug().Str("envelope_id", transaction.ID.String()).Msg("trisa message sent to counterparty")

	// Send a repair request back to the originator
	if _, err = counterpartyClient.AcceptPreview(ctx, transaction.ID); err != nil {
		return fmt.Errorf("could not retrieve accept preview from counterparty: %w", err)
	}

	rejection := &api.Rejection{
		Code:    "INCOMPLETE_IDENTITY",
		Message: "specified beneficiary has an incorrect or missing date of birth, which is required",
		Retry:   true,
	}

	if _, err = counterpartyClient.Reject(ctx, transaction.ID, rejection); err != nil {
		return fmt.Errorf("could not reject transaction: %w", err)
	}

	// Perform a repair as requested
	var repair *api.Repair
	if repair, err = envoyClient.RepairPreview(ctx, transaction.ID); err != nil {
		return fmt.Errorf("could not retrieve repair preview from envoy: %w", err)
	}

	beneficiary := repair.Envelope.Identity.Beneficiary.BeneficiaryPersons[0]
	beneficiary.GetNaturalPerson().DateAndPlaceOfBirth.DateOfBirth = randDoB()

	repair.Envelope.TransferState = "repaired"
	if _, err = envoyClient.Repair(ctx, transaction.ID, repair.Envelope); err != nil {
		return fmt.Errorf("could not send repaired transaction: %w", err)
	}

	// Accept the transaction without any changes to the payload.
	var preview *api.Envelope
	if preview, err = counterpartyClient.AcceptPreview(ctx, transaction.ID); err != nil {
		return fmt.Errorf("could not retrieve accept preview from counterparty: %w", err)
	}

	preview.TransferState = "accepted"
	if _, err = counterpartyClient.Accept(ctx, transaction.ID, preview); err != nil {
		return fmt.Errorf("could not accept transaction: %w", err)
	}

	// Complete the transaction
	// TODO: the accept preview should probably do this
	preview.Transaction = preview.Pending.Transaction
	preview.Transaction.Txid = randTxID()
	preview.TransferState = "completed"
	preview.Pending = nil

	if _, err = envoyClient.SendEnvelope(ctx, transaction.ID, preview); err != nil {
		return fmt.Errorf("could not complete transaction: %w", err)
	}

	return nil
}
