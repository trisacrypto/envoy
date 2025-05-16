package sqlite

import (
	"context"
	"database/sql"
	"errors"

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

	out = make(map[string]*models.CounterpartySourceInfo, len(info))
	for _, src := range info {
		if src.DirectoryID.Valid {
			out[src.DirectoryID.String] = src
		} else {
			log.Warn().Str("id", src.ID.String()).Msg("daybreak counterparty missing directory ID and should be removed from database")
		}
	}

	return out, nil
}

// Create the counterparty and any associated contacts in the database. If the
// counterparty already exists and it is a daybreak record, then the record will try to
// be fixed, otherwise an error will be returned; if a contact already exists associated
// with another counterparty, an error will be returned. The counterparty and all
// associated contacts will be created in a single transaction; if any contact fails to
// be created, the transaction will be rolled back and an error will be returned.
func (s *Store) CreateDaybreak(ctx context.Context, counterparty *models.Counterparty) (err error) {
	if !counterparty.ID.IsZero() {
		return dberr.ErrNoIDOnCreate
	}

	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	// Create the counterparty and associated contacts.
	if err = s.createCounterparty(tx, counterparty); err != nil {
		// If the counterparty is broken, then attempt to fix and try create again.
		if errors.Is(err, dberr.ErrAlreadyExists) {
			if cParty, cerr := lookupCounterparty(tx, "directory_id", counterparty.DirectoryID.String); cerr == nil {
				// Counterparty was found by the DirectoryID, see if it's supposed to be a "daybreak" based on the RegisteredDirectory
				// If so, it wasn't included in the source map; so the source must have been modified somehow.
				if cParty.RegisteredDirectory.Valid && cParty.RegisteredDirectory.String == "daybreak.rotational.io" {
					counterparty.ID = cParty.ID
					if err = s.updateDaybreak(tx, counterparty); err != nil {
						log.Warn().Err(err).Str("directory_id", counterparty.DirectoryID.String).Msg("could not fix original daybreak counterparty")
					} else {
						// If no error occurred, then we fixed the original counterparty
						// Need to commit and return here to short-circuit error handling
						log.Debug().Str("directory_id", counterparty.DirectoryID.String).Msg("fixed original counterparty that was not returned in source info (a rare edge case)")
						return tx.Commit()
					}
				}
			} else {
				log.Warn().Err(cerr).Str("directory_id", counterparty.DirectoryID.String).Msg("could not lookup original counterparty to fix")
			}
		}

		return err
	}

	// Commit if we successfully created the counterparty and all contacts.
	return tx.Commit()
}

// Updates the counterparty and any associated contacts in the database. All of the
// contacts will be be created or updated in this transaction. If a contact already
// exists associated with another counterparty, then it will be updated to use this
// counterparty. If any contact fails to be created or updated, the transaction will
// be rolled back and an error will be returned.
func (s *Store) UpdateDaybreak(ctx context.Context, counterparty *models.Counterparty) (err error) {
	if counterparty.ID.IsZero() {
		return dberr.ErrMissingID
	}

	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = s.updateDaybreak(tx, counterparty); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Store) updateDaybreak(tx *sql.Tx, counterparty *models.Counterparty) (err error) {
	// Update the counterparty record
	if err = updateCounterparty(tx, counterparty); err != nil {
		return err
	}

	// Retrieve the contacts currently in the DB
	var currContacts map[string]*models.Contact
	if currContacts, err = s.listMapCounterpartyContactsByEmail(tx, counterparty.ID); err != nil {
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
			if err = s.updateContact(tx, contact); err != nil {
				log.Warn().Err(err).Str("counterparty", counterparty.ID.String()).Str("contact", contact.Email).Msg("could not update contact when updating counterparty")
				return err
			}
		} else {
			// No Contact found with this email for this counterparty
			if !contact.ID.IsZero() {
				contact.ID = ulid.Zero
			}
			if err = s.createContact(tx, contact); err != nil {
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
// it in the database when `ignoreTxns` is not `true`.
func (s *Store) DeleteDaybreak(ctx context.Context, counterpartyID ulid.ULID, ignoreTxns bool) (err error) {
	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if ignoreTxns {
		if err = s.deleteDaybreakCounterparty(tx, counterpartyID); err != nil {
			log.Warn().Str("counterparty_id", counterpartyID.String()).Msg("error when deleting daybreak counterparty")
			return err
		}
	} else {
		if err = s.deleteDaybreakCounterpartyUnlessHasTxns(tx, counterpartyID); err != nil {
			log.Warn().Str("counterparty_id", counterpartyID.String()).Msg("error when deleting daybreak counterparty")
			return err
		}
	}

	return tx.Commit()
}

func (s *Store) deleteDaybreakCounterparty(tx *sql.Tx, counterpartyID ulid.ULID) (err error) {
	return s.deleteCounterparty(tx, counterpartyID)
}

func (s *Store) deleteDaybreakCounterpartyUnlessHasTxns(tx *sql.Tx, counterpartyID ulid.ULID) (err error) {
	if has, err := s.counterpartyHasTxns(tx, counterpartyID); has || err != nil {
		if err != nil {
			return err
		}
		return dberr.ErrDaybreakHasTxns

	} else {
		return s.deleteDaybreakCounterparty(tx, counterpartyID)
	}
}

//###############
//### Helpers ###
//###############

const counterpartyHasTxnsSQL = "SELECT counterparty_id FROM transactions WHERE counterparty_id=:counterpartyId LIMIT 1"

// Returns `true` if the `counterpartyID` is associated with any transactions, otherwise `false`.
func (s *Store) counterpartyHasTxns(tx *sql.Tx, counterpartyID ulid.ULID) (has bool, err error) {
	// Query
	var rows *sql.Rows
	if rows, err = tx.Query(counterpartyHasTxnsSQL, sql.Named("counterpartyId", counterpartyID)); err != nil {
		return true, dbe(err)
	}
	defer rows.Close()

	// `rows.Next()` returns `true` if there is a row, otherwise `false`
	hasTxns := rows.Next()

	return hasTxns, nil
}

// Returns a mapping of Contacts using their email as the keys; useful for comparing
// contacts we need to update/create in bulk imports of counterparties.
func (s *Store) listMapCounterpartyContactsByEmail(tx *sql.Tx, counterpartyID ulid.ULID) (contacts map[string]*models.Contact, err error) {
	var rows *sql.Rows
	if rows, err = tx.Query(listContactsSQL, sql.Named("counterpartyID", counterpartyID)); err != nil {
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
