package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/trisacrypto/envoy/pkg/enum"
	dberr "github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"go.rtnl.ai/ulid"
)

// ListDaybreak returns a map of all the daybreak counterparty sources in the database
// in order to match the directory ID to the internal database record ID.
func (s *Store) ListDaybreak(ctx context.Context) (out map[string]*models.CounterpartySourceInfo, err error) {
	var info []*models.CounterpartySourceInfo
	if info, err = s.ListCounterpartySourceInfo(ctx, enum.SourceDaybreak); err != nil {
		return nil, err
	}
	return convertSourceInfoToDaybreak(info), nil
}

// ListDaybreak returns a map of all the daybreak counterparty sources in the database
// in order to match the directory ID to the internal database record ID.
func (tx *Tx) ListDaybreak() (out map[string]*models.CounterpartySourceInfo, err error) {
	var info []*models.CounterpartySourceInfo
	if info, err = tx.ListCounterpartySourceInfo(enum.SourceDaybreak); err != nil {
		return nil, err
	}
	return convertSourceInfoToDaybreak(info), nil
}

// Shared functionality between Store and Tx ListDaybreak methods.
func convertSourceInfoToDaybreak(info []*models.CounterpartySourceInfo) (out map[string]*models.CounterpartySourceInfo) {
	out = make(map[string]*models.CounterpartySourceInfo, len(info))
	for _, src := range info {
		if src.DirectoryID.Valid {
			out[src.DirectoryID.String] = src
		} else {
			log.Warn().Str("id", src.ID.String()).Msg("daybreak counterparty missing directory ID and should be removed from database")
		}
	}
	return out
}

// Create the counterparty and any associated contacts in the database. If the
// counterparty already exists and it is a daybreak record, then the record will try to
// be fixed, otherwise an error will be returned; if a contact already exists associated
// with another counterparty, an error will be returned. The counterparty and all
// associated contacts will be created in a single transaction; if any contact fails to
// be created, the transaction will be rolled back and an error will be returned.
func (s *Store) CreateDaybreak(ctx context.Context, counterparty *models.Counterparty, auditLog *models.ComplianceAuditLog) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.CreateDaybreak(counterparty, auditLog); err != nil {
		return err
	}

	return tx.Commit()
}

func (t *Tx) CreateDaybreak(counterparty *models.Counterparty, auditLog *models.ComplianceAuditLog) (err error) {
	if !counterparty.ID.IsZero() {
		return dberr.ErrNoIDOnCreate
	}

	// Create change notes
	notes := sql.NullString{Valid: true, String: "Store.CreateDaybreak()"}
	if auditLog.ChangeNotes.Valid {
		notes.String = auditLog.ChangeNotes.String + "-" + notes.String
	}

	// Create the counterparty and associated contacts.
	if err = t.CreateCounterparty(counterparty, &models.ComplianceAuditLog{ChangeNotes: notes}); err != nil {
		// If the counterparty is broken, then attempt to fix and try create again.
		if errors.Is(err, dberr.ErrAlreadyExists) {
			if cParty, cerr := t.LookupCounterparty("directory_id", counterparty.DirectoryID.String); cerr == nil {
				// Counterparty was found by the DirectoryID, see if it's supposed to be a "daybreak" based on the RegisteredDirectory
				// If so, it wasn't included in the source map; so the source must have been modified somehow.
				if cParty.RegisteredDirectory.Valid && cParty.RegisteredDirectory.String == "daybreak.rotational.io" {
					counterparty.ID = cParty.ID
					if err = t.UpdateDaybreak(counterparty, auditLog); err != nil {
						log.Warn().Err(err).Str("directory_id", counterparty.DirectoryID.String).Msg("could not fix original daybreak counterparty")
					} else {
						// If no error occurred, then we fixed the original counterparty
						// Need to return here to short-circuit error handling
						log.Debug().Str("directory_id", counterparty.DirectoryID.String).Msg("fixed original counterparty that was not returned in source info (a rare edge case)")
						return nil
					}
				}
			} else {
				log.Warn().Err(cerr).Str("directory_id", counterparty.DirectoryID.String).Msg("could not lookup original counterparty to fix")
			}
		}

		return err
	}

	return nil
}

