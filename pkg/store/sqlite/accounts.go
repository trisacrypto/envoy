package sqlite

import (
	"context"
	"database/sql"
	"strings"
	"time"

	dberr "github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/models"

	"github.com/rs/zerolog/log"
	"go.rtnl.ai/ulid"
)

const listAccountsSQL = "SELECT a.id, a.customer_id, a.first_name, a.last_name, a.travel_address, a.ivms101 != :null, count(ca.id), a.created, a.modified FROM accounts a LEFT JOIN crypto_addresses ca ON a.id = ca.account_id GROUP BY a.id"

// Retrieve summary information for all accounts for the specified page, omitting
// crypto addresses and any other irrelevant information.
func (s *Store) ListAccounts(ctx context.Context, page *models.PageInfo) (out *models.AccountsPage, err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if out, err = tx.ListAccounts(page); err != nil {
		return nil, err
	}

	tx.Commit()
	return out, nil
}

// Retrieve summary information for all accounts for the specified page, omitting
// crypto addresses and any other irrelevant information.
func (t *Tx) ListAccounts(page *models.PageInfo) (out *models.AccountsPage, err error) {
	// TODO: handle pagination
	out = &models.AccountsPage{
		Accounts: make([]*models.Account, 0),
		Page:     models.PageInfoFrom(page),
	}

	var rows *sql.Rows
	if rows, err = t.tx.Query(listAccountsSQL, sql.Named("null", []byte("null"))); err != nil {
		return nil, dbe(err)
	}
	defer rows.Close()

	for rows.Next() {
		// Scan account into memory
		account := &models.Account{}
		if err = account.ScanSummary(rows); err != nil {
			return nil, err
		}

		// Ensure that addresses is non-nil and zero-valued
		account.SetCryptoAddresses(make([]*models.CryptoAddress, 0))

		// Append account to page
		out.Accounts = append(out.Accounts, account)
	}

	return out, nil
}

const createAccountSQL = "INSERT INTO accounts (id, customer_id, first_name, last_name, travel_address, ivms101, created, modified) VALUES (:id, :customerID, :firstName, :lastName, :travelAddress, :ivms101, :created, :modified)"

// Create an account and any crypto addresses associated with the account.
func (s *Store) CreateAccount(ctx context.Context, account *models.Account, auditLog *models.ComplianceAuditLog) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.CreateAccount(account, auditLog); err != nil {
		return err
	}

	return tx.Commit()
}

// Create an account and any crypto addresses associated with the account.
func (t *Tx) CreateAccount(account *models.Account, auditLog *models.ComplianceAuditLog) (err error) {
	// Basic validation
	if !account.ID.IsZero() {
		return dberr.ErrNoIDOnCreate
	}

	// Create IDs and model metadata, updating the account in place.
	account.ID = ulid.MakeSecure()
	account.Created = time.Now()
	account.Modified = account.Created

	// Create the travel address for the crypto address, logging errors without returning
	if t.mkta != nil {
		var travelAddress string
		if travelAddress, err = t.mkta(account); err != nil {
			log.Warn().Err(err).Str("type", "account").Str("id", account.ID.String()).Msg("could not assign travel address")
		}
		account.TravelAddress = sql.NullString{Valid: travelAddress != "", String: travelAddress}
	}

	// Execute the insert into the database
	if _, err = t.tx.Exec(createAccountSQL, account.Params()...); err != nil {
		return dbe(err)
	}

	// Insert the associated crypto addresses into the database
	addresses, _ := account.CryptoAddresses()
	for _, addr := range addresses {
		// Ensure the crypto address is associated with the new account
		addr.AccountID = account.ID
		//FIXME: COMPLETE AUDIT LOG
		if err = t.CreateCryptoAddress(addr, &models.ComplianceAuditLog{}); err != nil {
			return err
		}
	}

	//FIXME: CREATE THE AUDIT LOG
	_ = auditLog

	return nil
}

const lookupAccountSQL = "SELECT account_id FROM crypto_addresses WHERE crypto_address=:cryptoAddress"

// Lookup an account by an associated crypto address.
func (s *Store) LookupAccount(ctx context.Context, cryptoAddress string) (account *models.Account, err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if account, err = tx.LookupAccount(cryptoAddress); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return account, nil
}

