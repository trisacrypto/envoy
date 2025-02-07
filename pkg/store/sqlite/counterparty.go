package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	dberr "github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/models"

	"go.rtnl.ai/ulid"
)

const listCounterpartiesSQL = "SELECT id, source, protocol, endpoint, name, website, country, created FROM counterparties"

func (s *Store) ListCounterparties(ctx context.Context, page *models.PageInfo) (out *models.CounterpartyPage, err error) {
	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// TODO: handle pagination
	out = &models.CounterpartyPage{
		Counterparties: make([]*models.Counterparty, 0),
		Page:           models.PageInfoFrom(page),
	}

	var rows *sql.Rows
	if rows, err = tx.Query(listCounterpartiesSQL); err != nil {
		// TODO: handle database specific errors
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		// Scan counterparty into memory
		counterparty := &models.Counterparty{}
		if err = counterparty.ScanSummary(rows); err != nil {
			return nil, err
		}

		// Ensure that contacts is non-nil and zero-valued
		counterparty.SetContacts(make([]*models.Contact, 0))

		// Append counterparty to the page
		out.Counterparties = append(out.Counterparties, counterparty)
	}

	tx.Commit()
	return out, nil
}

const listCounterpartySourceInfoSQL = "SELECT id, source, directory_id, registered_directory, protocol FROM counterparties WHERE source=:source"

func (s *Store) ListCounterpartySourceInfo(ctx context.Context, source string) (out []*models.CounterpartySourceInfo, err error) {
	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	out = make([]*models.CounterpartySourceInfo, 0)

	var rows *sql.Rows
	if rows, err = tx.Query(listCounterpartySourceInfoSQL, sql.Named("source", source)); err != nil {
		// TODO: handle database specific errors
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		// Scan counterparty source info into memory
		info := &models.CounterpartySourceInfo{}
		if err = info.Scan(rows); err != nil {
			return nil, err
		}

		out = append(out, info)
	}

	tx.Commit()
	return out, nil
}

const createCounterpartySQL = "INSERT INTO counterparties (id, source, directory_id, registered_directory, protocol, common_name, endpoint, name, website, country, business_category, vasp_categories, verified_on, ivms101, created, modified) VALUES (:id, :source, :directoryID, :registeredDirectory, :protocol, :commonName, :endpoint, :name, :website, :country, :businessCategory, :vaspCategories, :verifiedOn, :ivms101, :created, :modified)"

func (s *Store) CreateCounterparty(ctx context.Context, counterparty *models.Counterparty) (err error) {
	// Basic validation
	if !counterparty.ID.IsZero() {
		return dberr.ErrNoIDOnCreate
	}

	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = s.createCounterparty(tx, counterparty); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Store) createCounterparty(tx *sql.Tx, counterparty *models.Counterparty) (err error) {
	// Update the model metadata in place and create a new ID
	counterparty.ID = ulid.MakeSecure()
	counterparty.Created = time.Now()
	counterparty.Modified = counterparty.Created

	// Insert the counterparty
	if _, err = tx.Exec(createCounterpartySQL, counterparty.Params()...); err != nil {
		// TODO: handle constraint violations
		return err
	}

	// Insert any associated contacts with the counterparty
	contacts, _ := counterparty.Contacts()
	for _, contact := range contacts {
		contact.CounterpartyID = counterparty.ID
		if err = s.createContact(tx, contact); err != nil {
			return err
		}
	}

	return nil
}

const retreiveCounterpartySQL = "SELECT * FROM counterparties WHERE id=:id"

func (s *Store) RetrieveCounterparty(ctx context.Context, counterpartyID ulid.ULID) (counterparty *models.Counterparty, err error) {
	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if counterparty, err = retrieveCounterparty(tx, counterpartyID); err != nil {
		return nil, err
	}

	if err = s.listContacts(tx, counterparty); err != nil {
		return nil, err
	}

	tx.Commit()
	return counterparty, nil
}

