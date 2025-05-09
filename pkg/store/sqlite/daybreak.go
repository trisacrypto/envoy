package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"github.com/rs/zerolog/log"
	"github.com/trisacrypto/envoy/pkg/enum"
	dberr "github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/models"
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
						log.Info().Str("directory_id", counterparty.DirectoryID.String).Msg("fixed original counterparty that was not returned in source info (a rare edge case)")
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

	var contacts []*models.Contact
	if contacts, err = counterparty.Contacts(); err != nil {
		return err
	}

	for _, contact := range contacts {
		if contact.CounterpartyID.IsZero() {
			contact.CounterpartyID = counterparty.ID
		}
		if err = s.createContact(tx, contact); err != nil {
			if errors.Is(err, dberr.ErrAlreadyExists) {
				if err = s.updateContact(tx, contact); err != nil {
					log.Warn().Err(err).Str("counterparty", counterparty.ID.String()).Str("contact", contact.Email).Msg("could not update contact")
					return err
				}
			} else {
				log.Warn().Err(err).Str("counterparty", counterparty.ID.String()).Str("contact", contact.Email).Msg("could not create contact")
				return err
			}

		}
	}

	return nil
}
