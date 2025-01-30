package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	dberr "github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/envoy/pkg/ulids"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
)

//===========================================================================
// Transaction CRUD interface
//==========================================================================

const listTransactionsSQL = "SELECT t.*, count(e.id) AS numEnvelopes FROM transactions t LEFT JOIN secure_envelopes e ON t.id=e.envelope_id GROUP BY t.id ORDER BY t.created DESC"

func (s *Store) ListTransactions(ctx context.Context, page *models.PageInfo) (out *models.TransactionPage, err error) {
	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// TODO: handle pagination
	out = &models.TransactionPage{
		Transactions: make([]*models.Transaction, 0),
		Page:         models.PageInfoFrom(page),
	}

	var rows *sql.Rows
	if rows, err = tx.Query(listTransactionsSQL); err != nil {
		// TODO: handle database specific errors
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		transaction := &models.Transaction{}
		if err = transaction.ScanWithCount(rows); err != nil {
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
	if transaction.ID != uuid.Nil {
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

const updateTransactionSQL = "UPDATE transactions SET source=:source, status=:status, counterparty=:counterparty, counterparty_id=:counterpartyID, originator=:originator, originator_address=:originatorAddress, beneficiary=:beneficiary, beneficiary_address=:beneficiaryAddress, virtual_asset=:virtualAsset, amount=:amount, last_update=:lastUpdate, modified=:modified WHERE id=:id"

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

	// NOTE: do not update `LastUpdate` timestamp - this refers to when a secure envelope is sent/received.

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

//===========================================================================
// Secure Envelopes CRUD Interface
//===========================================================================

const listSecureEnvelopesSQL = "SELECT * FROM secure_envelopes WHERE envelope_id=:envelopeID ORDER BY timestamp DESC"

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
		Page:      models.PageInfoFrom(page),
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
	return out, nil
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

const createSecureEnvelopeSQL = "INSERT INTO secure_envelopes (id, envelope_id, direction, remote, reply_to, is_error, encryption_key, hmac_secret, valid_hmac, timestamp, public_key, transfer_state, envelope, created, modified) VALUES (:id, :envelopeID, :direction, :remote, :replyTo, :isError, :encryptionKey, :hmacSecret, :validHMAC, :timestamp, :publicKey, :transferState, :envelope, :created, :modified)"

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

const updateSecureEnvelopeSQL = "UPDATE secure_envelopes SET direction=:direction, remote=:remote, reply_to=:replyTo, is_error=:is_error, encryption_key=:encryptionKey, hmac_secret=:hmacSecret, valid_hmac=:validHMAC, timestamp=:timestamp, public_key=:publicKey, transfer_state=:transferState, envelope=:envelope, modified=:modified WHERE id=:id and envelope_id=:envelopeID"

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

//===========================================================================
// Secure Envelope Fetching
//===========================================================================

const (
	latestSecEnvSQL            = "SELECT * FROM secure_envelopes WHERE envelope_id=:envelopeID ORDER BY timestamp DESC LIMIT 1"
	latestSecEnvByDirectionSQL = "SELECT * FROM secure_envelopes WHERE envelope_id=:envelopeID AND direction=:direction ORDER BY timestamp DESC LIMIT 1"
)

func (s *Store) LatestSecureEnvelope(ctx context.Context, envelopeID uuid.UUID, direction string) (env *models.SecureEnvelope, err error) {
	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var result *sql.Row
	if direction == "" || direction == models.DirectionAny {
		// Get the latest secure envelope regardless of the direction
		result = tx.QueryRow(latestSecEnvSQL, sql.Named("envelopeID", envelopeID))
	} else {
		// Specify the direction to get the latest envelope for
		result = tx.QueryRow(latestSecEnvByDirectionSQL, sql.Named("envelopeID", envelopeID), sql.Named("direction", direction))
	}

	env = &models.SecureEnvelope{}
	if err = env.Scan(result); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, dberr.ErrNotFound
		}
		return nil, err
	}

	tx.Commit()
	return env, nil
}

const (
	latestPayload               = "SELECT * FROM secure_envelopes WHERE envelope_id=:envelopeID AND is_error=false ORDER BY timestamp DESC LIMIT 1"
	latestPayloadByDirectionSQL = "SELECT * FROM secure_envelopes WHERE envelope_id=:envelopeID AND is_error=false AND direction=:direction ORDER BY timestamp DESC LIMIT 1"
)

func (s *Store) LatestPayloadEnvelope(ctx context.Context, envelopeID uuid.UUID, direction string) (env *models.SecureEnvelope, err error) {
	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var result *sql.Row
	if direction == "" || direction == models.DirectionAny {
		// Get the latest secure envelope regardless of the direction
		result = tx.QueryRow(latestPayload, sql.Named("envelopeID", envelopeID))
	} else {
		// Specify the direction to get the latest envelope for
		result = tx.QueryRow(latestPayloadByDirectionSQL, sql.Named("envelopeID", envelopeID), sql.Named("direction", direction))
	}

	env = &models.SecureEnvelope{}
	if err = env.Scan(result); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, dberr.ErrNotFound
		}
		return nil, err
	}

	tx.Commit()
	return env, nil
}