func retrieveCounterparty(tx *sql.Tx, counterpartyID ulid.ULID) (counterparty *models.Counterparty, err error) {
	counterparty = &models.Counterparty{}
	if err = counterparty.Scan(tx.QueryRow(retreiveCounterpartySQL, sql.Named("id", counterpartyID))); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, dberr.ErrNotFound
		}
		return nil, err
	}
	return counterparty, nil
}

const lookupCounterpartySQL = "SELECT * FROM counterparties WHERE common_name=:commonName"

func (s *Store) LookupCounterparty(ctx context.Context, commonName string) (counterparty *models.Counterparty, err error) {
	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	counterparty = &models.Counterparty{}
	if err = counterparty.Scan(tx.QueryRow(lookupCounterpartySQL, sql.Named("commonName", commonName))); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, dberr.ErrNotFound
		}
		return nil, err
	}

	tx.Commit()
	return counterparty, nil
}

const updateCounterpartySQL = "UPDATE counterparties SET source=:source, directory_id=:directoryID, registered_directory=:registeredDirectory, protocol=:protocol, common_name=:commonName, endpoint=:endpoint, name=:name, website=:website, country=:country, business_category=:businessCategory, vasp_categories=:vaspCategories, verified_on=:verifiedOn, ivms101=:ivms101, modified=:modified WHERE id=:id"

func (s *Store) UpdateCounterparty(ctx context.Context, counterparty *models.Counterparty) (err error) {
	// Basic validation before starting a transaction
	if counterparty.ID.IsZero() {
		return dberr.ErrMissingID
	}

	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = updateCounterparty(tx, counterparty); err != nil {
		return err
	}

	return tx.Commit()
}

func updateCounterparty(tx *sql.Tx, counterparty *models.Counterparty) (err error) {
	if counterparty.ID.IsZero() {
		return dberr.ErrMissingID
	}

	// Update modified timestamp (in place).
	counterparty.Modified = time.Now()

	// Execute the update into the database
	var result sql.Result
	if result, err = tx.Exec(updateCounterpartySQL, counterparty.Params()...); err != nil {
		// TODO: handle constraint violations
		return err
	} else if nRows, _ := result.RowsAffected(); nRows == 0 {
		return dberr.ErrNotFound
	}

	return nil
}

const deleteCounterpartySQL = "DELETE FROM counterparties WHERE id=:id"

func (s *Store) DeleteCounterparty(ctx context.Context, counterpartyID ulid.ULID) (err error) {
	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	var result sql.Result
	if result, err = tx.Exec(deleteCounterpartySQL, sql.Named("id", counterpartyID)); err != nil {
		return err
	} else if nRows, _ := result.RowsAffected(); nRows == 0 {
		return dberr.ErrNotFound
	}

	return tx.Commit()
}

const listContactsSQL = "SELECT * FROM contacts WHERE counterparty_id=:counterpartyID"

// List contacts associated with the specified counterparty.
func (s *Store) ListContacts(ctx context.Context, counterpartyID ulid.ULID, page *models.PageInfo) (out *models.ContactsPage, err error) {
	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Check to ensure the associated counterparty exists
	var counterparty *models.Counterparty
	if counterparty, err = retrieveCounterparty(tx, counterpartyID); err != nil {
		return nil, err
	}

	// TODO: handle pagination
	out = &models.ContactsPage{
		Contacts: make([]*models.Contact, 0),
		Page:     models.PageInfoFrom(page),
	}

	var rows *sql.Rows
	if rows, err = tx.Query(listContactsSQL, sql.Named("counterpartyID", counterpartyID)); err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		contact := &models.Contact{}
		if err = contact.Scan(rows); err != nil {
			return nil, err
		}

		contact.SetCounterparty(counterparty)
		out.Contacts = append(out.Contacts, contact)
	}

	if errors.Is(rows.Err(), sql.ErrNoRows) {
		return nil, dberr.ErrNotFound
	}

	tx.Commit()
	return out, nil
}

