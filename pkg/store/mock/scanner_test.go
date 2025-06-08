package mock_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/mock"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"go.rtnl.ai/ulid"
)

// A model to test MockScanner with.
type MockTestModel struct {

	// null types
	TestNullULID    ulid.NullULID
	TestNullTime    sql.NullTime
	TestNullInt64   sql.NullInt64
	TestNullFloat64 sql.NullFloat64
	TestNullString  sql.NullString

	// base types
	TestULID         ulid.ULID
	TestTime         time.Time
	TestStringToTime string
	TestInt          int
	TestFloat64      float64
	TestString       string
	TestBytes        []byte

	// "no scan" test (`convertAssign()` should fail for `time.Duration`)
	TestNoScan time.Duration
}

// Scans a MockTestModel.
func (m *MockTestModel) Scan(scanner models.Scanner) error {
	return scanner.Scan(
		&m.TestNullULID,
		&m.TestNullTime,
		&m.TestNullInt64,
		&m.TestNullFloat64,
		&m.TestNullString,
		&m.TestULID,
		&m.TestTime,
		&m.TestStringToTime,
		&m.TestInt,
		&m.TestFloat64,
		&m.TestString,
		&m.TestBytes,
		&m.TestNoScan,
	)
}

func TestScanner(t *testing.T) {
	t.Run("ScanTests", func(t *testing.T) {
		// setup
		data := []any{
			ulid.MakeSecure().String(),         // i = 0  (TestNullULID)
			time.Now(),                         // i = 1  (TestNullTime)
			808,                                // i = 2  (TestNullInt64)
			3.14159,                            // i = 3  (TestNullFloat64)
			"Mahalo",                           // i = 4  (TestNullString)
			ulid.MakeSecure().String(),         // i = 5  (TestULID)
			time.Now(),                         // i = 6  (TestTime)
			"2025-01-01T12:34:56.123456-10:00", // i = 7  (TestStringToTime)
			808,                                // i = 8  (TestInt)
			3.14159,                            // i = 9  (TestFloat64)
			"Mauka",                            // i = 10 (TestString)
			[]byte("Makai"),                    // i = 11 (TestBytes)
			nil,                                // i = 12 (TestNoScan)
		}
		mockScanner := &mock.MockScanner{}
		mockScanner.SetData(data)

		// test
		model := &MockTestModel{}
		err := model.Scan(mockScanner)
		require.NoError(t, err, "expected no errors from the scanner")
		mockScanner.AssertScanned(t, len(data)-1)
		mockScanner.AssertNotScanned(t, 1) // TestNoScan shouldn't scan with `convertAssign()`
	})

	t.Run("SetError", func(t *testing.T) {
		//setup
		mockScanner := &mock.MockScanner{}
		mockScanner.SetError(errors.ErrInternal)

		// test
		model := &MockTestModel{}
		err := model.Scan(mockScanner)
		require.Error(t, err, "expected an error from the scanner")
		require.Equal(t, errors.ErrInternal, err, "expected errors.ErrInternal from the scanner")

	})
}
