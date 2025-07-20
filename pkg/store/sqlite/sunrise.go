package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"go.rtnl.ai/ulid"

	"github.com/google/uuid"
	"github.com/trisacrypto/envoy/pkg/enum"
	dberr "github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/models"
)

const listSunriseSQL = "SELECT id, envelope_id, expiration, status, sent_on, verified_on FROM sunrise"

// Retrieve sunrise messages from the database and return them as a paginated list.
func (s *Store) ListSunrise(ctx context.Context, page *models.PageInfo) (out *models.SunrisePage, err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if out, err = tx.ListSunrise(page); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return out, nil
}

func (t *Tx) ListSunrise(page *models.PageInfo) (out *models.SunrisePage, err error) {
	// TODO: handle pagination
	out = &models.SunrisePage{
		Messages: make([]*models.Sunrise, 0),
		Page:     models.PageInfoFrom(page),
	}

	var rows *sql.Rows
	if rows, err = t.tx.Query(listSunriseSQL); err != nil {
		return nil, dbe(err)
	}
	defer rows.Close()

	for rows.Next() {
		// Scan sunrise into memory
		msg := &models.Sunrise{}
		if err = msg.ScanSummary(rows); err != nil {
			return nil, err
		}

		// Append sunrise to page
		out.Messages = append(out.Messages, msg)
	}

	return out, nil
}

const createSunriseSQL = "INSERT INTO sunrise (id, envelope_id, email, expiration, signature, status, sent_on, verified_on, created, modified) VALUES (:id, :envelopeID, :email, :expiration, :signature, :status, :sentOn, :verifiedOn, :created, :modified)"

// Create a sunrise message in the database.
func (s *Store) CreateSunrise(ctx context.Context, msg *models.Sunrise, auditLog *models.ComplianceAuditLog) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.CreateSunrise(msg, auditLog); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

// Create a sunrise message in the database.
func (t *Tx) CreateSunrise(msg *models.Sunrise, auditLog *models.ComplianceAuditLog) (err error) {
	// Basic validation
	if !msg.ID.IsZero() {
		return dberr.ErrNoIDOnCreate
	}

	// Create IDs and model metadata, updating the sunrise message in place.
	msg.ID = ulid.MakeSecure()
	msg.Created = time.Now()
	msg.Modified = msg.Created

	// Execute the insert into the database
	if _, err = t.tx.Exec(createSunriseSQL, msg.Params()...); err != nil {
		return dbe(err)
	}

	//FIXME: CREATE THE AUDIT LOG
	_ = auditLog

	return nil
}

const retrieveSunriseSQL = "SELECT * FROM sunrise WHERE id=:id"

// Retrieve sunrise message detail information.
func (s *Store) RetrieveSunrise(ctx context.Context, id ulid.ULID) (msg *models.Sunrise, err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if msg, err = tx.RetrieveSunrise(id); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return msg, nil
}

// Retrieve sunrise message detail information.
func (t *Tx) RetrieveSunrise(sunriseID ulid.ULID) (msg *models.Sunrise, err error) {
	msg = &models.Sunrise{}
	if err = msg.Scan(t.tx.QueryRow(retrieveSunriseSQL, sql.Named("id", sunriseID))); err != nil {
		return nil, dbe(err)
	}
	return msg, nil
}

const updateSunriseSQL = "UPDATE sunrise SET envelope_id=:envelopeID, email=:email, expiration=:expiration, signature=:signature, status=:status, sent_on=:sentOn, verified_on=:verifiedOn, modified=:modified WHERE id=:id"

// Update sunrise message information.
func (s *Store) UpdateSunrise(ctx context.Context, msg *models.Sunrise, auditLog *models.ComplianceAuditLog) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.UpdateSunrise(msg, auditLog); err != nil {
		return err
	}

	return tx.Commit()
}

// Update sunrise message information.
func (t *Tx) UpdateSunrise(msg *models.Sunrise, auditLog *models.ComplianceAuditLog) (err error) {
	// Basic validation
	if msg.ID.IsZero() {
		return dberr.ErrMissingID
	}

	// Update modified timestamp (in place).
	msg.Modified = time.Now()

	// Execute the sunrise message into the database
	var result sql.Result
	if result, err = t.tx.Exec(updateSunriseSQL, msg.Params()...); err != nil {
		return dbe(err)
	} else if nRows, _ := result.RowsAffected(); nRows == 0 {
		return dberr.ErrNotFound
	}

	//FIXME: CREATE THE AUDIT LOG
	_ = auditLog

	return nil
}

const updateSunriseStatusSQL = "UPDATE sunrise SET status=:status, modified=:modified WHERE envelope_id=:envelopeID"

func (s *Store) UpdateSunriseStatus(ctx context.Context, txID uuid.UUID, status enum.Status, auditLog *models.ComplianceAuditLog) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.UpdateSunriseStatus(txID, status, auditLog); err != nil {
		return err
	}

	return tx.Commit()
}

func (t *Tx) UpdateSunriseStatus(txID uuid.UUID, status enum.Status, auditLog *models.ComplianceAuditLog) (err error) {
	params := []interface{}{
		sql.Named("status", status),
		sql.Named("modified", time.Now()),
		sql.Named("envelopeID", txID),
	}

	var result sql.Result
	if result, err = t.tx.Exec(updateSunriseStatusSQL, params...); err != nil {
		return dbe(err)
	} else if nRows, _ := result.RowsAffected(); nRows == 0 {
		return dberr.ErrNotFound
	}

	//FIXME: CREATE THE AUDIT LOG
	_ = auditLog

	return nil
}

