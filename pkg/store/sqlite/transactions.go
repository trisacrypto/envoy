package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/trisacrypto/envoy/pkg/enum"
	dberr "github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/models"

	"github.com/google/uuid"
	"go.rtnl.ai/ulid"
)

//===========================================================================
// Transaction CRUD interface
//==========================================================================

const listTransactionsSQL = "SELECT t.id, t.source, t.status, t.counterparty, t.counterparty_id, t.originator, t.originator_address, t.beneficiary, t.beneficiary_address, t.virtual_asset, t.amount, t.archived, t.archived_on, t.last_update, t.modified, t.created, count(e.id) AS numEnvelopes FROM transactions t LEFT JOIN secure_envelopes e ON t.id=e.envelope_id WHERE t.archived=:archives GROUP BY t.id ORDER BY t.created DESC"

func (s *Store) ListTransactions(ctx context.Context, page *models.TransactionPageInfo) (out *models.TransactionPage, err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if out, err = tx.ListTransactions(page); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return out, nil
}

func (t *Tx) ListTransactions(page *models.TransactionPageInfo) (out *models.TransactionPage, err error) {
	out = &models.TransactionPage{
		Transactions: make([]*models.Transaction, 0),
		Page: &models.TransactionPageInfo{
			PageInfo:     *models.PageInfoFrom(&page.PageInfo),
			Status:       page.Status,
			VirtualAsset: page.VirtualAsset,
			Archives:     page.Archives,
		},
	}

	// Create the base query and the query parameters list.
	query := listTransactionsSQL
	params := []interface{}{sql.Named("archives", page.Archives)}

	// If there are filters in the page query, then modify the SQL query with them.
	if len(page.Status) > 0 || len(page.VirtualAsset) > 0 {
		filters := make([]string, 0, 2)
		if len(page.Status) > 0 {
			inquery, inparams := listParametrize(page.Status, "s")
			filters = append(filters, "status IN "+inquery)
			params = append(params, inparams...)
		}

		if len(page.VirtualAsset) > 0 {
			inquery, inparams := listParametrize(page.VirtualAsset, "a")
			filters = append(filters, "virtual_asset IN "+inquery)
			params = append(params, inparams...)
		}

		query = "WITH txns AS (" + listTransactionsSQL + ") SELECT * FROM txns WHERE "
		query += strings.Join(filters, " AND ")
	}

	var rows *sql.Rows
	if rows, err = t.tx.Query(query, params...); err != nil {
		return nil, dbe(err)
	}
	defer rows.Close()

	for rows.Next() {
		transaction := &models.Transaction{}
		if err = transaction.ScanWithCount(rows); err != nil {
			return nil, err
		}
		out.Transactions = append(out.Transactions, transaction)
	}

	return out, nil
}

const createTransactionSQL = "INSERT INTO transactions (id, source, status, counterparty, counterparty_id, originator, originator_address, beneficiary, beneficiary_address, virtual_asset, amount, archived, archived_on, last_update, created, modified) VALUES (:id, :source, :status, :counterparty, :counterpartyID, :originator, :originatorAddress, :beneficiary, :beneficiaryAddress, :virtualAsset, :amount, :archived, :archivedOn, :lastUpdate, :created, :modified)"

func (s *Store) CreateTransaction(ctx context.Context, transaction *models.Transaction, auditLog *models.ComplianceAuditLog) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.CreateTransaction(transaction, auditLog); err != nil {
		return err
	}

	return tx.Commit()
}

func (t *Tx) CreateTransaction(transaction *models.Transaction, auditLog *models.ComplianceAuditLog) (err error) {
	// Basic validation
	if transaction.ID != uuid.Nil {
		return dberr.ErrNoIDOnCreate
	}

	// Create IDs and model metadata, updating the transaction in place
	transaction.ID = uuid.New()
	transaction.Created = time.Now()
	transaction.Modified = transaction.Created

	// Insert the transaction into the database
	if _, err = t.tx.Exec(createTransactionSQL, transaction.Params()...); err != nil {
		return dbe(err)
	}

	//FIXME: COMPLETE AUDIT LOG
	_ = auditLog

	return nil
}

