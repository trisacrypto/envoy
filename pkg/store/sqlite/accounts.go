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

const listAccountsSQL = "SELECT id, customer_id, first_name, last_name, travel_address FROM accounts"

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

func (s *Store) RetrieveAccount(ctx context.Context, id ulid.ULID) (account *models.Account, err error) {
	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	account = &models.Account{}
	if err = account.Scan(tx.QueryRow(retreiveAccountSQL, sql.Named("id", id))); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, dberr.ErrNotFound
		}
		return nil, err
	}

	// Retrieve associated crypto addresses with the account.
	if err = s.retrieveCryptoAddresses(tx, account); err != nil {
		return nil, err
	}

	tx.Commit()
	return account, nil
}

const updateAccountSQL = "UPDATE accounts SET customer_id=:customerID, first_name=:firstName, last_name=:lastName, travel_address=:travelAddress, ivms101=:ivms101, modified=:modified WHERE id=:id"

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
	if _, err = tx.Exec(updateAccountSQL, a.Params()...); err != nil {
		// TODO: handle constraint violations
		return err
	}

	// Update the associated crypto addresses into the database
	addresses, _ := a.CryptoAddresses()
	for _, addr := range addresses {
		// Ensure the crypto address is associated with the new account
		addr.AccountID = a.ID
		if err = s.updateCryptoAddress(tx, addr); err != nil {
			return err
		}
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

const deleteAccountSQL = "DELETE FROM accounts WHERE id=:id"

func (s *Store) DeleteAccount(ctx context.Context, id ulid.ULID) (err error) {
	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err = tx.Exec(deleteAccountSQL, sql.Named("id", id)); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

const retrieveCryptoAddressesSQL = "SELECT * FROM crypto_addresses WHERE account_id=:accountID"

func (s *Store) retrieveCryptoAddresses(tx *sql.Tx, account *models.Account) (err error) {
	var rows *sql.Rows
	if rows, err = tx.Query(retrieveCryptoAddressesSQL, sql.Named("accountID", account.ID)); err != nil {
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

const createCryptoAddressSQL = "INSERT INTO crypto_addresses (id, account_id, crypto_address, network, asset_type, tag, created, modified) VALUES (:id, :accountID, :cryptoAddress, :network, :assetType, :tag, :created, :modified)"

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
		// TODO: handle constraint violations
		return err
	}
	return nil
}

// TODO: this must be an upsert/delete since the data is being modified on the relation
const updateCryptoAddressSQL = "UPDATE crypto_addresses SET account_id=:accountID, crypto_address=:cryptoAddress, network=:network, asset_type=:assetType, tag=:tag, modified=:modified WHERE id=:id"

func (s *Store) updateCryptoAddress(tx *sql.Tx, addr *models.CryptoAddress) (err error) {
	// Basic validation
	if ulids.IsZero(addr.ID) {
		return dberr.ErrMissingID
	}

	if ulids.IsZero(addr.AccountID) {
		return dberr.ErrMissingReference
	}

	// Update modified timestamp (in place).
	addr.Modified = time.Now()

	// Execute the update into the database
	if _, err = tx.Exec(updateCryptoAddressSQL, addr.Params()...); err != nil {
		// TODO: handle constraint violations
		return err
	}

	return nil
}
