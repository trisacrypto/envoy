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

	"github.com/trisacrypto/envoy/pkg/enum"
	dberr "github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/models"
)

const listSunriseSQL = "SELECT id, envelope_id, expiration, status, sent_on, verified_on FROM sunrise"

// Retrieve sunrise messages from the database and return them as a paginated list.
func (s *Store) ListSunrise(ctx context.Context, page *models.PageInfo) (out *models.SunrisePage, err error) {
	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// TODO: handle pagination
	out = &models.SunrisePage{
		Messages: make([]*models.Sunrise, 0),
		Page:     models.PageInfoFrom(page),
	}

	var rows *sql.Rows
	if rows, err = tx.Query(listSunriseSQL); err != nil {
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

	tx.Commit()
	return out, nil
}

const createSunriseSQL = "INSERT INTO sunrise (id, envelope_id, email, expiration, signature, status, sent_on, verified_on, created, modified) VALUES (:id, :envelopeID, :email, :expiration, :signature, :status, :sentOn, :verifiedOn, :created, :modified)"

// Create a sunrise message in the database.
func (s *Store) CreateSunrise(ctx context.Context, msg *models.Sunrise) (err error) {
	// Basic validation
	// Note: this is duplicated in updateSunrise but better to check before starting a
	// transaction that will take up system resources.
	if !msg.ID.IsZero() {
		return dberr.ErrNoIDOnCreate
	}

	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = createSunrise(tx, msg); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

func createSunrise(tx *sql.Tx, msg *models.Sunrise) (err error) {
	// Basic validation
	if !msg.ID.IsZero() {
		return dberr.ErrNoIDOnCreate
	}

	// Create IDs and model metadata, updating the sunrise message in place.
	msg.ID = ulid.MakeSecure()
	msg.Created = time.Now()
	msg.Modified = msg.Created

	// Execute the insert into the database
	if _, err = tx.Exec(createSunriseSQL, msg.Params()...); err != nil {
		return dbe(err)
	}
	return nil
}

const retrieveSunriseSQL = "SELECT * FROM sunrise WHERE id=:id"

// Retrieve sunrise message detail information.
func (s *Store) RetrieveSunrise(ctx context.Context, id ulid.ULID) (msg *models.Sunrise, err error) {
	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if msg, err = retrieveSunrise(tx, id); err != nil {
		return nil, err
	}

	tx.Commit()
	return msg, nil
}

func retrieveSunrise(tx *sql.Tx, sunriseID ulid.ULID) (msg *models.Sunrise, err error) {
	msg = &models.Sunrise{}
	if err = msg.Scan(tx.QueryRow(retrieveSunriseSQL, sql.Named("id", sunriseID))); err != nil {
		return nil, dbe(err)
	}
	return msg, nil
}

const updateSunriseSQL = "UPDATE sunrise SET envelope_id=:envelopeID, email=:email, expiration=:expiration, signature=:signature, status=:status, sent_on=:sentOn, verified_on=:verifiedOn, modified=:modified WHERE id=:id"

// Update sunrise message information.
func (s *Store) UpdateSunrise(ctx context.Context, msg *models.Sunrise) (err error) {
	// Basic validation
	// Note: this is duplicated in updateSunrise but better to check before starting a
	// transaction that will take up system resources.
	if msg.ID.IsZero() {
		return dberr.ErrMissingID
	}

	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = updateSunrise(tx, msg); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

func updateSunrise(tx *sql.Tx, msg *models.Sunrise) (err error) {
	// Basic validation
	if msg.ID.IsZero() {
		return dberr.ErrMissingID
	}

	// Update modified timestamp (in place).
	msg.Modified = time.Now()

	// Execute the sunrise message into the database
	var result sql.Result
	if result, err = tx.Exec(updateSunriseSQL, msg.Params()...); err != nil {
		return dbe(err)
	} else if nRows, _ := result.RowsAffected(); nRows == 0 {
		return dberr.ErrNotFound
	}

	return nil
}

const deleteSunriseSQL = "DELETE FROM sunrise WHERE id=:id"

// Delete sunrise message from the database.
func (s *Store) DeleteSunrise(ctx context.Context, id ulid.ULID) (err error) {
	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	var result sql.Result
	if result, err = tx.Exec(deleteSunriseSQL, sql.Named("id", id)); err != nil {
		return dbe(err)
	} else if nRows, _ := result.RowsAffected(); nRows == 0 {
		return dberr.ErrNotFound
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

const (
	lookupContactEmailSQL     = "SELECT counterparty_id FROM contacts WHERE email=:email"
	countCounterpartyNameSQL  = "SELECT count(id) FROM counterparties WHERE name LIKE :name"
	lookupCounterpartyNameSQL = "SELECT id FROM counterparties WHERE name LIKE :name LIMIT 1"
)

// Get or create a sunrise counterparty from an email address.
func (s *Store) GetOrCreateSunriseCounterparty(ctx context.Context, email, name string) (out *models.Counterparty, err error) {
	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Attempt the lookup by first looking up a contact with the email address and if
	// that's not found, by looking up the counterparty by name (case insensitive). Any
	// ErrNotFound are ignored, which should cause the method to create the counterparty.
	var counterpartyID ulid.ULID
	if counterpartyID, err = lookupContactCounterparty(tx, email); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			if counterpartyID, err = lookupCounterpartyName(tx, name); err != nil {
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
			CommonName: domainFromEmail(email),
			Endpoint:   fmt.Sprintf("mailto:%s", email),
			Name:       name,
		}

		if out.CommonName != "" {
			out.Website = sql.NullString{Valid: true, String: "https://" + out.CommonName}
		}

		// Add contact to the counterparty
		var contact *models.Contact
		if contact, err = contactFromEmail(email); err != nil {
			return nil, fmt.Errorf("contact is required to create counterparty: %w", err)
		}

		out.SetContacts([]*models.Contact{contact})

		if err = s.createCounterparty(tx, out); err != nil {
			return nil, err
		}
	} else {
		// Retrieve the counterparty that we found earlier
		if out, err = retrieveCounterparty(tx, counterpartyID); err != nil {
			return nil, err
		}

		// Ensure the contacts are set on the counterparty
		if err = s.listContacts(tx, out); err != nil {
			return nil, err
		}

		// Add the email address to the contacts if it didn't already exist
		// If the email isn't passed in or is empty just ignore, assuming the
		// counterparty already has contacts associated with it.
		var contact *models.Contact
		if contact, err = contactFromEmail(email); err == nil {
			if exists, _ := out.HasContact(contact.Email); !exists {
				if err = s.createContact(tx, contact); err != nil {
					return nil, err
				}

				// Reset the contacts list with the newly added contact
				if err = s.listContacts(tx, out); err != nil {
					return nil, err
				}
			}
		}

	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return out, nil
}

func lookupContactCounterparty(tx *sql.Tx, email string) (counterpartyID ulid.ULID, err error) {
	if err = tx.QueryRow(lookupContactEmailSQL, sql.Named("email", email)).Scan(&counterpartyID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ulid.ULID{}, dberr.ErrNotFound
		}
		return ulid.ULID{}, err
	}
	return counterpartyID, nil
}

func lookupCounterpartyName(tx *sql.Tx, name string) (counterpartyID ulid.ULID, err error) {
	var count int
	if err = tx.QueryRow(countCounterpartyNameSQL, sql.Named("name", name)).Scan(&count); err != nil {
		return ulid.ULID{}, err
	}

	if count == 0 {
		return ulid.ULID{}, dberr.ErrNotFound
	}

	if count > 1 {
		return ulid.ULID{}, dberr.ErrAmbiguous
	}

	if err = tx.QueryRow(lookupCounterpartyNameSQL, sql.Named("name", name)).Scan(&counterpartyID); err != nil {
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
