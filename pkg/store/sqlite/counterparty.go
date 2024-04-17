package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	dberr "github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/envoy/pkg/ulids"

	"github.com/oklog/ulid/v2"
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
	if !ulids.IsZero(counterparty.ID) {
		return dberr.ErrNoIDOnCreate
	}

	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	counterparty.ID = ulids.New()
	counterparty.Created = time.Now()
	counterparty.Modified = counterparty.Created

	if _, err = tx.Exec(createCounterpartySQL, counterparty.Params()...); err != nil {
		// TODO: handle constraint violations
		return err
	}

	return tx.Commit()
}

const retreiveCounterpartySQL = "SELECT * FROM counterparties WHERE id=:id"

func (s *Store) RetrieveCounterparty(ctx context.Context, counterpartyID ulid.ULID) (counterparty *models.Counterparty, err error) {
	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	counterparty = &models.Counterparty{}
	if err = counterparty.Scan(tx.QueryRow(retreiveCounterpartySQL, sql.Named("id", counterpartyID))); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, dberr.ErrNotFound
		}
		return nil, err
	}

	tx.Commit()
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
	if ulids.IsZero(counterparty.ID) {
		return dberr.ErrMissingID
	}

	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	counterparty.Modified = time.Now()

	var result sql.Result
	if result, err = tx.Exec(updateCounterpartySQL, counterparty.Params()...); err != nil {
		// TODO: handle constraint violations
		return err
	} else if nRows, _ := result.RowsAffected(); nRows == 0 {
		return dberr.ErrNotFound
	}

	return tx.Commit()
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