const retrieveTransactionSQL = "SELECT id, source, status, counterparty, counterparty_id, originator, originator_address, beneficiary, beneficiary_address, virtual_asset, amount, archived, archived_on, last_update, created, modified FROM transactions WHERE id=:id"

// Retrieve a transaction record by its ID and any related secure envelopes.
func (s *Store) RetrieveTransaction(ctx context.Context, id uuid.UUID) (transaction *models.Transaction, err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if transaction, err = tx.RetrieveTransaction(id); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return transaction, nil
}

// Retrieve a transaction record by its ID and any related secure envelopes.
func (t *Tx) RetrieveTransaction(transactionID uuid.UUID) (transaction *models.Transaction, err error) {
	if transaction, err = t.retrieveTransaction(transactionID); err != nil {
		return nil, err
	}

	// Retrieve associated secure envelopes with the transaction
	if err = t.associateSecureEnvelopes(transaction); err != nil {
		return nil, err
	}

	return transaction, nil
}

// Retrieve only a transaction record by its ID without associated secure envelopes.
func (t *Tx) retrieveTransaction(transactionID uuid.UUID) (transaction *models.Transaction, err error) {
	transaction = &models.Transaction{}
	if err = transaction.Scan(t.tx.QueryRow(retrieveTransactionSQL, sql.Named("id", transactionID))); err != nil {
		return nil, dbe(err)
	}
	return transaction, nil
}

const updateTransactionSQL = "UPDATE transactions SET source=:source, status=:status, counterparty=:counterparty, counterparty_id=:counterpartyID, originator=:originator, originator_address=:originatorAddress, beneficiary=:beneficiary, beneficiary_address=:beneficiaryAddress, virtual_asset=:virtualAsset, amount=:amount, archived=:archived, archived_on=:archivedOn, last_update=:lastUpdate, modified=:modified WHERE id=:id"

func (s *Store) UpdateTransaction(ctx context.Context, t *models.Transaction, auditLog *models.ComplianceAuditLog) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.UpdateTransaction(t, auditLog); err != nil {
		return err
	}
	return tx.Commit()
}

func (t *Tx) UpdateTransaction(transaction *models.Transaction, auditLog *models.ComplianceAuditLog) (err error) {
	// Basic validation
	if transaction.ID == uuid.Nil {
		return dberr.ErrMissingID
	}

	// Update modified timestamp (in place).
	transaction.Modified = time.Now()

	// NOTE: do not update `LastUpdate` timestamp - this refers to when a secure envelope is sent/received.

	// Execute the update into the database
	var result sql.Result
	if result, err = t.tx.Exec(updateTransactionSQL, transaction.Params()...); err != nil {
		return dbe(err)
	}

	if nRows, _ := result.RowsAffected(); nRows == 0 {
		return dberr.ErrNotFound
	}

	//FIXME: COMPLETE AUDIT LOG
	_ = auditLog

	return nil
}

const deleteTransactionSQL = "DELETE FROM transactions WHERE id=:id"

func (s *Store) DeleteTransaction(ctx context.Context, id uuid.UUID, auditLog *models.ComplianceAuditLog) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.DeleteTransaction(id, auditLog); err != nil {
		return err
	}

	return tx.Commit()
}

func (t *Tx) DeleteTransaction(id uuid.UUID, auditLog *models.ComplianceAuditLog) (err error) {
	var result sql.Result
	if result, err = t.tx.Exec(deleteTransactionSQL, sql.Named("id", id)); err != nil {
		return dbe(err)
	}

	if nRows, _ := result.RowsAffected(); nRows == 0 {
		return dberr.ErrNotFound
	}

	//FIXME: COMPLETE AUDIT LOG
	_ = auditLog

	return nil
}

const archiveTransactionSQL = "UPDATE transactions SET archived=:archived, archived_on=:archivedOn, modified=:modified WHERE id=:id"

func (s *Store) ArchiveTransaction(ctx context.Context, transactionID uuid.UUID, auditLog *models.ComplianceAuditLog) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.ArchiveTransaction(transactionID, auditLog); err != nil {
		return err
	}

	return tx.Commit()
}

