package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	dberr "self-hosted-node/pkg/store/errors"
	"self-hosted-node/pkg/store/models"
	"self-hosted-node/pkg/ulids"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
)

const listTransactionsSQL = "SELECT id, source, status, counterparty, counterparty_id, originator, originator_address, beneficiary, beneficiary_address, virtual_asset, amount, last_update, created, modified FROM transactions"

func (s *Store) ListTransactions(ctx context.Context, page *models.PageInfo) (out *models.TransactionPage, err error) {
	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// TODO: handle pagination
	out = &models.TransactionPage{
		Transactions: make([]*models.Transaction, 0),
	}

	var rows *sql.Rows
	if rows, err = tx.Query(listTransactionsSQL); err != nil {
		// TODO: handle database specific errors
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		transaction := &models.Transaction{}
		if err = transaction.Scan(rows); err != nil {
			return nil, err
		}
		out.Transactions = append(out.Transactions, transaction)
	}

	tx.Commit()
	return out, nil
}

const createTransactionSQL = "INSERT INTO transactions (id, source, status, counterparty, counterparty_id, originator, originator_address, beneficiary, beneficiary_address, virtual_asset, amount, last_update, created, modified) VALUES (:id, :source, :status, :counterparty, :counterpartyID, :originator, :originatorAddress, :beneficiary, :beneficiaryAddress, :virtualAsset, :amount, :lastUpdate, :created, :modified)"

func (s *Store) CreateTransaction(ctx context.Context, transaction *models.Transaction) (err error) {
	// Basic validation
	if transaction.ID == uuid.Nil {
		return dberr.ErrNoIDOnCreate
	}

	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	// Create IDs and model metadata, updating the transaction in place
	transaction.ID = uuid.New()
	transaction.Created = time.Now()
	transaction.Modified = transaction.Created

	// Insert the transaction into the database
	if _, err = tx.Exec(createTransactionSQL, transaction.Params()...); err != nil {
		// TODO: handle constraint violations
		return err
	}

	return tx.Commit()
}

const retrieveTransactionSQL = "SELECT * FROM transactions WHERE id=:id"

func (s *Store) RetrieveTransaction(ctx context.Context, id uuid.UUID) (transaction *models.Transaction, err error) {
	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if transaction, err = s.retrieveTransaction(tx, id); err != nil {
		return nil, err
	}

	// Retrieve associated secure envelopes with the transaction
	if err = s.listSecureEnvelopes(tx, transaction); err != nil {
		return nil, err
	}

	tx.Commit()
	return transaction, nil
}

func (s *Store) retrieveTransaction(tx *sql.Tx, transactionID uuid.UUID) (transaction *models.Transaction, err error) {
	transaction = &models.Transaction{}
	if err = transaction.Scan(tx.QueryRow(retrieveTransactionSQL, sql.Named("id", transactionID))); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, dberr.ErrNotFound
		}
		return nil, err
	}
	return transaction, nil
}

const updateTransactionSQL = "UPDATE transactions SET source=:source, status=:status, counterparty=:counterparty, counterparty_id=:counterpatyID, originator=:originator, originator_address=:originatorAddress, beneficiary=:beneficiary, beneficiary_address=:beneficiaryAddress, virtual_asset=:virtualAsset, amount=:amount, lastUpdate=:lastUpdate, modified=:modified WHERE id=:id"

func (s *Store) UpdateTransaction(ctx context.Context, t *models.Transaction) (err error) {
	// Basic validation
	if t.ID == uuid.Nil {
		return dberr.ErrMissingID
	}

	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	// Update modified timestamp (in place).
	t.Modified = time.Now()

	// Execute the update into the database
	var result sql.Result
	if result, err = tx.Exec(updateTransactionSQL, t.Params()...); err != nil {
		// TODO: handle constraint violations
		return err
	}

	if nRows, _ := result.RowsAffected(); nRows == 0 {
		return dberr.ErrNotFound
	}

	return tx.Commit()
}

const deleteTransactionSQL = "DELETE FROM transactions WHERE id=:id"

func (s *Store) DeleteTransaction(ctx context.Context, id uuid.UUID) (err error) {
	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	var result sql.Result
	if result, err = tx.Exec(deleteTransactionSQL, sql.Named("id", id)); err != nil {
		return err
	}

	if nRows, _ := result.RowsAffected(); nRows == 0 {
		return dberr.ErrNotFound
	}

	return tx.Commit()
}

const listSecureEnvelopesSQL = "SELECT * FROM secure_envelopes WHERE envelope_id=:envelopeID"