// Lookup an account by an associated crypto address.
func (t *Tx) LookupAccount(cryptoAddress string) (account *models.Account, err error) {
	var accountID ulid.ULID
	if err = t.tx.QueryRow(lookupAccountSQL, sql.Named("cryptoAddress", cryptoAddress)).Scan(&accountID); err != nil {
		return nil, dbe(err)
	}

	if account, err = t.RetrieveAccount(accountID); err != nil {
		return nil, err
	}

	return account, nil
}

const retreiveAccountSQL = "SELECT * FROM accounts WHERE id=:id"

// Retrieve account detail information including all associated crypto addresses.
func (s *Store) RetrieveAccount(ctx context.Context, accountID ulid.ULID) (account *models.Account, err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if account, err = tx.RetrieveAccount(accountID); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return account, nil
}

// Retrieve account detail information including all associated crypto addresses.
func (t *Tx) RetrieveAccount(accountID ulid.ULID) (account *models.Account, err error) {
	account = &models.Account{}
	if err = account.Scan(t.tx.QueryRow(retreiveAccountSQL, sql.Named("id", accountID))); err != nil {
		return nil, dbe(err)
	}

	// Retrieve associated crypto addresses with the account.
	if err = t.listCryptoAddressesForAccount(account); err != nil {
		return nil, err
	}

	return account, nil
}

const updateAccountSQL = "UPDATE accounts SET customer_id=:customerID, first_name=:firstName, last_name=:lastName, travel_address=:travelAddress, ivms101=:ivms101, modified=:modified WHERE id=:id"

// Update account information; ignores any associated crypto addresses.
func (s *Store) UpdateAccount(ctx context.Context, account *models.Account, auditLog *models.ComplianceAuditLog) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.UpdateAccount(account, auditLog); err != nil {
		return err
	}

	return tx.Commit()
}

// Update account information; ignores any associated crypto addresses.
func (t *Tx) UpdateAccount(account *models.Account, auditLog *models.ComplianceAuditLog) (err error) {
	// Basic validation
	if account.ID.IsZero() {
		return dberr.ErrMissingID
	}

	// If the travel address is not set, then generate it (mirroring create behavior).
	if !account.TravelAddress.Valid && t.mkta != nil {
		var travelAddress string
		if travelAddress, err = t.mkta(account); err != nil {
			log.Warn().Err(err).Str("type", "account").Str("id", account.ID.String()).Msg("could not assign travel address")
		}
		account.TravelAddress = sql.NullString{Valid: travelAddress != "", String: travelAddress}
	}

	// Update modified timestamp (in place).
	account.Modified = time.Now()

	// Execute the update into the database
	var result sql.Result
	if result, err = t.tx.Exec(updateAccountSQL, account.Params()...); err != nil {
		return dbe(err)
	} else if nRows, _ := result.RowsAffected(); nRows == 0 {
		return dberr.ErrNotFound
	}

	//FIXME: CREATE THE AUDIT LOG
	_ = auditLog

	return nil
}

const deleteAccountSQL = "DELETE FROM accounts WHERE id=:id"

// Delete account and all associated crypto addresses
func (s *Store) DeleteAccount(ctx context.Context, accountID ulid.ULID, auditLog *models.ComplianceAuditLog) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.DeleteAccount(accountID, auditLog); err != nil {
		return err
	}

	return tx.Commit()
}

// Delete account and all associated crypto addresses
func (t *Tx) DeleteAccount(accountID ulid.ULID, auditLog *models.ComplianceAuditLog) (err error) {
	var result sql.Result
	if result, err = t.tx.Exec(deleteAccountSQL, sql.Named("id", accountID)); err != nil {
		return dbe(err)
	}

	if nRows, _ := result.RowsAffected(); nRows == 0 {
		return dberr.ErrNotFound
	}

	//FIXME: CREATE THE AUDIT LOG
	_ = auditLog

	return nil
}

const listAccountTxnsSQL = `
	WITH wallet AS (SELECT crypto_address FROM crypto_addresses WHERE account_id=:accountID)
	SELECT t.id, t.source, t.status, t.counterparty, t.counterparty_id, t.originator, t.originator_address, t.beneficiary, t.beneficiary_address, t.virtual_asset, t.amount, t.archived, t.archived_on, t.last_update, t.modified, t.created, count(e.id) AS numEnvelopes
		FROM transactions t
		LEFT JOIN secure_envelopes e ON t.id=e.envelope_id
		WHERE t.archived=:archives AND (
			t.originator_address IN (SELECT * FROM wallet) OR
			t.beneficiary_address IN (SELECT * FROM wallet)
		)
		GROUP BY t.id
		ORDER BY t.created DESC`

