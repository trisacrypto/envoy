package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/trisacrypto/envoy/pkg/enum"
	dberr "github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/models"

	"go.rtnl.ai/ulid"
)

const (
	listCounterpartiesSQL   = "SELECT id, source, protocol, endpoint, name, website, country, verified_on, created FROM counterparties ORDER BY name ASC"
	filterCounterpartiesSQL = "SELECT id, source, protocol, endpoint, name, website, country, verified_on, created FROM counterparties WHERE source=:source ORDER BY name ASC"
)

func (s *Store) ListCounterparties(ctx context.Context, page *models.CounterpartyPageInfo) (out *models.CounterpartyPage, err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if out, err = tx.ListCounterparties(page); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return out, nil
}

func (t *Tx) ListCounterparties(page *models.CounterpartyPageInfo) (out *models.CounterpartyPage, err error) {
	// TODO: handle pagination
	out = &models.CounterpartyPage{
		Counterparties: make([]*models.Counterparty, 0),
		Page:           &models.CounterpartyPageInfo{PageInfo: *models.PageInfoFrom(&page.PageInfo), Source: page.Source},
	}

	var rows *sql.Rows
	if page.Source != "" {
		if rows, err = t.tx.Query(filterCounterpartiesSQL, sql.Named("source", page.Source)); err != nil {
			return nil, dbe(err)
		}
	} else {
		if rows, err = t.tx.Query(listCounterpartiesSQL); err != nil {
			return nil, dbe(err)
		}
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

	return out, nil
}

const listCounterpartySourceInfoSQL = "SELECT id, source, directory_id, registered_directory, protocol FROM counterparties WHERE source=:source"

func (s *Store) ListCounterpartySourceInfo(ctx context.Context, source enum.Source) (out []*models.CounterpartySourceInfo, err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if out, err = tx.ListCounterpartySourceInfo(source); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return out, nil
}

func (t *Tx) ListCounterpartySourceInfo(source enum.Source) (out []*models.CounterpartySourceInfo, err error) {
	out = make([]*models.CounterpartySourceInfo, 0)

	var rows *sql.Rows
	if rows, err = t.tx.Query(listCounterpartySourceInfoSQL, sql.Named("source", source)); err != nil {
		return nil, dbe(err)
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

	return out, nil
}

const createCounterpartySQL = "INSERT INTO counterparties (id, source, directory_id, registered_directory, protocol, common_name, endpoint, name, website, country, business_category, vasp_categories, verified_on, ivms101, lei, created, modified) VALUES (:id, :source, :directoryID, :registeredDirectory, :protocol, :commonName, :endpoint, :name, :website, :country, :businessCategory, :vaspCategories, :verifiedOn, :ivms101, :lei, :created, :modified)"

func (s *Store) CreateCounterparty(ctx context.Context, counterparty *models.Counterparty) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.CreateCounterparty(counterparty); err != nil {
		return err
	}

	return tx.Commit()
}

func (t *Tx) CreateCounterparty(counterparty *models.Counterparty) (err error) {
	// Basic validation
	if !counterparty.ID.IsZero() {
		return dberr.ErrNoIDOnCreate
	}

	// Update the model metadata in place and create a new ID
	counterparty.ID = ulid.MakeSecure()
	counterparty.Created = time.Now()
	counterparty.Modified = counterparty.Created

	// Insert the counterparty
	if _, err = t.tx.Exec(createCounterpartySQL, counterparty.Params()...); err != nil {
		return dbe(err)
	}

	// Insert any associated contacts with the counterparty
	contacts, _ := counterparty.Contacts()
	for _, contact := range contacts {
		contact.CounterpartyID = counterparty.ID
		if err = t.CreateContact(contact); err != nil {
			return fmt.Errorf("could not create contact for counterparty: %w", err)
		}
	}

	return nil
}

const retreiveCounterpartySQL = "SELECT * FROM counterparties WHERE id=:id"

func (s *Store) RetrieveCounterparty(ctx context.Context, counterpartyID ulid.ULID) (counterparty *models.Counterparty, err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// tx.RetrieveCounterparty returns the counterparty with it's contacts
	if counterparty, err = tx.RetrieveCounterparty(counterpartyID); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return counterparty, nil
}

func (t *Tx) RetrieveCounterparty(counterpartyID ulid.ULID) (counterparty *models.Counterparty, err error) {
	counterparty = &models.Counterparty{}
	if err = counterparty.Scan(t.tx.QueryRow(retreiveCounterpartySQL, sql.Named("id", counterpartyID))); err != nil {
		return nil, dbe(err)
	}

	// Add associated contacts to this counterparty
	if err = t.listCounterpartyContacts(counterparty); err != nil {
		return nil, err
	}

	return counterparty, nil
}

const (
	countCounterpartySQL  = "SELECT count(id) FROM counterparties WHERE %s=:%s"
	lookupCounterpartySQL = "SELECT * FROM counterparties WHERE %s=:%s LIMIT 1"
)

func (s *Store) LookupCounterparty(ctx context.Context, field, value string) (counterparty *models.Counterparty, err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if counterparty, err = tx.LookupCounterparty(field, value); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return counterparty, nil
}

func (t *Tx) LookupCounterparty(field, value string) (counterparty *models.Counterparty, err error) {
	// Check to make sure that the counterparty exists and there is only 1 matching counterparty
	var count int
	query := fmt.Sprintf(countCounterpartySQL, field, field)
	if err = t.tx.QueryRow(query, sql.Named(field, value)).Scan(&count); err != nil {
		return nil, dbe(err)
	}

	switch {
	case count == 0:
		return nil, dberr.ErrNotFound
	case count > 1:
		return nil, dberr.ErrAmbiguous
	}

	counterparty = &models.Counterparty{}
	query = fmt.Sprintf(lookupCounterpartySQL, field, field)
	if err = counterparty.Scan(t.tx.QueryRow(query, sql.Named(field, value))); err != nil {
		return nil, dbe(err)
	}

	return counterparty, nil
}

const updateCounterpartySQL = "UPDATE counterparties SET source=:source, directory_id=:directoryID, registered_directory=:registeredDirectory, protocol=:protocol, common_name=:commonName, endpoint=:endpoint, name=:name, website=:website, country=:country, business_category=:businessCategory, vasp_categories=:vaspCategories, verified_on=:verifiedOn, ivms101=:ivms101, lei=:lei, modified=:modified WHERE id=:id"

func (s *Store) UpdateCounterparty(ctx context.Context, counterparty *models.Counterparty) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.UpdateCounterparty(counterparty); err != nil {
		return err
	}

	return tx.Commit()
}

