package mock_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/mock"
	"github.com/trisacrypto/envoy/pkg/store/models"
)

//==========================================================================
// Helpers
//==========================================================================

// Setup a mock PreparedTransaction for tests.
func setupPreparedTransaction(store *mock.Store) *mock.PreparedTransaction {
	//FIXME: COMPLETE AUDIT LOG
	if p, err := store.PrepareTransaction(context.Background(), uuid.UUID{}, &models.ComplianceAuditLog{}); err != nil {
		panic("Error when setting up the mock PreparedTransaction")
	} else {
		return p.(*mock.PreparedTransaction)
	}
}

//==========================================================================
// Tests
//==========================================================================

// NOTE: these tests are pretty much the EXACT SAME as the tests in `txn_test.go`,
// except these use PreparedTransaction. They both share the same Commit/Rollback
// logic and the same mock helper functions that we want to test here. As long as
// we test the logic in Commit/Rollback and the helpers, then the rest of this code
// should be fine as it's all just boilerplate.

func TestPreparedTransactionReset(t *testing.T) {
	// setup
	store := setupStore()
	p := setupPreparedTransaction(store)
	p.OnCommit(func() error { return errors.ErrInternal })

	// test
	// 1: this is to ensure a callback is set and working before reset
	err := p.Commit()
	require.Error(t, err, "no error when calling Commit")
	require.Equal(t, errors.ErrInternal, err, "expected the error ErrInternal")
	p.AssertNoCommit(t)
	p.AssertCalls(t, "Commit", 1)

	// 2: do a rollback before reset
	err = p.Rollback()
	require.NoError(t, err, "error when calling Rollback")
	p.AssertRollback(t)
	p.AssertCalls(t, "Rollback", 1)

	// 3: reset and ensure the commit/rollback and counters get reset
	p.Reset()
	p.AssertNoCommit(t)
	p.AssertCalls(t, "Commit", 0)
	p.AssertNoRollback(t)
	p.AssertCalls(t, "Rollback", 0)

	// 4: ensure no error is returned, which means the callbacks were reset
	err = p.Commit()
	require.NoError(t, err, "error when calling Commit")
	p.AssertCommit(t)
	p.AssertCalls(t, "Commit", 1)
}

func TestPreparedTransactionCommit(t *testing.T) {

	t.Run("Success", func(t *testing.T) {
		// setup
		store := setupStore()
		p := setupPreparedTransaction(store)

		// test
		err := p.Commit()
		require.NoError(t, err, "Error when calling Commit")
		p.AssertCommit(t)
	})

	t.Run("DoubleCall", func(t *testing.T) {
		// setup
		store := setupStore()
		p := setupPreparedTransaction(store)

		// test
		err := p.Commit()
		require.NoError(t, err, "Error when calling Commit")
		p.AssertCommit(t)
		err = p.Commit()
		require.Error(t, err, "No error returned when double-calling Commit")
	})

	t.Run("CalledRollbackFirst", func(t *testing.T) {
		// setup
		store := setupStore()
		p := setupPreparedTransaction(store)

		// test
		err := p.Rollback()
		require.NoError(t, err, "Error when calling Rollback")
		p.AssertRollback(t)
		err = p.Commit()
		require.Error(t, err, "No error returned when calling Rollback and then Commit")
		require.Equal(t, sql.ErrTxDone, err, "Expected an ErrTxDone")

	})

	t.Run("WithErrorCallback", func(t *testing.T) {
		// setup
		store := setupStore()
		p := setupPreparedTransaction(store)

		// test
		p.OnCommit(func() error { return errors.ErrInternal })
		err := p.Commit()
		require.Error(t, err, "No error returned when calling Commit with an error callback")
		require.Equal(t, errors.ErrInternal, err, "Expected an ErrInternal")
	})
}

func TestPreparedTransactionRollback(t *testing.T) {

	t.Run("Success", func(t *testing.T) {
		// setup
		store := setupStore()
		p := setupPreparedTransaction(store)

		// test
		err := p.Rollback()
		require.NoError(t, err, "Error when calling Rollback")
		p.AssertRollback(t)
	})

	t.Run("DoubleCall", func(t *testing.T) {
		// setup
		store := setupStore()
		p := setupPreparedTransaction(store)

		// test
		err := p.Rollback()
		require.NoError(t, err, "Error when calling Rollback")
		p.AssertRollback(t)
		err = p.Rollback()
		require.Error(t, err, "No error returned when double-calling Rollback")
	})

	t.Run("CalledCommitFirst", func(t *testing.T) {
		// setup
		store := setupStore()
		p := setupPreparedTransaction(store)

		// test
		err := p.Commit()
		require.NoError(t, err, "Error when calling Commit")
		p.AssertCommit(t)
		err = p.Rollback()
		require.Error(t, err, "No error returned when calling Commit and then Rollback")
		require.Equal(t, sql.ErrTxDone, err, "Expected an ErrTxDone")

	})

	t.Run("WithErrorCallback", func(t *testing.T) {
		// setup
		store := setupStore()
		p := setupPreparedTransaction(store)

		// test
		p.OnRollback(func() error { return errors.ErrInternal })
		err := p.Rollback()
		require.Error(t, err, "No error returned when calling Rollback with an error callback")
		require.Equal(t, errors.ErrInternal, err, "Expected an ErrInternal")
	})
}
