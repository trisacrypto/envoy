package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	dberr "self-hosted-node/pkg/store/errors"
	"self-hosted-node/pkg/store/models"
	"self-hosted-node/pkg/ulids"

	"github.com/oklog/ulid/v2"
)

const listAccountsSQL = "SELECT id, customer_id, first_name, last_name, travel_address, created FROM accounts"

// Retrieve summary information for all accounts for the specified page, omitting
// crypto addresses and any other irrelevant information.
func (s *Store) ListAccounts(ctx context.Context, page *models.PageInfo) (out *models.AccountsPage, err error) {
	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// TODO: handle pagination
	out = &models.AccountsPage{
		Accounts: make([]*models.Account, 0),
	}

	var rows *sql.Rows
	if rows, err = tx.Query(listAccountsSQL); err != nil {
		// TODO: handle database specific errors
		return nil, err
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

	tx.Commit()
	return out, nil
}

const createAccountSQL = "INSERT INTO accounts (id, customer_id, first_name, last_name, travel_address, ivms101, created, modified) VALUES (:id, :customerID, :firstName, :lastName, :travelAddress, :ivms101, :created, :modified)"

// Create an account and any crypto addresses associated with the account.
func (s *Store) CreateAccount(ctx context.Context, account *models.Account) (err error) {
	// Basic validation
	if !ulids.IsZero(account.ID) {
		return dberr.ErrNoIDOnCreate
	}

	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	// Create IDs and model metadata, updating the account in place.
	account.ID = ulids.New()
	account.Created = time.Now()
	account.Modified = account.Created

	// Execute the insert into the database
	if _, err = tx.Exec(createAccountSQL, account.Params()...); err != nil {
		// TODO: handle constraint violations
		return err
	}

	// Insert the associated crypto addresses into the database
	addresses, _ := account.CryptoAddresses()
	for _, addr := range addresses {
		// Ensure the crypto address is associated with the new account
		addr.AccountID = account.ID
		if err = s.createCryptoAddress(tx, addr); err != nil {
			return err
		}
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

const retreiveAccountSQL = "SELECT * FROM accounts WHERE id=:id"

// Retrieve account detail information including all associated crypto addresses.
func (s *Store) RetrieveAccount(ctx context.Context, id ulid.ULID) (account *models.Account, err error) {
	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if account, err = retrieveAccount(tx, id); err != nil {
		return nil, err
	}

	// Retrieve associated crypto addresses with the account.
	if err = s.listCryptoAddresses(tx, account); err != nil {
		return nil, err
	}

	tx.Commit()
	return account, nil
}

func retrieveAccount(tx *sql.Tx, accountID ulid.ULID) (account *models.Account, err error) {
	account = &models.Account{}
	if err = account.Scan(tx.QueryRow(retreiveAccountSQL, sql.Named("id", accountID))); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, dberr.ErrNotFound
		}
		return nil, err
	}
	return account, nil
}

const updateAccountSQL = "UPDATE accounts SET customer_id=:customerID, first_name=:firstName, last_name=:lastName, travel_address=:travelAddress, ivms101=:ivms101, modified=:modified WHERE id=:id"

// Update account information; ignores any associated crypto addresses.
func (s *Store) UpdateAccount(ctx context.Context, a *models.Account) (err error) {
	// Basic validation
	if ulids.IsZero(a.ID) {
		return dberr.ErrMissingID
	}

	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	// Update modified timestamp (in place).
	a.Modified = time.Now()

	// Execute the update into the database
	var result sql.Result
	if result, err = tx.Exec(updateAccountSQL, a.Params()...); err != nil {
		// TODO: handle constraint violations
		return err
	} else if nRows, _ := result.RowsAffected(); nRows == 0 {
		return dberr.ErrNotFound
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

const deleteAccountSQL = "DELETE FROM accounts WHERE id=:id"

// Delete account and all associated crypto addresses
func (s *Store) DeleteAccount(ctx context.Context, id ulid.ULID) (err error) {
	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	var result sql.Result
	if result, err = tx.Exec(deleteAccountSQL, sql.Named("id", id)); err != nil {
		return err
	} else if nRows, _ := result.RowsAffected(); nRows == 0 {
		return dberr.ErrNotFound
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

const listCryptoAddressesSQL = "SELECT * FROM crypto_addresses WHERE account_id=:accountID"

// List crypto addresses associated with the specified accountID.
func (s *Store) ListCryptoAddresses(ctx context.Context, accountID ulid.ULID, page *models.PageInfo) (out *models.CryptoAddressPage, err error) {
	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Check to ensure the associated account exists
	var account *models.Account
	if account, err = retrieveAccount(tx, accountID); err != nil {
		return nil, err
	}

	// TODO: handle pagination
	out = &models.CryptoAddressPage{
		CryptoAddresses: make([]*models.CryptoAddress, 0),
	}

	var rows *sql.Rows
	if rows, err = tx.Query(listCryptoAddressesSQL, sql.Named("accountID", accountID)); err != nil {
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

	if errors.Is(rows.Err(), sql.ErrNoRows) {
		return nil, dberr.ErrNotFound
	}

	tx.Commit()
	return out, nil
}

func (s *Store) listCryptoAddresses(tx *sql.Tx, account *models.Account) (err error) {
	var rows *sql.Rows
	if rows, err = tx.Query(listCryptoAddressesSQL, sql.Named("accountID", account.ID)); err != nil {
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

func (s *Store) CreateCryptoAddress(ctx context.Context, addr *models.CryptoAddress) (err error) {
	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = s.createCryptoAddress(tx, addr); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (s *Store) createCryptoAddress(tx *sql.Tx, addr *models.CryptoAddress) (err error) {
	if !ulids.IsZero(addr.ID) {
		return dberr.ErrNoIDOnCreate
	}

	if ulids.IsZero(addr.AccountID) {
		return dberr.ErrMissingReference
	}

	// Create IDs and model metadata, updating the account in place.
	addr.ID = ulids.New()
	addr.Created = time.Now()
	addr.Modified = addr.Created

	if _, err = tx.Exec(createCryptoAddressSQL, addr.Params()...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dberr.ErrNotFound
		}

		// TODO: handle constraint violations
		return err
	}
	return nil
}

const retrieveCryptoAddressSQL = "SELECT * FROM crypto_addresses WHERE id=:cryptoAddressID and account_id=:accountID"

func (s *Store) RetrieveCryptoAddress(ctx context.Context, accountID, cryptoAddressID ulid.ULID) (addr *models.CryptoAddress, err error) {
	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	addr = &models.CryptoAddress{}
	if err = addr.Scan(tx.QueryRow(retrieveCryptoAddressSQL, sql.Named("cryptoAddressID", cryptoAddressID), sql.Named("accountID", accountID))); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, dberr.ErrNotFound
		}
		return nil, err
	}

	tx.Commit()
	return addr, nil
}

// TODO: this must be an upsert/delete since the data is being modified on the relation
const updateCryptoAddressSQL = "UPDATE crypto_addresses SET crypto_address=:cryptoAddress, network=:network, asset_type=:assetType, tag=:tag, travel_address=:travelAddress modified=:modified WHERE id=:id and account_id=:accountID"

func (s *Store) UpdateCryptoAddress(ctx context.Context, addr *models.CryptoAddress) (err error) {
	// Basic validation
	if ulids.IsZero(addr.ID) {
		return dberr.ErrMissingID
	}

	if ulids.IsZero(addr.AccountID) {
		return dberr.ErrMissingReference
	}

	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	// Update modified timestamp (in place).
	addr.Modified = time.Now()

	// Execute the update into the database
	var result sql.Result
	if result, err = tx.Exec(updateCryptoAddressSQL, addr.Params()...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dberr.ErrNotFound
		}

		// TODO: handle constraint violations
		return err
	} else if nRows, _ := result.RowsAffected(); nRows == 0 {
		return dberr.ErrNotFound
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

const deleteCryptoAddressSQL = "DELETE FROM crypto_addresses WHERE id=:cryptoAddressID and account_id=:accountID"

func (s *Store) DeleteCryptoAddress(ctx context.Context, accountID, cryptoAddressID ulid.ULID) (err error) {
	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	var result sql.Result
	if result, err = tx.Exec(deleteCryptoAddressSQL, sql.Named("cryptoAddressID", cryptoAddressID), sql.Named("accountID", accountID)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dberr.ErrNotFound
		}
		return err
	} else if nRows, _ := result.RowsAffected(); nRows == 0 {
		return dberr.ErrNotFound
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}