func (t *Tx) UpdateCounterparty(counterparty *models.Counterparty) (err error) {
	if counterparty.ID.IsZero() {
		return dberr.ErrMissingID
	}

	// Update modified timestamp (in place).
	counterparty.Modified = time.Now()

	// Execute the update into the database
	var result sql.Result
	if result, err = t.tx.Exec(updateCounterpartySQL, counterparty.Params()...); err != nil {
		return dbe(err)
	} else if nRows, _ := result.RowsAffected(); nRows == 0 {
		return dberr.ErrNotFound
	}

	return nil
}

const deleteCounterpartySQL = "DELETE FROM counterparties WHERE id=:id"

func (s *Store) DeleteCounterparty(ctx context.Context, counterpartyID ulid.ULID) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.DeleteCounterparty(counterpartyID); err != nil {
		return err
	}

	return tx.Commit()
}

func (t *Tx) DeleteCounterparty(counterpartyID ulid.ULID) (err error) {
	var result sql.Result
	if result, err = t.tx.Exec(deleteCounterpartySQL, sql.Named("id", counterpartyID)); err != nil {
		return dbe(err)
	} else if nRows, _ := result.RowsAffected(); nRows == 0 {
		return dberr.ErrNotFound
	}
	return nil
}

const listContactsSQL = "SELECT * FROM contacts WHERE counterparty_id=:counterpartyID"