//===========================================================================
// Prepared Transactions
//===========================================================================

const (
	transactionExistsSQL = "SELECT EXISTS(SELECT 1 FROM transactions WHERE id=:envelopeID)"
)

func (s *Store) PrepareTransaction(ctx context.Context, envelopeID uuid.UUID) (_ models.PreparedTransaction, err error) {
	var tx *sql.Tx
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
			Source:       models.SourceUnknown,
			Status:       models.StatusDraft,
			Counterparty: models.CounterpartyUnknown,
			VirtualAsset: models.VirtualAssetUnknown,
			Amount:       0.0,
			Created:      now,
			Modified:     now,
		}

		if _, err = tx.Exec(createTransactionSQL, transaction.Params()...); err != nil {
			// TODO: handle constraint violations
			tx.Rollback()
			return nil, fmt.Errorf("create transaction failed: %w", err)
		}
	}

	// Create the prepared transaction for the user to interact with
	return &PreparedTransaction{tx: tx, envelopeID: envelopeID, created: !exists}, nil
}

type PreparedTransaction struct {
	tx         *sql.Tx
	envelopeID uuid.UUID
	created    bool
}

func (p *PreparedTransaction) Created() bool {
	return p.created
}

func (p *PreparedTransaction) Fetch() (transaction *models.Transaction, err error) {
	transaction = &models.Transaction{}
	if err = transaction.Scan(p.tx.QueryRow(retrieveTransactionSQL, sql.Named("id", p.envelopeID))); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, dberr.ErrNotFound
		}
		return nil, err
	}
	return transaction, nil
}

func (p *PreparedTransaction) Update(in *models.Transaction) (err error) {
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
		// TODO: handle constraint violations specially
		return fmt.Errorf("could not update transaction: %w", err)
	}
	return nil
}

const (
	lookupTRISACounterpartySQL      = "SELECT * FROM counterparties WHERE registered_directory=:registeredDirectory AND directory_id=:directoryID"
	lookupCounterpartyCommonNameSQL = "SELECT * FROM counterparties WHERE common_name=:commonName LIMIT 1"
	updateTransferCounterpartySQL   = "UPDATE transactions SET counterparty=:counterparty, counterparty_id=:counterpartyID WHERE id=:txID"
)

// TODO: this method needs to be tested extensively!!
func (p *PreparedTransaction) AddCounterparty(in *models.Counterparty) (err error) {
	// Lookup counterparty information in the database
	switch {
	case !ulids.IsZero(in.ID):
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
				in.ID = ulids.New()
				in.Created = time.Now()
				in.Modified = in.Created

				if _, err = p.tx.Exec(createCounterpartySQL, in.Params()...); err != nil {
					// TODO: handle constraints specifically
					return fmt.Errorf("unable to create counterparty with directory id: %w", err)
				}
			} else {
				return fmt.Errorf("unable to lookup counterparty by directory id: %w", err)
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
		in.ID = ulids.New()
		in.Created = time.Now()
		in.Modified = in.Created

		if _, err = p.tx.Exec(createCounterpartySQL, in.Params()...); err != nil {
			// TODO: handle constraints specifically
			return fmt.Errorf("unable to create counterparty: %w", err)
		}
	}

	// Update the transaction with the counterparty information
	params := []any{
		sql.Named("txID", p.envelopeID),
		sql.Named("counterparty", in.Name),
		sql.Named("counterpartyID", in.ID),
	}

	if _, err = p.tx.Exec(updateTransferCounterpartySQL, params...); err != nil {
		// TODO: handle constraint violations if necessary
		return fmt.Errorf("could not update transaction with counterparty info: %w", err)
	}
	return nil
}

func (p *PreparedTransaction) UpdateCounterparty(in *models.Counterparty) (err error) {
	return updateCounterparty(p.tx, in)
}

func (p *PreparedTransaction) AddEnvelope(in *models.SecureEnvelope) (err error) {
	if in.EnvelopeID != uuid.Nil && in.EnvelopeID != p.envelopeID {
		return dberr.ErrIDMismatch
	}

	if !ulids.IsZero(in.ID) {
		return dberr.ErrNoIDOnCreate
	}

	in.ID = ulids.New()
	in.EnvelopeID = p.envelopeID
	in.Created = time.Now()
	in.Modified = in.Created

	if _, err = p.tx.Exec(createSecureEnvelopeSQL, in.Params()...); err != nil {
		// TODO: handle constraint violations specially
		return fmt.Errorf("could not add secure envelope: %w", err)
	}
	return nil
}

func (p *PreparedTransaction) CreateSunrise(in *models.Sunrise) error {
	return createSunrise(p.tx, in)
}

func (p *PreparedTransaction) UpdateSunrise(in *models.Sunrise) error {
	return updateSunrise(p.tx, in)
}

func (p *PreparedTransaction) Rollback() error {
	return p.tx.Rollback()
}

func (p *PreparedTransaction) Commit() error {
	return p.tx.Commit()
}