func (t *Tx) ArchiveTransaction(transactionID uuid.UUID, auditLog *models.ComplianceAuditLog) (err error) {
	timestamp := time.Now()
	params := []any{
		sql.Named("id", transactionID),
		sql.Named("archived", true),
		sql.Named("archivedOn", timestamp),
		sql.Named("modified", timestamp),
	}

	var result sql.Result
	if result, err = t.tx.Exec(archiveTransactionSQL, params...); err != nil {
		return dbe(err)
	}

	if nRows, _ := result.RowsAffected(); nRows == 0 {
		return dberr.ErrNotFound
	}

	//FIXME: COMPLETE AUDIT LOG
	_ = auditLog

	return nil
}

func (s *Store) UnarchiveTransaction(ctx context.Context, transactionID uuid.UUID, auditLog *models.ComplianceAuditLog) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.UnarchiveTransaction(transactionID, auditLog); err != nil {
		return err
	}

	return tx.Commit()
}

func (t *Tx) UnarchiveTransaction(transactionID uuid.UUID, auditLog *models.ComplianceAuditLog) (err error) {
	params := []any{
		sql.Named("id", transactionID),
		sql.Named("archived", false),
		sql.Named("archivedOn", sql.NullTime{Valid: false}),
		sql.Named("modified", time.Now()),
	}

	var result sql.Result
	if result, err = t.tx.Exec(archiveTransactionSQL, params...); err != nil {
		return dbe(err)
	}

	if nRows, _ := result.RowsAffected(); nRows == 0 {
		return dberr.ErrNotFound
	}

	//FIXME: COMPLETE AUDIT LOG
	_ = auditLog

	return nil
}

const countTransactionsSQL = "SELECT count(id), status FROM transactions WHERE archived=:archived GROUP BY status"

func (s *Store) CountTransactions(ctx context.Context) (counts *models.TransactionCounts, err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if counts, err = tx.CountTransactions(); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return counts, nil
}

func (t *Tx) CountTransactions() (counts *models.TransactionCounts, err error) {

	counts = &models.TransactionCounts{
		Active:   make(map[string]int),
		Archived: make(map[string]int),
	}

	if err = t.countTransactions(counts, false); err != nil {
		return nil, err
	}

	if err = t.countTransactions(counts, true); err != nil {
		return nil, err
	}

	return counts, nil
}

func (t *Tx) countTransactions(counts *models.TransactionCounts, archived bool) (err error) {
	var rows *sql.Rows
	if rows, err = t.tx.Query(countTransactionsSQL, sql.Named("archived", archived)); err != nil {
		return dbe(err)
	}
	defer rows.Close()

	for rows.Next() {
		var count int
		var status string
		if err = rows.Scan(&count, &status); err != nil {
			return err
		}

		if archived {
			counts.Archived[status] = count
		} else {
			counts.Active[status] = count
		}
	}

	return rows.Err()
}

const transactionStateSQL = "SELECT archived, status FROM transactions WHERE id=:id"

func (s *Store) TransactionState(ctx context.Context, transactionID uuid.UUID) (archived bool, status enum.Status, err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return archived, status, err
	}
	defer tx.Rollback()

	if archived, status, err = tx.TransactionState(transactionID); err != nil {
		return archived, status, err
	}

	if err = tx.Commit(); err != nil {
		return false, enum.StatusUnspecified, err
	}

	return archived, status, nil
}

func (t *Tx) TransactionState(transactionID uuid.UUID) (archived bool, status enum.Status, err error) {
	row := t.tx.QueryRow(transactionStateSQL, sql.Named("id", transactionID))
	if err = row.Scan(&archived, &status); err != nil {
		return archived, status, dbe(err)
	}
	return archived, status, nil
}

//===========================================================================
// Secure Envelopes CRUD Interface
//===========================================================================

const listSecureEnvelopesSQL = "SELECT * FROM secure_envelopes WHERE envelope_id=:envelopeID ORDER BY timestamp DESC"

func (s *Store) ListSecureEnvelopes(ctx context.Context, txID uuid.UUID, page *models.PageInfo) (out *models.SecureEnvelopePage, err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if out, err = tx.ListSecureEnvelopes(txID, page); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return out, nil
}