// List all transactions that have one of the account wallet addresses in either the
// originator or beneficiary wallet address fields.
func (s *Store) ListAccountTransactions(ctx context.Context, accountID ulid.ULID, page *models.TransactionPageInfo) (out *models.TransactionPage, err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if out, err = tx.ListAccountTransactions(accountID, page); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return out, nil
}

// List all transactions that have one of the account wallet addresses in either the
// originator or beneficiary wallet address fields.
func (t *Tx) ListAccountTransactions(accountID ulid.ULID, page *models.TransactionPageInfo) (out *models.TransactionPage, err error) {
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
	query := listAccountTxnsSQL
	params := []interface{}{
		sql.Named("accountID", accountID),
		sql.Named("archives", page.Archives),
	}

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

		query = "WITH txns AS (" + listAccountTxnsSQL + ") SELECT * FROM txns WHERE "
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

const listCryptoAddressesSQL = "SELECT * FROM crypto_addresses WHERE account_id=:accountID"

// List crypto addresses associated with the specified accountID.
func (s *Store) ListCryptoAddresses(ctx context.Context, accountID ulid.ULID, page *models.PageInfo) (out *models.CryptoAddressPage, err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if out, err = tx.ListCryptoAddresses(accountID, page); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return out, nil
}

// List crypto addresses associated with the specified accountID.
func (t *Tx) ListCryptoAddresses(accountID ulid.ULID, page *models.PageInfo) (out *models.CryptoAddressPage, err error) {
	// Check to ensure the associated account exists
	var account *models.Account
	if account, err = t.RetrieveAccount(accountID); err != nil {
		return nil, err
	}

	// TODO: handle pagination
	out = &models.CryptoAddressPage{
		CryptoAddresses: make([]*models.CryptoAddress, 0),
		Page:            models.PageInfoFrom(page),
	}

	var rows *sql.Rows
	if rows, err = t.tx.Query(listCryptoAddressesSQL, sql.Named("accountID", accountID)); err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		addr := &models.CryptoAddress{}
		if err = addr.Scan(rows); err != nil {
			return nil, err
		}

		addr.SetAccount(account)
		out.CryptoAddresses = append(out.CryptoAddresses, addr)
	}

	if err = rows.Err(); err != nil {
		return nil, dbe(err)
	}

	return out, nil
}

// Special list function to retrieve all related crypto addresses for an account, this
// method is used by the account store not the crypto address store; but is placed here
// for easy reference to the listCryptoAddressesSQL query.
func (t *Tx) listCryptoAddressesForAccount(account *models.Account) (err error) {
	var rows *sql.Rows
	if rows, err = t.tx.Query(listCryptoAddressesSQL, sql.Named("accountID", account.ID)); err != nil {
		return err
	}
	defer rows.Close()

	addresses := make([]*models.CryptoAddress, 0)
	for rows.Next() {
		addr := &models.CryptoAddress{}
		if err = addr.Scan(rows); err != nil {
			return err
		}
		addr.SetAccount(account)
		addresses = append(addresses, addr)
	}

	account.SetCryptoAddresses(addresses)
	return nil
}

const createCryptoAddressSQL = "INSERT INTO crypto_addresses (id, account_id, crypto_address, network, asset_type, tag, travel_address, created, modified) VALUES (:id, :accountID, :cryptoAddress, :network, :assetType, :tag, :travelAddress, :created, :modified)"

func (s *Store) CreateCryptoAddress(ctx context.Context, addr *models.CryptoAddress, auditLog *models.ComplianceAuditLog) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.CreateCryptoAddress(addr, auditLog); err != nil {
		return err
	}

	return tx.Commit()
}

func (t *Tx) CreateCryptoAddress(addr *models.CryptoAddress, auditLog *models.ComplianceAuditLog) (err error) {
	if !addr.ID.IsZero() {
		return dberr.ErrNoIDOnCreate
	}

	if addr.AccountID.IsZero() {
		return dberr.ErrMissingReference
	}

	// Create IDs and model metadata, updating the crypto address in place.
	addr.ID = ulid.MakeSecure()
	addr.Created = time.Now()
	addr.Modified = addr.Created

	// Create the travel address for the crypto address, logging errors without returning
	if t.mkta != nil {
		var travelAddress string
		if travelAddress, err = t.mkta(addr); err != nil {
			log.Warn().Err(err).Str("type", "crypto_address").Str("id", addr.ID.String()).Msg("could not assign travel address")
		}
		addr.TravelAddress = sql.NullString{Valid: travelAddress != "", String: travelAddress}
	}

	if _, err = t.tx.Exec(createCryptoAddressSQL, addr.Params()...); err != nil {
		return dbe(err)
	}

	//FIXME: CREATE THE AUDIT LOG
	_ = auditLog

	return nil
}