// List contacts associated with the specified counterparty. The counterparty can either
// be a ULID of the counterparty ID or a pointer to the Counterparty model. If the
// ID is specified then the associated counterparty is retrieved from the database and
// attached to all returned contacts. If the model is specified, then the contacts,
// will be attached to the model.
func (s *Store) ListContacts(ctx context.Context, counterparty any, page *models.PageInfo) (out *models.ContactsPage, err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if out, err = tx.ListContacts(counterparty, page); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return out, nil
}

func (t *Tx) ListContacts(counterparty any, page *models.PageInfo) (out *models.ContactsPage, err error) {
	var (
		counterpartyID    ulid.ULID
		counterpartyModel *models.Counterparty
	)

	if counterparty == nil {
		return nil, dberr.ErrMissingAssociation
	}

	// Handle the input counterparty and retrieve the counterparty model if necessary.
	switch c := counterparty.(type) {
	case *models.Counterparty:
		counterpartyID = c.ID
		counterpartyModel = c
	case ulid.ULID:
		counterpartyID = c
		if counterpartyModel, err = t.RetrieveCounterparty(counterpartyID); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("invalid type for counterparty: %T", counterparty)
	}

	// TODO: handle pagination
	out = &models.ContactsPage{
		Contacts: make([]*models.Contact, 0),
		Page:     models.PageInfoFrom(page),
	}

	var rows *sql.Rows
	if rows, err = t.tx.Query(listContactsSQL, sql.Named("counterpartyID", counterpartyID)); err != nil {
		return nil, dbe(err)
	}
	defer rows.Close()

	for rows.Next() {
		contact := &models.Contact{}
		if err = contact.Scan(rows); err != nil {
			return nil, err
		}

		contact.SetCounterparty(counterpartyModel)
		out.Contacts = append(out.Contacts, contact)
	}

	if errors.Is(rows.Err(), sql.ErrNoRows) {
		return nil, dberr.ErrNotFound
	}

	// Associate the contacts with the counterparty model (useful if a pointer to
	// a counterparty was passed in).
	counterpartyModel.SetContacts(out.Contacts)
	return out, nil
}

// This is a helper function to list and set contacts for a specific counterparty; it is
// only used by the RetrieveCounterparty method but is defined here for easy reference
// to the listContactsSQL query.
func (t *Tx) listCounterpartyContacts(counterparty *models.Counterparty) (err error) {
	var rows *sql.Rows
	if rows, err = t.tx.Query(listContactsSQL, sql.Named("counterpartyID", counterparty.ID)); err != nil {
		return dbe(err)
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
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.CreateContact(contact); err != nil {
		return err
	}

	return tx.Commit()
}

func (t *Tx) CreateContact(contact *models.Contact) (err error) {
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

	if _, err = t.tx.Exec(createContactSQL, contact.Params()...); err != nil {
		return dbe(err)
	}

	return nil
}

const retrieveContactSQL = "SELECT * FROM contacts WHERE id=:id and counterparty_id=:counterpartyID"

// Retrieve the contact with the specified ID and associate it with the
// specified counterparty. The counterparty can either be the ULID of the counterparty
// or a pointer to the Counterparty model. If the ID is specified then the
// associated counterparty is retrieved from the database and attached to the
// contact. Note that if a pointer to the Counterparty model is specified, it is not
// modified in place.
func (s *Store) RetrieveContact(ctx context.Context, contactID, counterparty any) (contact *models.Contact, err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if contact, err = tx.RetrieveContact(contactID, counterparty); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return contact, err
}

func (t *Tx) RetrieveContact(contactID, counterparty any) (contact *models.Contact, err error) {
	var (
		counterpartyID    ulid.ULID
		counterpartyModel *models.Counterparty
	)

	if counterparty == nil {
		return nil, dberr.ErrMissingAssociation
	}

	// Handle the input counterparty and retrieve the counterparty model if necessary.
	switch c := counterparty.(type) {
	case *models.Counterparty:
		counterpartyID = c.ID
		counterpartyModel = c
	case ulid.ULID:
		counterpartyID = c
		if counterpartyModel, err = t.RetrieveCounterparty(counterpartyID); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("invalid type for counterparty: %T", counterparty)
	}

	// Retrieve the contact
	contact = &models.Contact{}
	if err = contact.Scan(t.tx.QueryRow(retrieveContactSQL, sql.Named("id", contactID), sql.Named("counterpartyID", counterpartyID))); err != nil {
		return nil, dbe(err)
	}

	// Associate the contact with the counterparty model.
	contact.SetCounterparty(counterpartyModel)
	return contact, nil
}

// TODO: this must be an upsert/delete since the data is being modified on the relation
const updateContactSQL = "UPDATE contacts SET name=:name, email=:email, role=:role, modified=:modified WHERE id=:id AND counterparty_id=:counterpartyID"

func (s *Store) UpdateContact(ctx context.Context, contact *models.Contact) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.UpdateContact(contact); err != nil {
		return err
	}

	return tx.Commit()
}