func (t *Tx) ListSecureEnvelopes(txID uuid.UUID, page *models.PageInfo) (out *models.SecureEnvelopePage, err error) {
	var transaction *models.Transaction
	if transaction, err = t.retrieveTransaction(txID); err != nil {
		return nil, err
	}

	// TODO: handle pagination
	out = &models.SecureEnvelopePage{
		Envelopes: make([]*models.SecureEnvelope, 0),
		Page:      models.PageInfoFrom(page),
	}

	var rows *sql.Rows
	if rows, err = t.tx.Query(listSecureEnvelopesSQL, sql.Named("envelopeID", txID)); err != nil {
		return nil, dbe(err)
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
	return out, nil
}

func (t *Tx) associateSecureEnvelopes(transaction *models.Transaction) (err error) {
	var rows *sql.Rows
	if rows, err = t.tx.Query(listSecureEnvelopesSQL, sql.Named("envelopeID", transaction.ID)); err != nil {
		return dbe(err)
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

const createSecureEnvelopeSQL = "INSERT INTO secure_envelopes (id, envelope_id, direction, remote, reply_to, is_error, encryption_key, hmac_secret, valid_hmac, timestamp, public_key, transfer_state, envelope, created, modified) VALUES (:id, :envelopeID, :direction, :remote, :replyTo, :isError, :encryptionKey, :hmacSecret, :validHMAC, :timestamp, :publicKey, :transferState, :envelope, :created, :modified)"

func (s *Store) CreateSecureEnvelope(ctx context.Context, env *models.SecureEnvelope, auditLog *models.ComplianceAuditLog) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.CreateSecureEnvelope(env, auditLog); err != nil {
		return err
	}

	return tx.Commit()
}

func (t *Tx) CreateSecureEnvelope(env *models.SecureEnvelope, auditLog *models.ComplianceAuditLog) (err error) {
	if !env.ID.IsZero() {
		return dberr.ErrNoIDOnCreate
	}

	if env.EnvelopeID == uuid.Nil {
		return dberr.ErrMissingReference
	}

	// Create IDs and model metadata updating the secure envelope in place.
	env.ID = ulid.MakeSecure()
	env.Created = time.Now()
	env.Modified = env.Created

	if _, err = t.tx.Exec(createSecureEnvelopeSQL, env.Params()...); err != nil {
		return dbe(err)
	}

	//FIXME: COMPLETE AUDIT LOG
	_ = auditLog

	return nil
}

const retrieveSecureEnvelopeSQL = "SELECT * FROM secure_envelopes WHERE id=:envID and envelope_id=:txID"

func (s *Store) RetrieveSecureEnvelope(ctx context.Context, txID uuid.UUID, envID ulid.ULID) (env *models.SecureEnvelope, err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if env, err = tx.RetrieveSecureEnvelope(txID, envID); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return env, nil
}

func (t *Tx) RetrieveSecureEnvelope(txID uuid.UUID, envID ulid.ULID) (env *models.SecureEnvelope, err error) {
	env = &models.SecureEnvelope{}
	if err = env.Scan(t.tx.QueryRow(retrieveSecureEnvelopeSQL, sql.Named("envID", envID), sql.Named("txID", txID))); err != nil {
		return nil, dbe(err)
	}
	return env, nil
}

const updateSecureEnvelopeSQL = "UPDATE secure_envelopes SET direction=:direction, remote=:remote, reply_to=:replyTo, is_error=:is_error, encryption_key=:encryptionKey, hmac_secret=:hmacSecret, valid_hmac=:validHMAC, timestamp=:timestamp, public_key=:publicKey, transfer_state=:transferState, envelope=:envelope, modified=:modified WHERE id=:id and envelope_id=:envelopeID"

func (s *Store) UpdateSecureEnvelope(ctx context.Context, env *models.SecureEnvelope, auditLog *models.ComplianceAuditLog) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.UpdateSecureEnvelope(env, auditLog); err != nil {
		return err
	}

	return tx.Commit()
}

func (t *Tx) UpdateSecureEnvelope(env *models.SecureEnvelope, auditLog *models.ComplianceAuditLog) (err error) {
	// Basic validation
	if env.ID.IsZero() {
		return dberr.ErrMissingID
	}

	if env.EnvelopeID == uuid.Nil {
		return dberr.ErrMissingReference
	}

	env.Modified = time.Now()

	var result sql.Result
	if result, err = t.tx.Exec(updateSecureEnvelopeSQL, env.Params()...); err != nil {
		return dbe(err)
	}

	if nRows, _ := result.RowsAffected(); nRows == 0 {
		return dberr.ErrNotFound
	}

	//FIXME: COMPLETE AUDIT LOG
	_ = auditLog

	return nil
}

const deleteSecureEnvelopeSQL = "DELETE FROM secure_envelopes WHERE id=:envID AND envelope_id=:txID"

func (s *Store) DeleteSecureEnvelope(ctx context.Context, txID uuid.UUID, envID ulid.ULID, auditLog *models.ComplianceAuditLog) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.DeleteSecureEnvelope(txID, envID, auditLog); err != nil {
		return err
	}

	return tx.Commit()
}

