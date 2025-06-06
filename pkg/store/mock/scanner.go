package mock

import (
	"database/sql"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

// MockScanner allows testing `Scan` interfaces for models. You can add an error
// to return or the data items you wish to scan into a model using `SetError()`
// or `SetData()`.
type MockScanner struct {
	err        error
	data       []any
	scanned    int
	notScanned int
}

// Scan will return an error if set, otherwise it will check that the lengths of
// `dest` and the `data` are equal and then attempt to scan each `data` item into
// the `dest` item with the same index, returning any errors.
func (m *MockScanner) Scan(dest ...any) (err error) {
	// if the user set an error, return it
	if m.err != nil {
		return m.err
	}

	// ensure each data item can be scanned into the destination it's assigned to
	for i, dst := range dest {
		if scanner, ok := dst.(sql.Scanner); ok {
			if err = scanner.Scan(m.data[i]); err != nil { // FIXME: pointers to base types aren't recognized here
				return fmt.Errorf("failed to scan data into destination at index %d: %w", i, err)
			}
			m.scanned++
		} else {
			m.notScanned++
			fmt.Printf("item of kind %s at index %d is not a Scanner\n", reflect.TypeOf(dst).Kind().String(), i)
		}

	}

	return nil
}

// Assert that the expected number of `data` items were scanned successfully.
func (m *MockScanner) AssertScanned(t testing.TB, expected int) {
	require.Equal(t, expected, m.scanned, "expected %d scans, got %d", expected, m.scanned)
}

// Assert that the expected number of `data` items were *not* scanned (ignored/nil).
func (m *MockScanner) AssertNotScanned(t testing.TB, expected int) {
	require.Equal(t, expected, m.notScanned, "expected %d non-scans, got %d", expected, m.notScanned)
}

// Sets an error to be returned from the scanner when `Scan()` is called. SetError
// will panic if `data` is already set.
func (m *MockScanner) SetError(err error) {
	// we probably don't want to set both of these at the same
	if m.data != nil {
		panic("data is not nil so data would not be returned")
	}
	m.err = err
}

// Sets data to be scanned into the destinations given when `Scan()` is called.
// SetData will panic if `err` is already set.
func (m *MockScanner) SetData(data []any) {
	// we probably don't want to set both of these at the same
	if m.err != nil {
		panic("err is not nil so data would not be returned")
	}
	m.data = data
}

// Resets the MockScanner to it's original state.
func (m *MockScanner) Reset() {
	m.err = nil
	m.data = nil
	m.scanned = 0
}