func (s *Store) ListSecureEnvelopes(ctx context.Context, txID uuid.UUID, page *models.PageInfo) (out *models.SecureEnvelopePage, err error) {
	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var transaction *models.Transaction
	if transaction, err = s.retrieveTransaction(tx, txID); err != nil {
		return nil, err
	}

	// TODO: handle pagination
	out = &models.SecureEnvelopePage{
		Envelopes: make([]*models.SecureEnvelope, 0),
	}

	var rows *sql.Rows
	if rows, err = tx.Query(listSecureEnvelopesSQL, sql.Named("envelopeID", txID)); err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		env := &models.SecureEnvelope{}
		if err = env.Scan(rows); err != nil {
			return nil, err
		}

		env.SetTransaction(transaction)
		out.Envelopes = append(out.Envelopes, env)
	}

	if errors.Is(rows.Err(), sql.ErrNoRows) {
		return nil, dberr.ErrNotFound
	}

	tx.Commit()
	return nil, nil
}

func (s *Store) listSecureEnvelopes(tx *sql.Tx, transaction *models.Transaction) (err error) {
	var rows *sql.Rows
	if rows, err = tx.Query(listSecureEnvelopesSQL, sql.Named("envelopeID", transaction.ID)); err != nil {
		return err
	}
	defer rows.Close()

	envelopes := make([]*models.SecureEnvelope, 0)
	for rows.Next() {
		env := &models.SecureEnvelope{}
		if err = env.Scan(rows); err != nil {
			return err
		}

		env.SetTransaction(transaction)
		envelopes = append(envelopes, env)
	}

	transaction.SetSecureEnvelopes(envelopes)
	return nil
}

const createSecureEnvelopeSQL = "INSERT INTO secure_envelopes (id, envelope_id, direction, is_error, encryption_key, hmac_secret, valid_hmac, timestamp, public_key, envelope, created, modified) VALUES (:id, :envelopeID, :direction, :isError, :encryptionKey, :hmacSecret, :validHMAC, :timestamp, :publicKey, :envelope, :created, :modified)"

func (s *Store) CreateSecureEnvelope(ctx context.Context, env *models.SecureEnvelope) (err error) {
	if !ulids.IsZero(env.ID) {
		return dberr.ErrNoIDOnCreate
	}

	if env.EnvelopeID == uuid.Nil {
		return dberr.ErrMissingReference
	}

	// Create IDs and model metadata updating the secure envelope in place.
	env.ID = ulids.New()
	env.Created = time.Now()
	env.Modified = env.Created

	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err = tx.Exec(createSecureEnvelopeSQL, env.Params()...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dberr.ErrNotFound
		}

		// TODO: handle constraint violations
		return err
	}

	return tx.Commit()
}

const retrieveSecureEnvelopeSQL = "SELECT * FROM secure_envelopes WHERE id=:envID and envelope_id=:txID"

func (s *Store) RetrieveSecureEnvelope(ctx context.Context, txID uuid.UUID, envID ulid.ULID) (env *models.SecureEnvelope, err error) {
	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	env = &models.SecureEnvelope{}
	if err = env.Scan(tx.QueryRow(retrieveSecureEnvelopeSQL, sql.Named("envID", envID), sql.Named("txID", txID))); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, dberr.ErrNotFound
		}
		return nil, err
	}

	tx.Commit()
	return env, nil
}

const updateSecureEnvelopeSQL = "UPDATE secure_envelopes SET direction=:direction, is_error=:is_error, encryption_key=:encryptionKey, hmac_secret=:hmacSecret, valid_hmac=:validHMAC, timestamp=:timestamp, public_key=:publicKey, envelope=:envelope, modified=:modified WHERE id=:id and envelope_id=:envelopeID"

func (s *Store) UpdateSecureEnvelope(ctx context.Context, env *models.SecureEnvelope) (err error) {
	// Basic validation
	if ulids.IsZero(env.ID) {
		return dberr.ErrMissingID
	}

	if env.EnvelopeID == uuid.Nil {
		return dberr.ErrMissingReference
	}

	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	env.Modified = time.Now()

	var result sql.Result
	if result, err = tx.Exec(updateSecureEnvelopeSQL, env.Params()...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dberr.ErrNotFound
		}

		// TODO: handle constraint violations
		return err
	}

	if nRows, _ := result.RowsAffected(); nRows == 0 {
		return dberr.ErrNotFound
	}

	return tx.Commit()
}

const deleteSecureEnvelopeSQL = "DELETE FROM secure_envelopes WHERE id=:envID AND envelope_id=:txID"

func (s *Store) DeleteSecureEnvelope(ctx context.Context, txID uuid.UUID, envID ulid.ULID) (err error) {
	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	var result sql.Result
	if result, err = tx.Exec(deleteSecureEnvelopeSQL, sql.Named("txID", txID), sql.Named("envID", envID)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dberr.ErrNotFound
		}
		return err
	}

	if nRows, _ := result.RowsAffected(); nRows == 0 {
		return dberr.ErrNotFound
	}

	return tx.Commit()
}