func (s *Store) listContacts(tx *sql.Tx, counterparty *models.Counterparty) (err error) {
	var rows *sql.Rows
	if rows, err = tx.Query(listContactsSQL, sql.Named("counterpartyID", counterparty.ID)); err != nil {
		return err
	}
	defer rows.Close()

	contacts := make([]*models.Contact, 0)
	for rows.Next() {
		contact := &models.Contact{}
		if err = contact.Scan(rows); err != nil {
			return err
		}

		contact.SetCounterparty(counterparty)
		contacts = append(contacts, contact)
	}

	counterparty.SetContacts(contacts)
	return nil
}

const createContactSQL = "INSERT INTO contacts (id, name, email, role, counterparty_id, created, modified) VALUES (:id, :name, :email, :role, :counterpartyID, :created, :modified)"

func (s *Store) CreateContact(ctx context.Context, contact *models.Contact) (err error) {
	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = s.createContact(tx, contact); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Store) createContact(tx *sql.Tx, contact *models.Contact) (err error) {
	if !contact.ID.IsZero() {
		return dberr.ErrNoIDOnCreate
	}

	if contact.CounterpartyID.IsZero() {
		return dberr.ErrMissingReference
	}

	// Update the model metadata in place and create a new ID
	contact.ID = ulid.MakeSecure()
	contact.Created = time.Now()
	contact.Modified = contact.Created

	if _, err = tx.Exec(createContactSQL, contact.Params()...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dberr.ErrNotFound
		}

		// TODO: handle constraint violations
		return err
	}
	return nil
}

const retrieveContactSQL = "SELECT * FROM contacts WHERE id=:id and counterparty_id=:counterpartyID"

func (s *Store) RetrieveContact(ctx context.Context, contactID, counterpartyID ulid.ULID) (contact *models.Contact, err error) {
	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	contact = &models.Contact{}
	if err = contact.Scan(tx.QueryRow(retrieveContactSQL, sql.Named("id", contactID), sql.Named("counterpartyID", counterpartyID))); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, dberr.ErrNotFound
		}
		return nil, err
	}

	// TODO: retrieve counterparty and associate it with the contact.

	tx.Commit()
	return contact, nil
}

// TODO: this must be an upsert/delete since the data is being modified on the relation
const updateContactSQL = "UPDATE contacts SET name=:name, email=:email, role=:role, modified=:modified WHERE id=:id AND counterparty_id=:counterpartyID"

func (s *Store) UpdateContact(ctx context.Context, contact *models.Contact) (err error) {
	// Basic validation
	if contact.ID.IsZero() {
		return dberr.ErrMissingID
	}

	if contact.CounterpartyID.IsZero() {
		return dberr.ErrMissingReference
	}

	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	// Update modified timestamp (in place).
	contact.Modified = time.Now()

	// Execute the update into the database
	var result sql.Result
	if result, err = tx.Exec(updateContactSQL, contact.Params()...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dberr.ErrNotFound
		}

		// TODO: handle constraint violations
		return err
	} else if nRows, _ := result.RowsAffected(); nRows == 0 {
		return dberr.ErrNotFound
	}

	return tx.Commit()
}

const deleteContact = "DELETE FROM contacts WHERE id=:id AND counterparty_id=:counterpartyID"

func (s *Store) DeleteContact(ctx context.Context, contactID, counterpartyID ulid.ULID) (err error) {
	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	var result sql.Result
	if result, err = tx.Exec(deleteContact, sql.Named("id", contactID), sql.Named("counterpartyID", counterpartyID)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dberr.ErrNotFound
		}
		return err
	} else if nRows, _ := result.RowsAffected(); nRows == 0 {
		return dberr.ErrNotFound
	}

	return tx.Commit()
}