const retrieveCryptoAddressSQL = "SELECT * FROM crypto_addresses WHERE id=:cryptoAddressID and account_id=:accountID"

func (s *Store) RetrieveCryptoAddress(ctx context.Context, accountID, cryptoAddressID ulid.ULID) (addr *models.CryptoAddress, err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if addr, err = tx.RetrieveCryptoAddress(accountID, cryptoAddressID); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return addr, nil
}

func (t *Tx) RetrieveCryptoAddress(accountID, cryptoAddressID ulid.ULID) (addr *models.CryptoAddress, err error) {
	addr = &models.CryptoAddress{}
	if err = addr.Scan(t.tx.QueryRow(retrieveCryptoAddressSQL, sql.Named("cryptoAddressID", cryptoAddressID), sql.Named("accountID", accountID))); err != nil {
		return nil, dbe(err)
	}

	// TODO: retrieve account and associate it with the crypto address.
	return addr, nil
}

// TODO: this must be an upsert/delete since the data is being modified on the relation
const updateCryptoAddressSQL = "UPDATE crypto_addresses SET crypto_address=:cryptoAddress, network=:network, asset_type=:assetType, tag=:tag, travel_address=:travelAddress, modified=:modified WHERE id=:id and account_id=:accountID"

func (s *Store) UpdateCryptoAddress(ctx context.Context, addr *models.CryptoAddress, auditLog *models.ComplianceAuditLog) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.UpdateCryptoAddress(addr, auditLog); err != nil {
		return err
	}

	return tx.Commit()
}

func (t *Tx) UpdateCryptoAddress(addr *models.CryptoAddress, auditLog *models.ComplianceAuditLog) (err error) {
	// Basic validation
	if addr.ID.IsZero() {
		return dberr.ErrMissingID
	}

	if addr.AccountID.IsZero() {
		return dberr.ErrMissingReference
	}

	// If the travel address is not set, then generate it (mirroring create behavior).
	if !addr.TravelAddress.Valid && t.mkta != nil {
		var travelAddress string
		if travelAddress, err = t.mkta(addr); err != nil {
			log.Warn().Err(err).Str("type", "crypto_address").Str("id", addr.ID.String()).Msg("could not assign travel address")
		}
		addr.TravelAddress = sql.NullString{Valid: travelAddress != "", String: travelAddress}
	}

	// Update modified timestamp (in place).
	addr.Modified = time.Now()

	// Execute the update into the database
	var result sql.Result
	if result, err = t.tx.Exec(updateCryptoAddressSQL, addr.Params()...); err != nil {
		return dbe(err)
	} else if nRows, _ := result.RowsAffected(); nRows == 0 {
		return dberr.ErrNotFound
	}

	//FIXME: CREATE THE AUDIT LOG
	_ = auditLog

	return nil
}

const deleteCryptoAddressSQL = "DELETE FROM crypto_addresses WHERE id=:cryptoAddressID and account_id=:accountID"

func (s *Store) DeleteCryptoAddress(ctx context.Context, accountID, cryptoAddressID ulid.ULID, auditLog *models.ComplianceAuditLog) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.DeleteCryptoAddress(accountID, cryptoAddressID, auditLog); err != nil {
		return err
	}

	return tx.Commit()
}

func (t *Tx) DeleteCryptoAddress(accountID, cryptoAddressID ulid.ULID, auditLog *models.ComplianceAuditLog) (err error) {
	var result sql.Result
	if result, err = t.tx.Exec(deleteCryptoAddressSQL, sql.Named("cryptoAddressID", cryptoAddressID), sql.Named("accountID", accountID)); err != nil {
		return dbe(err)
	} else if nRows, _ := result.RowsAffected(); nRows == 0 {
		return dberr.ErrNotFound
	}

	//FIXME: CREATE THE AUDIT LOG
	_ = auditLog

	return nil
}