func (t *Tx) DeleteSecureEnvelope(txID uuid.UUID, envID ulid.ULID, auditLog *models.ComplianceAuditLog) (err error) {
	var result sql.Result
	if result, err = t.tx.Exec(deleteSecureEnvelopeSQL, sql.Named("txID", txID), sql.Named("envID", envID)); err != nil {
		return dbe(err)
	}

	if nRows, _ := result.RowsAffected(); nRows == 0 {
		return dberr.ErrNotFound
	}

	//FIXME: COMPLETE AUDIT LOG
	_ = auditLog

	return nil
}

//===========================================================================
// Secure Envelope Fetching
//===========================================================================

const (
	latestSecEnvSQL            = "SELECT * FROM secure_envelopes WHERE envelope_id=:envelopeID ORDER BY timestamp DESC LIMIT 1"
	latestSecEnvByDirectionSQL = "SELECT * FROM secure_envelopes WHERE envelope_id=:envelopeID AND direction=:direction ORDER BY timestamp DESC LIMIT 1"
)

func (s *Store) LatestSecureEnvelope(ctx context.Context, envelopeID uuid.UUID, direction enum.Direction) (env *models.SecureEnvelope, err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if env, err = tx.LatestSecureEnvelope(envelopeID, direction); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return env, nil
}

func (t *Tx) LatestSecureEnvelope(envelopeID uuid.UUID, direction enum.Direction) (env *models.SecureEnvelope, err error) {
	var result *sql.Row
	if direction == enum.DirectionUnknown || direction == enum.DirectionAny {
		// Get the latest secure envelope regardless of the direction
		result = t.tx.QueryRow(latestSecEnvSQL, sql.Named("envelopeID", envelopeID))
	} else {
		// Specify the direction to get the latest envelope for
		result = t.tx.QueryRow(latestSecEnvByDirectionSQL, sql.Named("envelopeID", envelopeID), sql.Named("direction", direction))
	}

	env = &models.SecureEnvelope{}
	if err = env.Scan(result); err != nil {
		return nil, dbe(err)
	}

	return env, nil
}

const (
	latestPayload               = "SELECT * FROM secure_envelopes WHERE envelope_id=:envelopeID AND is_error=false ORDER BY timestamp DESC LIMIT 1"
	latestPayloadByDirectionSQL = "SELECT * FROM secure_envelopes WHERE envelope_id=:envelopeID AND is_error=false AND direction=:direction ORDER BY timestamp DESC LIMIT 1"
)

func (s *Store) LatestPayloadEnvelope(ctx context.Context, envelopeID uuid.UUID, direction enum.Direction) (env *models.SecureEnvelope, err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if env, err = tx.LatestPayloadEnvelope(envelopeID, direction); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return env, nil
}

func (t *Tx) LatestPayloadEnvelope(envelopeID uuid.UUID, direction enum.Direction) (env *models.SecureEnvelope, err error) {
	var result *sql.Row
	if direction == enum.DirectionUnknown || direction == enum.DirectionAny {
		// Get the latest secure envelope regardless of the direction
		result = t.tx.QueryRow(latestPayload, sql.Named("envelopeID", envelopeID))
	} else {
		// Specify the direction to get the latest envelope for
		result = t.tx.QueryRow(latestPayloadByDirectionSQL, sql.Named("envelopeID", envelopeID), sql.Named("direction", direction))
	}

	env = &models.SecureEnvelope{}
	if err = env.Scan(result); err != nil {
		return nil, dbe(err)
	}
	return env, nil
}

