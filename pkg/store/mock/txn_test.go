package mock_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/mock"
)

//==========================================================================
// Helpers
//==========================================================================

// Setup a mock transaction for tests.
func setupTxn(store *mock.Store, opts *sql.TxOptions) *mock.Tx {
	if tx, err := store.Begin(context.Background(), opts); err != nil {
		panic("Error when setting up the mock transaction")
	} else {
		return tx.(*mock.Tx)
	}
}

//==========================================================================
// Tests
//==========================================================================

func TestTxnBegin(t *testing.T) {
	t.Run("Success_StoreRW_TxnRW", func(t *testing.T) {
		// setup
		store := setupStore()

		// test
		ctx := context.Background()
		tx, err := store.Begin(ctx, &sql.TxOptions{ReadOnly: false})
		require.NoError(t, err, "error when calling Begin")
		require.NotNil(t, tx, "tx is nil")
	})

	t.Run("Success_StoreRO_TxnRO", func(t *testing.T) {
		// setup
		store := setupReadOnlyStore()

		// test
		ctx := context.Background()
		tx, err := store.Begin(ctx, &sql.TxOptions{ReadOnly: true})
		require.NoError(t, err, "error when calling Begin")
		require.NotNil(t, tx, "tx is nil")
	})

	t.Run("Success_StoreRW_TxnRO", func(t *testing.T) {
		// setup
		store := setupStore()

		// test
		ctx := context.Background()
		tx, err := store.Begin(ctx, &sql.TxOptions{ReadOnly: true})
		require.NoError(t, err, "error when calling Begin")
		require.NotNil(t, tx, "tx is nil")
	})

	t.Run("Fail_StoreRO_TxnRW", func(t *testing.T) {
		// setup
		store := setupReadOnlyStore()

		// test
		ctx := context.Background()
		tx, err := store.Begin(ctx, &sql.TxOptions{ReadOnly: false})
		require.Error(t, err, "no error when calling Begin")
		require.Equal(t, errors.ErrReadOnly, err, "expected error ErrReadOnly")
		require.Nil(t, tx, "tx is not nil")
	})

}

// NOTE: these tests are pretty much the EXACT SAME as the tests in `prepared_test.go`,
// except these use Txn. They both share the same Commit/Rollback logic and the
// same mock helper functions that we want to test here. As long as we test the
// logic in Commit/Rollback and the helpers, then the rest of this code should
// be fine as it's all just boilerplate.

func TestTxnReset(t *testing.T) {
	// setup
	store := setupStore()
	tx := setupTxn(store, nil)
	tx.OnCommit(func() error { return errors.ErrInternal })

	// test
	// 1: this is to ensure a callback is set and working before reset
	err := tx.Commit()
	require.Error(t, err, "no error when calling Commit")
	require.Equal(t, errors.ErrInternal, err, "expected the error ErrInternal")
	tx.AssertNoCommit(t)
	tx.AssertCalls(t, "Commit", 1)

	// 2: do a rollback before reset
	err = tx.Rollback()
	require.NoError(t, err, "error when calling Rollback")
	tx.AssertRollback(t)
	tx.AssertCalls(t, "Rollback", 1)

	// 3: reset and ensure the commit/rollback and counters get reset
	tx.Reset()
	tx.AssertNoCommit(t)
	tx.AssertCalls(t, "Commit", 0)
	tx.AssertNoRollback(t)
	tx.AssertCalls(t, "Rollback", 0)

	// 4: ensure no error is returned, which means the callbacks were reset
	err = tx.Commit()
	require.NoError(t, err, "error when calling Commit")
	tx.AssertCommit(t)
	tx.AssertCalls(t, "Commit", 1)
}

func TestTxnCommit(t *testing.T) {

	t.Run("Success", func(t *testing.T) {
		// setup
		store := setupStore()
		tx := setupTxn(store, nil)

		// test
		err := tx.Commit()
		require.NoError(t, err, "Error when calling Commit")
		tx.AssertCommit(t)
	})

	t.Run("DoubleCall", func(t *testing.T) {
		// setup
		store := setupStore()
		tx := setupTxn(store, nil)

		// test
		err := tx.Commit()
		require.NoError(t, err, "Error when calling Commit")
		tx.AssertCommit(t)
		err = tx.Commit()
		require.Error(t, err, "No error returned when double-calling Commit")
	})

	t.Run("CalledRollbackFirst", func(t *testing.T) {
		// setup
		store := setupStore()
		tx := setupTxn(store, nil)

		// test
		err := tx.Rollback()
		require.NoError(t, err, "Error when calling Rollback")
		tx.AssertRollback(t)
		err = tx.Commit()
		require.Error(t, err, "No error returned when calling Rollback and then Commit")
		require.Equal(t, sql.ErrTxDone, err, "Expected an ErrTxDone")

	})

	t.Run("WithErrorCallback", func(t *testing.T) {
		// setup
		store := setupStore()
		tx := setupTxn(store, nil)

		// test
		tx.OnCommit(func() error { return errors.ErrInternal })
		err := tx.Commit()
		require.Error(t, err, "No error returned when calling Commit with an error callback")
		require.Equal(t, errors.ErrInternal, err, "Expected an ErrInternal")
	})
}

func TestTxnRollback(t *testing.T) {

	t.Run("Success", func(t *testing.T) {
		// setup
		store := setupStore()
		tx := setupTxn(store, nil)

		// test
		err := tx.Rollback()
		require.NoError(t, err, "Error when calling Rollback")
		tx.AssertRollback(t)
	})

	t.Run("DoubleCall", func(t *testing.T) {
		// setup
		store := setupStore()
		tx := setupTxn(store, nil)

		// test
		err := tx.Rollback()
		require.NoError(t, err, "Error when calling Rollback")
		tx.AssertRollback(t)
		err = tx.Rollback()
		require.Error(t, err, "No error returned when double-calling Rollback")
	})

	t.Run("CalledCommitFirst", func(t *testing.T) {
		// setup
		store := setupStore()
		tx := setupTxn(store, nil)

		// test
		err := tx.Commit()
		require.NoError(t, err, "Error when calling Commit")
		tx.AssertCommit(t)
		err = tx.Rollback()
		require.Error(t, err, "No error returned when calling Commit and then Rollback")
		require.Equal(t, sql.ErrTxDone, err, "Expected an ErrTxDone")

	})

	t.Run("WithErrorCallback", func(t *testing.T) {
		// setup
		store := setupStore()
		tx := setupTxn(store, nil)

		// test
		tx.OnRollback(func() error { return errors.ErrInternal })
		err := tx.Rollback()
		require.Error(t, err, "No error returned when calling Rollback with an error callback")
		require.Equal(t, errors.ErrInternal, err, "Expected an ErrInternal")
	})
}