const deleteSunriseSQL = "DELETE FROM sunrise WHERE id=:id"

// Delete sunrise message from the database.
func (s *Store) DeleteSunrise(ctx context.Context, id ulid.ULID, auditLog *models.ComplianceAuditLog) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.DeleteSunrise(id, auditLog); err != nil {
		return err
	}

	return tx.Commit()
}

// Delete sunrise message from the database.
func (t *Tx) DeleteSunrise(id ulid.ULID, auditLog *models.ComplianceAuditLog) (err error) {
	var result sql.Result
	if result, err = t.tx.Exec(deleteSunriseSQL, sql.Named("id", id)); err != nil {
		return dbe(err)
	} else if nRows, _ := result.RowsAffected(); nRows == 0 {
		return dberr.ErrNotFound
	}

	//FIXME: CREATE THE AUDIT LOG
	_ = auditLog

	return nil
}

const (
	lookupContactEmailSQL     = "SELECT counterparty_id FROM contacts WHERE LOWER(email)=LOWER(:email)"
	countCounterpartyNameSQL  = "SELECT count(id) FROM counterparties WHERE name LIKE :name"
	lookupCounterpartyNameSQL = "SELECT id FROM counterparties WHERE name LIKE :name LIMIT 1"
)

// Get or create a sunrise counterparty from an email address.
func (s *Store) GetOrCreateSunriseCounterparty(ctx context.Context, email, name string, auditLog *models.ComplianceAuditLog) (out *models.Counterparty, err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if out, err = tx.GetOrCreateSunriseCounterparty(email, name, auditLog); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return out, nil
}

func (t *Tx) GetOrCreateSunriseCounterparty(email, name string, auditLog *models.ComplianceAuditLog) (out *models.Counterparty, err error) {
	// Parse the email address to separate the parts if it's RFC spec and put it
	// into a Contact to use later
	var contact *models.Contact
	if contact, err = contactFromEmail(email); err != nil {
		return nil, fmt.Errorf("could not parse the provided email address: %w", err)
	}

	// Attempt the lookup by first looking up a contact with the email address and if
	// that's not found, by looking up the counterparty by name (case insensitive). Any
	// ErrNotFound are ignored, which should cause the method to create the counterparty.
	var counterpartyID ulid.ULID
	if counterpartyID, err = t.lookupContactCounterparty(contact.Email); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			if counterpartyID, err = t.lookupCounterpartyName(name); err != nil {
				if !errors.Is(err, dberr.ErrNotFound) {
					return nil, err
				}
			}
		} else {
			return nil, err
		}
	}

	if counterpartyID.IsZero() {
		// Create the counterparty
		out = &models.Counterparty{
			Source:     enum.SourceUserEntry,
			Protocol:   enum.ProtocolSunrise,
			CommonName: domainFromEmail(contact.Email),
			Endpoint:   fmt.Sprintf("mailto:%s", contact.Email),
			Name:       name,
		}

		if out.CommonName != "" {
			out.Website = sql.NullString{Valid: true, String: "https://" + out.CommonName}
		}

		// Add contact to the counterparty
		out.SetContacts([]*models.Contact{contact})

		//FIXME: COMPLETE AUDIT LOG
		if err = t.CreateCounterparty(out, &models.ComplianceAuditLog{}); err != nil {
			return nil, err
		}
	} else {
		// Retrieve the counterparty that we found earlier
		if out, err = t.RetrieveCounterparty(counterpartyID); err != nil {
			return nil, err
		}

		// Add the email address to the contacts if it didn't already exist
		if exists, _ := out.HasContact(contact.Email); !exists {
			contact.CounterpartyID = out.ID
			//FIXME: COMPLETE AUDIT LOG
			if err = t.CreateContact(contact, &models.ComplianceAuditLog{}); err != nil {
				return nil, err
			}

			// Reset the contacts list with the newly added contact
			if err = t.listCounterpartyContacts(out); err != nil {
				return nil, err
			}
		}
	}

	//FIXME: CREATE THE AUDIT LOG
	_ = auditLog

	return out, nil
}

func (t *Tx) lookupContactCounterparty(email string) (counterpartyID ulid.ULID, err error) {
	if err = t.tx.QueryRow(lookupContactEmailSQL, sql.Named("email", email)).Scan(&counterpartyID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ulid.ULID{}, dberr.ErrNotFound
		}
		return ulid.ULID{}, err
	}
	return counterpartyID, nil
}

func (t *Tx) lookupCounterpartyName(name string) (counterpartyID ulid.ULID, err error) {
	var count int
	if err = t.tx.QueryRow(countCounterpartyNameSQL, sql.Named("name", name)).Scan(&count); err != nil {
		return ulid.ULID{}, err
	}

	if count == 0 {
		return ulid.ULID{}, dberr.ErrNotFound
	}

	if count > 1 {
		return ulid.ULID{}, dberr.ErrAmbiguous
	}

	if err = t.tx.QueryRow(lookupCounterpartyNameSQL, sql.Named("name", name)).Scan(&counterpartyID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ulid.ULID{}, dberr.ErrNotFound
		}
		return ulid.ULID{}, err
	}
	return counterpartyID, nil
}

func domainFromEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) > 1 {
		return parts[1]
	}
	return ""
}

func contactFromEmail(email string) (_ *models.Contact, err error) {
	var addr *mail.Address
	if addr, err = mail.ParseAddress(email); err != nil {
		return nil, err
	}

	return &models.Contact{
		Name:  addr.Name,
		Email: strings.ToLower(addr.Address),
	}, nil
}