func (t *Tx) UpdateContact(contact *models.Contact) (err error) {
	// Basic validation
	if contact.ID.IsZero() {
		return dberr.ErrMissingID
	}

	if contact.CounterpartyID.IsZero() {
		return dberr.ErrMissingReference
	}

	// Update modified timestamp (in place).
	contact.Modified = time.Now()

	// Execute the update into the database
	var result sql.Result
	if result, err = t.tx.Exec(updateContactSQL, contact.Params()...); err != nil {
		return dbe(err)
	} else if nRows, _ := result.RowsAffected(); nRows == 0 {
		return dberr.ErrNotFound
	}

	return nil
}

const deleteContact = "DELETE FROM contacts WHERE id=:id AND counterparty_id=:counterpartyID"

// Delete contact associated with the specified counterparty. The counterparty can
// either be a ULID of the counterparty or a pointer to the Counterparty model. If the
// ID is specified then the associated counterparty is used to identify the contact to
// delete. If the model is specified, then the contact is deleted from the model as well.
func (s *Store) DeleteContact(ctx context.Context, contactID, counterparty any) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.DeleteContact(contactID, counterparty); err != nil {
		return err
	}
	return tx.Commit()
}

func (t *Tx) DeleteContact(contactID, counterparty any) (err error) {
	var (
		counterpartyID    ulid.ULID
		counterpartyModel *models.Counterparty
	)

	if counterparty == nil {
		return dberr.ErrMissingAssociation
	}

	// Handle the input counterparty.
	switch c := counterparty.(type) {
	case *models.Counterparty:
		counterpartyID = c.ID
		counterpartyModel = c
	case ulid.ULID:
		counterpartyID = c
	default:
		return fmt.Errorf("invalid type for counterparty: %T", counterparty)
	}

	var result sql.Result
	if result, err = t.tx.Exec(deleteContact, sql.Named("id", contactID), sql.Named("counterpartyID", counterpartyID)); err != nil {
		return dbe(err)
	} else if nRows, _ := result.RowsAffected(); nRows == 0 {
		return dberr.ErrNotFound
	}

	// If a counterparty model was passed in, update its contacts to delete the deleted contact.
	if counterpartyModel != nil {
		if contacts, cerr := counterpartyModel.Contacts(); cerr == nil {
			// Remove the contact from the counterparty model
			for i, contact := range contacts {
				if contact.ID == contactID {
					counterpartyModel.SetContacts(append(contacts[:i], contacts[i+1:]...))
					break
				}
			}
		}
	}

	return nil
}
