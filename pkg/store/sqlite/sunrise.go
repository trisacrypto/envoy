package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/oklog/ulid/v2"

	dberr "github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/envoy/pkg/ulids"
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
		// TODO: handle database specific errors
		return nil, err
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
	if !ulids.IsZero(msg.ID) {
		return dberr.ErrNoIDOnCreate
	}

	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	// Create IDs and model metadata, updating the sunrise message in place.
	msg.ID = ulids.New()
	msg.Created = time.Now()
	msg.Modified = msg.Created

	// Execute the insert into the database
	if _, err = tx.Exec(createSunriseSQL, msg.Params()...); err != nil {
		// TODO: handle constraint violations
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
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
		if errors.Is(err, sql.ErrNoRows) {
			return nil, dberr.ErrNotFound
		}
		return nil, err
	}
	return msg, nil
}

const updateSunriseSQL = "UPDATE sunrise SET envelope_id=:envelopeID, email=:email, expiration=:expiration, signature=:signature, status=:status, sent_on=:sentOn, verified_on=:verifiedOn, modified=:modified WHERE id=:id"

// Update sunrise message information.
func (s *Store) UpdateSunrise(ctx context.Context, msg *models.Sunrise) (err error) {
	// Basic validation
	if ulids.IsZero(msg.ID) {
		return dberr.ErrMissingID
	}

	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	// Update modified timestamp (in place).
	msg.Modified = time.Now()

	// Execute the sunrise message into the database
	var result sql.Result
	if result, err = tx.Exec(updateSunriseSQL, msg.Params()...); err != nil {
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

const deleteSunriseSQL = "DELETE FROM sunrise WHERE id=:id"

// Delete sunrise message from the database
func (s *Store) DeleteSunrise(ctx context.Context, id ulid.ULID) (err error) {
	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	var result sql.Result
	if result, err = tx.Exec(deleteSunriseSQL, sql.Named("id", id)); err != nil {
		return err
	} else if nRows, _ := result.RowsAffected(); nRows == 0 {
		return dberr.ErrNotFound
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}
