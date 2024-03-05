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

const listAccountsSQL = "SELECT * FROM accounts"

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
		a := &models.Account{}
		if err = rows.Scan(&a.ID, &a.CustomerID, &a.FirstName, &a.LastName, &a.TravelAddress, &a.IVMSRecord, &a.Created, &a.Modified); err != nil {
			return nil, err
		}

		// Scan related crypto addresses into memory
		if err = s.retrieveCryptoAddresses(tx, a); err != nil {
			return nil, err
		}

		// Append account to page
		out.Accounts = append(out.Accounts, a)
	}

	tx.Commit()
	return out, nil
}

const createAccountSQL = "INSERT INTO accounts (id, customer_id, first_name, last_name, travel_address, ivms101, created, modified) VALUES (:id, :customerID, :firstName, :lastName, :travelAddress, :ivms101, :created, :modified)"

func (s *Store) CreateAccount(ctx context.Context, a *models.Account) (err error) {
	// Basic validation
	if !ulids.IsZero(a.ID) {
		return dberr.ErrNoIDOnCreate
	}

	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	// Create IDs and model metadata, updating the account in place.
	a.ID = ulids.New()
	a.Created = time.Now()
	a.Modified = a.Created

	// Create parameters array
	params := []any{
		sql.Named("id", a.ID),
		sql.Named("customerID", a.CustomerID),
		sql.Named("firstName", a.FirstName),
		sql.Named("lastName", a.LastName),
		sql.Named("travelAddress", a.TravelAddress),
		sql.Named("ivms101", a.IVMSRecord),
		sql.Named("created", a.Created),
		sql.Named("modified", a.Modified),
	}

	// Execute the insert into the database
	if _, err = tx.Exec(createAccountSQL, params...); err != nil {
		// TODO: handle constraint violations
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

const retreiveAccountSQL = "SELECT * FROM accounts WHERE id=:id"

func (s *Store) RetrieveAccount(ctx context.Context, id ulid.ULID) (a *models.Account, err error) {
	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	a = &models.Account{}
	if err = tx.QueryRow(retreiveAccountSQL, sql.Named("id", id)).Scan(&a.ID, &a.CustomerID, &a.FirstName, &a.LastName, &a.TravelAddress, &a.IVMSRecord, &a.Created, &a.Modified); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, dberr.ErrNotFound
		}
		return nil, err
	}

	tx.Commit()
	return a, nil
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
	a.Modified = a.Created

	// Create parameters array
	params := []any{
		sql.Named("id", a.ID),
		sql.Named("customerID", a.CustomerID),
		sql.Named("firstName", a.FirstName),
		sql.Named("lastName", a.LastName),
		sql.Named("travelAddress", a.TravelAddress),
		sql.Named("ivms101", a.IVMSRecord),
		sql.Named("modified", a.Modified),
	}

	// Execute the update into the database
	if _, err = tx.Exec(updateAccountSQL, params...); err != nil {
		// TODO: handle constraint violations
		return err
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

const getAccountCryptoAddressesSQL = "SELECT * FROM crypto_addresses WHERE account_id=:accountID"

func (s *Store) retrieveCryptoAddresses(tx *sql.Tx, account *models.Account) (err error) {
	var rows *sql.Rows
	if rows, err = tx.Query(getAccountCryptoAddressesSQL, sql.Named("accountID", account.ID)); err != nil {
		return err
	}
	defer rows.Close()

	addresses := make([]*models.CryptoAddress, 0)
	for rows.Next() {
		a := &models.CryptoAddress{}
		if err = rows.Scan(&a.ID, &a.AccountID, &a.CryptoAddress, &a.Network, &a.AssetType, &a.Tag, &a.Created, &a.Modified); err != nil {
			return err
		}
		a.SetAccount(account)
	}

	account.SetCryptoAddresses(addresses)
	return nil
}

func (s *Store) ListCryptoAddresses(ctx context.Context, page *models.PageInfo) (*models.CryptoAddressPage, error) {
	return nil, dberr.ErrNotImplemented
}

func (s *Store) CreateCryptoAddress(context.Context, *models.CryptoAddress) error {
	return dberr.ErrNotImplemented
}

func (s *Store) RetrieveCryptoAddress(ctx context.Context, id ulid.ULID) (*models.CryptoAddress, error) {
	return nil, dberr.ErrNotImplemented
}

func (s *Store) UpdateCryptoAddress(context.Context, *models.CryptoAddress) error {
	return dberr.ErrNotImplemented
}

func (s *Store) DeleteCryptoAddress(ctx context.Context, id ulid.ULID) error {
	return dberr.ErrNotImplemented
}