//===========================================================================
// Prepared Transactions
//===========================================================================

const (
	transactionExistsSQL = "SELECT EXISTS(SELECT 1 FROM transactions WHERE id=:envelopeID)"
)

func (s *Store) PrepareTransaction(ctx context.Context, envelopeID uuid.UUID, auditLog *models.ComplianceAuditLog) (_ models.PreparedTransaction, err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return nil, err
	}

	// Check if a transaction exists with the specified envelope ID
	var exists bool
	if err = tx.QueryRow(transactionExistsSQL, sql.Named("envelopeID", envelopeID)).Scan(&exists); err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("transaction existence check failed: %w", err)
	}

	// If the transaction does not exist then create a stub transaction with the ID
	// Fill in all not nullable fields with a default placeholder.
	if !exists {
		now := time.Now()
		transaction := &models.Transaction{
			ID:           envelopeID,
			Source:       enum.SourceUnknown,
			Status:       enum.StatusDraft,
			Counterparty: models.CounterpartyUnknown,
			VirtualAsset: models.VirtualAssetUnknown,
			Amount:       0.0,
			Created:      now,
			Modified:     now,
		}

		if _, err = tx.Exec(createTransactionSQL, transaction.Params()...); err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("create transaction failed: %w", dbe(err))
		}

		//FIXME: CREATE THE AUDIT LOG
		_ = auditLog
	}

	// Create the prepared transaction for the user to interact with
	return &PreparedTransaction{tx: tx, envelopeID: envelopeID, created: !exists}, nil
}

type PreparedTransaction struct {
	tx         *Tx
	envelopeID uuid.UUID
	created    bool
}

func (p *PreparedTransaction) Created() bool {
	return p.created
}

func (p *PreparedTransaction) Fetch() (transaction *models.Transaction, err error) {
	transaction = &models.Transaction{}
	if err = transaction.Scan(p.tx.QueryRow(retrieveTransactionSQL, sql.Named("id", p.envelopeID))); err != nil {
		return nil, dbe(err)
	}
	return transaction, nil
}

func (p *PreparedTransaction) Update(in *models.Transaction, auditLog *models.ComplianceAuditLog) (err error) {
	// Ensure that the input transaction matches the prepared transaction
	if in.ID != uuid.Nil && in.ID != p.envelopeID {
		return dberr.ErrIDMismatch
	}

	// Fetch the previous transaction and update from the input only non-zero values
	var orig *models.Transaction
	if orig, err = p.Fetch(); err != nil {
		return err
	}

	// Update orig with incoming values and updated modified timestamp
	orig.Update(in)
	orig.Modified = time.Now()

	if _, err = p.tx.Exec(updateTransactionSQL, orig.Params()...); err != nil {
		return fmt.Errorf("could not update transaction: %w", dbe(err))
	}

	//FIXME: CREATE THE AUDIT LOG
	_ = auditLog

	return nil
}

const (
	lookupTRISACounterpartySQL      = "SELECT * FROM counterparties WHERE registered_directory=:registeredDirectory AND directory_id=:directoryID"
	lookupCounterpartyCommonNameSQL = "SELECT * FROM counterparties WHERE common_name=:commonName LIMIT 1"
	updateTransferCounterpartySQL   = "UPDATE transactions SET counterparty=:counterparty, counterparty_id=:counterpartyID WHERE id=:txID"
)

