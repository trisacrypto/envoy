package mock

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var (
	ErrNilPtr      = errors.New("destination pointer is nil")
	ErrUnknownType = errors.New("unknown type for conversion: reflection required")
)

// MockScanner allows testing `Scan` interfaces for models. You can add an error
// to return or the data items you wish to scan into a model using `SetError()`
// or `SetData()`.
type MockScanner struct {
	err        error
	data       []any
	scanned    int
	notScanned int
	testLog    []string
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
		src := m.data[i]

		// If the type is a scanner type then scan it directly.
		if scanner, ok := dst.(sql.Scanner); ok {
			if err = scanner.Scan(src); err != nil {
				return fmt.Errorf("failed to scan data into destination at index %d: %w", i, err)
			}
			m.scanned++
			continue
		}

		// Check the common cases without reflection.
		if err = convertAssign(dst, src); err != nil {
			if errors.Is(err, ErrUnknownType) {
				m.notScanned++
				m.logf("unsupported Scan, storing drive.Value type %T into type %T at index %d", src, dst, i)
				continue
			}
			return err
		} else {
			m.scanned++
		}
	}

	return nil
}

// Assigns `src` to `dst`, doing some conversions if necessary.
func convertAssign(dst, src any) error {
	switch s := src.(type) {
	case string:
		switch d := dst.(type) {
		case *string: // string into string
			if d == nil {
				return ErrNilPtr
			}
			*d = s
			return nil
		case *[]byte: // string into byte
			if d == nil {
				return ErrNilPtr
			}
			*d = []byte(s)
			return nil
		case *time.Time: // string into time
			t, err := time.Parse("", s)
			if err != nil {
				return err
			}
			*d = t
			return nil
		}
	case []byte:
		switch d := dst.(type) {
		case *string: // bytes into string
			if d == nil {
				return ErrNilPtr
			}
			*d = string(s)
			return nil
		case *any: // bytes into any
			if d == nil {
				return ErrNilPtr
			}
			*d = bytes.Clone(s)
			return nil
		case *[]byte: // bytes into bytes
			if d == nil {
				return ErrNilPtr
			}
			*d = bytes.Clone(s)
			return nil
		}
	case time.Time:
		switch d := dst.(type) {
		case *time.Time: // time into time
			*d = s
			return nil
		case *string: // time into string
			if d == nil {
				return ErrNilPtr
			}
			*d = s.Format(time.RFC3339Nano)
			return nil
		}
	case int:
		switch d := dst.(type) {
		case *int: // int into int
			if d == nil {
				return ErrNilPtr
			}
			*d = s
			return nil
		case *int64: // int into int64
			if d == nil {
				return ErrNilPtr
			}
			*d = int64(s)
			return nil
		}
	case float64:
		switch d := dst.(type) {
		case *float64: // float64 into float64
			if d == nil {
				return ErrNilPtr
			}
			*d = s
			return nil
		}
	case nil:
		switch d := dst.(type) {
		case *any: // nil into any
			if d == nil {
				return ErrNilPtr
			}
			*d = nil
			return nil
		case *[]byte: // nil into bytes
			if d == nil {
				return ErrNilPtr
			}
			*d = nil
			return nil
		}
	}

	// Could not convert: the type src or dst unrecognized.
	return ErrUnknownType
}

// Assert that the expected number of `data` items were scanned successfully.
func (m *MockScanner) AssertScanned(t testing.TB, expected int) {
	m.Log(t)
	require.Equal(t, expected, m.scanned, "expected %d scans, got %d", expected, m.scanned)
}

// Assert that the expected number of `data` items were *not* scanned (ignored/nil).
func (m *MockScanner) AssertNotScanned(t testing.TB, expected int) {
	m.Log(t)
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
	m.notScanned = 0
	m.testLog = nil
}

// Logs any test log messages that were recorded during the test; then clears the log.
func (m *MockScanner) Log(t testing.TB) {
	for _, log := range m.testLog {
		t.Log(log)
	}
	m.testLog = nil
}

// Internal method to append a log message to the test log.
func (m *MockScanner) logf(format string, args ...any) {
	m.testLog = append(m.testLog, fmt.Sprintf(format, args...))
}