// Updates the counterparty and any associated contacts in the database. All of the
// contacts will be be created or updated in this transaction. If a contact already
// exists associated with another counterparty, then it will be updated to use this
// counterparty. If any contact fails to be created or updated, the transaction will
// be rolled back and an error will be returned.
func (s *Store) UpdateDaybreak(ctx context.Context, counterparty *models.Counterparty, auditLog *models.ComplianceAuditLog) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.UpdateDaybreak(counterparty, auditLog); err != nil {
		return err
	}

	return tx.Commit()
}

func (t *Tx) UpdateDaybreak(counterparty *models.Counterparty, auditLog *models.ComplianceAuditLog) (err error) {
	if counterparty.ID.IsZero() {
		return dberr.ErrMissingID
	}

	// Create change notes
	notes := sql.NullString{Valid: true, String: "Store.UpdateDaybreak()"}
	if auditLog.ChangeNotes.Valid {
		notes.String = auditLog.ChangeNotes.String + "-" + notes.String
	}

	// Update the counterparty record
	if err = t.UpdateCounterparty(counterparty, &models.ComplianceAuditLog{ChangeNotes: notes}); err != nil {
		return err
	}

	// Retrieve the contacts currently in the DB
	var currContacts map[string]*models.Contact
	if currContacts, err = t.listMapCounterpartyContactsByEmail(counterparty.ID); err != nil {
		return err
	}

	// get the contacts off the incoming counterparty
	var contacts []*models.Contact
	if contacts, err = counterparty.Contacts(); err != nil {
		return err
	}

	// Update or create each contact
	for _, contact := range contacts {
		if contact.CounterpartyID.IsZero() {
			contact.CounterpartyID = counterparty.ID
		}

		if currContact, ok := currContacts[contact.Email]; ok {
			// Contact with this email is present in DB
			contact.ID = currContact.ID
			if err = t.UpdateContact(contact, &models.ComplianceAuditLog{ChangeNotes: notes}); err != nil {
				log.Warn().Err(err).Str("counterparty", counterparty.ID.String()).Str("contact", contact.Email).Msg("could not update contact when updating counterparty")
				return err
			}
		} else {
			// No Contact found with this email for this counterparty
			if !contact.ID.IsZero() {
				contact.ID = ulid.Zero
			}
			if err = t.CreateContact(contact, &models.ComplianceAuditLog{ChangeNotes: notes}); err != nil {
				if err == dberr.ErrAlreadyExists {
					// This email address is proboably associated with a contact for a different counterparty
					log.Warn().Err(err).Str("contact", contact.Email).Msg("contact is associated with two counterparties")
					return err
				} else {
					log.Warn().Err(err).Str("counterparty", counterparty.ID.String()).Str("contact", contact.Email).Msg("could not create contact when updating counterparty")
					return err
				}
			}
		}
	}

	return nil
}

// Deletes the counterparty and any associated contacts in the database, unless
// the counterparty has transactions associated with it. If `ignoreTxns` is `true`,
// then it will delete the counterparty and contacts without checking for associated
// transactions. This function will return the `errors.ErrDaybreakHasTxns` error
// if trying to delete a Daybreak Counterparty with transactions associated to
// it in the database when `ignoreTxns` is not `true`. This function will only
// delete Counterparties with `source='daybreak'`.
func (s *Store) DeleteDaybreak(ctx context.Context, counterpartyID ulid.ULID, ignoreTxns bool, auditLog *models.ComplianceAuditLog) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.DeleteDaybreak(counterpartyID, ignoreTxns, auditLog); err != nil {
		return err
	}

	return tx.Commit()
}

