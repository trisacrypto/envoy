package mock_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
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
	TestULID    ulid.ULID
	TestTime    time.Time // FIXME: only the `NullType` version works, not the base type
	TestInt     int       // FIXME: only the `NullType` version works, not the base type
	TestFloat64 float64   // FIXME: only the `NullType` version works, not the base type
	TestString  string    // FIXME: only the `NullType` version works, not the base type
	TestBytes   []byte    // FIXME: this doesn't work

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
		&m.TestInt,
		&m.TestFloat64,
		&m.TestString,
		&m.TestBytes,
	)
}

func TestScanner(t *testing.T) {
	t.Run("SuccessWithAllValues", func(t *testing.T) {
		// setup
		data := []any{
			ulid.MakeSecure().String(), // i = 0  (TestNullULID)
			time.Now(),                 // i = 1  (TestNullTime)
			808,                        // i = 2  (TestNullInt64)
			3.14159,                    // i = 3  (TestNullFloat64)
			"Mahalo",                   // i = 4  (TestNullString)
			ulid.MakeSecure().String(), // i = 5  (TestULID)
			time.Now(),                 // i = 6  (TestTime)
			808,                        // i = 7  (TestInt)
			3.14159,                    // i = 8  (TestFloat64)
			"Mauka",                    // i = 9  (TestString)
			[]byte("Makai"),            // i = 10 (TestBytes)
		}
		mockScanner := &mock.MockScanner{}
		mockScanner.SetData(data)

		// test
		model := &MockTestModel{}
		err := model.Scan(mockScanner)
		require.NoError(t, err, "expected no errors from the scanner")
		mockScanner.AssertScanned(t, len(data))
	})
}