// TODO: this method needs to be tested extensively!!
func (p *PreparedTransaction) AddCounterparty(in *models.Counterparty, auditLog *models.ComplianceAuditLog) (err error) {
	// Lookup counterparty information in the database
	switch {
	case !in.ID.IsZero():
		// Populate the counterparty record from the database
		if err = in.Scan(p.tx.QueryRow(retreiveCounterpartySQL, sql.Named("id", in.ID))); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return dberr.ErrNotFound
			}
			return fmt.Errorf("unable to lookup counterparty by id: %w", err)
		}
	case in.RegisteredDirectory.String != "" && in.DirectoryID.String != "":
		// Lookup the counterparty record by directory information in the database
		if err = in.Scan(p.tx.QueryRow(lookupTRISACounterpartySQL, sql.Named("registeredDirectory", in.RegisteredDirectory), sql.Named("directoryID", in.DirectoryID))); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// NOTE: if a valid TRISA peer sends a message before directory sync,
				// then this record will not be found; create the temporary TRISA record
				// prior to sync just in case.
				in.ID = ulid.MakeSecure()
				in.Created = time.Now()
				in.Modified = in.Created

				if _, err = p.tx.Exec(createCounterpartySQL, in.Params()...); err != nil {
					return fmt.Errorf("unable to create counterparty with directory id: %w", dbe(err))
				}
			} else {
				return fmt.Errorf("unable to lookup counterparty by directory id: %w", dbe(err))
			}
		}
	case in.CommonName != "":
		// Lookup the counterparty record by unique endpoint information
		if err = in.Scan(p.tx.QueryRow(lookupCounterpartyCommonNameSQL, sql.Named("commonName", in.CommonName))); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return dberr.ErrNotFound
			}
			return fmt.Errorf("unable to lookup counterparty by common name: %w", err)
		}
	default:
		// In this case, we're pretty sure the counterparty is not in the database
		// so we should try to add the counterparty and hope for the best ...
		in.ID = ulid.MakeSecure()
		in.Created = time.Now()
		in.Modified = in.Created

		if _, err = p.tx.Exec(createCounterpartySQL, in.Params()...); err != nil {
			return fmt.Errorf("unable to create counterparty: %w", dbe(err))
		}
	}

	// Update the transaction with the counterparty information
	params := []any{
		sql.Named("txID", p.envelopeID),
		sql.Named("counterparty", in.Name),
		sql.Named("counterpartyID", in.ID),
	}

	if _, err = p.tx.Exec(updateTransferCounterpartySQL, params...); err != nil {
		return fmt.Errorf("could not update transaction with counterparty info: %w", dbe(err))
	}

	//FIXME: CREATE THE AUDIT LOG
	_ = auditLog

	return nil
}

func (p *PreparedTransaction) UpdateCounterparty(in *models.Counterparty, auditLog *models.ComplianceAuditLog) (err error) {
	return p.tx.UpdateCounterparty(in, auditLog)
}

func (p *PreparedTransaction) LookupCounterparty(field, value string) (*models.Counterparty, error) {
	return p.tx.LookupCounterparty(field, value)
}

func (p *PreparedTransaction) AddEnvelope(in *models.SecureEnvelope, auditLog *models.ComplianceAuditLog) (err error) {
	if in.EnvelopeID != uuid.Nil && in.EnvelopeID != p.envelopeID {
		return dberr.ErrIDMismatch
	}

	if !in.ID.IsZero() {
		return dberr.ErrNoIDOnCreate
	}

	in.ID = ulid.MakeSecure()
	in.EnvelopeID = p.envelopeID
	in.Created = time.Now()
	in.Modified = in.Created

	if _, err = p.tx.Exec(createSecureEnvelopeSQL, in.Params()...); err != nil {
		return fmt.Errorf("could not add secure envelope: %w", dbe(err))
	}

	//FIXME: CREATE THE AUDIT LOG
	_ = auditLog

	return nil
}

func (p *PreparedTransaction) CreateSunrise(in *models.Sunrise, auditLog *models.ComplianceAuditLog) error {
	return p.tx.CreateSunrise(in, auditLog)
}

func (p *PreparedTransaction) UpdateSunrise(in *models.Sunrise, auditLog *models.ComplianceAuditLog) error {
	return p.tx.UpdateSunrise(in, auditLog)
}

func (p *PreparedTransaction) UpdateSunriseStatus(txID uuid.UUID, status enum.Status, auditLog *models.ComplianceAuditLog) error {
	return p.tx.UpdateSunriseStatus(txID, status, auditLog)
}

func (p *PreparedTransaction) Rollback() error {
	return p.tx.Rollback()
}

func (p *PreparedTransaction) Commit() error {
	return p.tx.Commit()
}