func (t *Tx) DeleteDaybreak(counterpartyID ulid.ULID, ignoreTxns bool, auditLog *models.ComplianceAuditLog) (err error) {
	if ignoreTxns {
		if err = t.deleteDaybreakCounterparty(counterpartyID, auditLog); err != nil {
			log.Warn().Str("counterparty_id", counterpartyID.String()).Msg("error when deleting daybreak counterparty")
			return err
		}
	} else {
		if err = t.deleteDaybreakCounterpartyUnlessHasTxns(counterpartyID, auditLog); err != nil {
			log.Warn().Str("counterparty_id", counterpartyID.String()).Msg("error when deleting daybreak counterparty")
			return err
		}
	}
	return nil
}

const deleteDaybreakCounterpartySQL = "DELETE FROM counterparties WHERE id=:id AND source='daybreak'"

// Delete a Daybreak Counterparty. This function will only delete Counterparties
// with `source='daybreak'`.
func (t *Tx) deleteDaybreakCounterparty(counterpartyID ulid.ULID, auditLog *models.ComplianceAuditLog) (err error) {
	var result sql.Result
	if result, err = t.tx.Exec(deleteDaybreakCounterpartySQL, sql.Named("id", counterpartyID)); err != nil {
		return dbe(err)
	} else if nRows, _ := result.RowsAffected(); nRows == 0 {
		return dberr.ErrNotFound
	}

	// Create ChangeNotes
	notes := sql.NullString{Valid: true, String: "Tx.deleteDaybreakCounterparty()"}
	if auditLog.ChangeNotes.Valid {
		notes.String = auditLog.ChangeNotes.String + "-" + notes.String
	}

	// Fill the audit log and create it
	actorID, actorType := t.GetActor()
	if err := t.CreateComplianceAuditLog(&models.ComplianceAuditLog{
		ActorID:          actorID,
		ActorType:        actorType,
		ResourceID:       counterpartyID.Bytes(),
		ResourceType:     enum.ResourceCounterparty,
		ResourceModified: time.Now(),
		Action:           enum.ActionDelete,
		ChangeNotes:      notes,
	}); err != nil {
		return err
	}

	return nil
}

// Delete a Daybreak Counterparty unless it has transactions associated with it.
// This function will only delete Counterparties with `source='daybreak'`.
func (t *Tx) deleteDaybreakCounterpartyUnlessHasTxns(counterpartyID ulid.ULID, auditLog *models.ComplianceAuditLog) (err error) {
	if has, err := t.counterpartyHasTxns(counterpartyID); has || err != nil {
		if err != nil {
			return err
		}
		return dberr.ErrDaybreakHasTxns

	} else {
		return t.deleteDaybreakCounterparty(counterpartyID, auditLog)
	}
}

//###############
//### Helpers ###
//###############

const counterpartyHasTxnsSQL = "SELECT counterparty_id FROM transactions WHERE counterparty_id=:counterpartyId LIMIT 1"

// Returns `true` if the `counterpartyID` is associated with any transactions, otherwise `false`.
func (t *Tx) counterpartyHasTxns(counterpartyID ulid.ULID) (has bool, err error) {
	// Query
	var rows *sql.Rows
	if rows, err = t.tx.Query(counterpartyHasTxnsSQL, sql.Named("counterpartyId", counterpartyID)); err != nil {
		return true, dbe(err)
	}
	defer rows.Close()

	// `rows.Next()` returns `true` if there is a row, otherwise `false`
	hasTxns := rows.Next()

	return hasTxns, nil
}

// Returns a mapping of Contacts using their email as the keys; useful for comparing
// contacts we need to update/create in bulk imports of counterparties.
func (t *Tx) listMapCounterpartyContactsByEmail(counterpartyID ulid.ULID) (contacts map[string]*models.Contact, err error) {
	var rows *sql.Rows
	if rows, err = t.tx.Query(listContactsSQL, sql.Named("counterpartyID", counterpartyID)); err != nil {
		return nil, dbe(err)
	}
	defer rows.Close()

	contacts = make(map[string]*models.Contact)
	for rows.Next() {
		contact := &models.Contact{}
		if err = contact.Scan(rows); err != nil {
			return nil, err
		}

		contact.CounterpartyID = counterpartyID
		contacts[contact.Email] = contact
	}

	return contacts, nil
}
