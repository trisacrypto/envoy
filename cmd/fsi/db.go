package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/mattn/go-sqlite3"

	"github.com/trisacrypto/envoy/pkg/store/dsn"
)

func openDB(dburl string) (conn *sql.DB, err error) {
	var uri *dsn.DSN
	if uri, err = dsn.Parse(dburl); err != nil {
		return nil, fmt.Errorf("could not parse dsn: %w", err)
	}

	if uri.Scheme != dsn.SQLite && uri.Scheme != dsn.SQLite3 {
		return nil, fmt.Errorf("unhandled database scheme %q", uri.Scheme)
	}

	if uri.Path == "" {
		return nil, fmt.Errorf("path required to open database connection")
	}

	if _, err := os.Stat(uri.Path); os.IsNotExist(err) {
		return nil, fmt.Errorf("database does not exist at %s", uri.Path)
	}

	if conn, err = sql.Open("sqlite3", uri.Path); err != nil {
		return nil, fmt.Errorf("could not open database connection: %w", err)
	}

	return conn, nil
}

func resetDB(tx *sql.Tx, exclude []string) (err error) {
	dnd := make(map[string]struct{})
	for _, table := range exclude {
		dnd[table] = struct{}{}
	}

	var tables []string
	if tables, err = listTables(tx); err != nil {
		return fmt.Errorf("could not list tables: %w", err)
	}

	var nrows, ntables int

	for _, table := range tables {
		if _, ok := dnd[table]; ok {
			continue
		}

		var result sql.Result
		drop := fmt.Sprintf("DELETE FROM %s WHERE 1=1;", table)
		if result, err = tx.Exec(drop); err != nil {
			return fmt.Errorf("could not reset table %s: %w", table, err)
		}

		r, _ := result.RowsAffected()
		nrows += int(r)
		ntables++
	}

	fmt.Printf("Deleted %d rows from %d tables\n", nrows, ntables)
	return nil
}

func listTables(tx *sql.Tx) (tables []string, err error) {
	var rows *sql.Rows
	if rows, err = tx.Query("SELECT name FROM sqlite_master WHERE type='table';"); err != nil {
		return nil, err
	}
	defer rows.Close()

	tables = make([]string, 0)
	for rows.Next() {
		var table string
		if err = rows.Scan(&table); err != nil {
			return nil, err
		}
		tables = append(tables, table)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return tables, nil
}
