package mock_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/store/dsn"
	"github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/mock"
)

//==========================================================================
// Helpers
//==========================================================================

// Setup a mock store for tests.
func setupStore() (store *mock.Store) {
	var err error
	if store, err = mock.Open(&dsn.DSN{Scheme: dsn.Mock}); err != nil {
		panic("Error when setting up the mock store")
	}
	return store
}

//==========================================================================
// Tests
//==========================================================================

func TestStoreOpen(t *testing.T) {
	store, err := mock.Open(&dsn.DSN{Scheme: dsn.Mock})
	require.NoError(t, err, "error when opening store")
	require.NotNil(t, store, "store was nil")
}

func TestStoreClose(t *testing.T) {

	t.Run("Success", func(t *testing.T) {
		// setup
		store := setupStore()

		// test
		err := store.Close()
		require.NoError(t, err, "error when closing store")
		store.AssertCalls(t, "Close", 1)
	})

	t.Run("CallbackWithError", func(t *testing.T) {
		// setup
		store := setupStore()
		store.OnClose(func() error { return errors.ErrInternal }) // the error type doesn't matter

		// test
		err := store.Close()
		require.Error(t, err, "expected an error calling Close")
		store.AssertCalls(t, "Close", 1)
	})
}

func TestStoreMultipleCallbacksWithReset(t *testing.T) {
	// setup
	store := setupStore()

	// test
	err := store.Close()
	require.NoError(t, err, "there was an error calling Close")
	store.AssertCalls(t, "Close", 1)

	err = store.Close()
	require.NoError(t, err, "there was an error calling Close")
	store.AssertCalls(t, "Close", 2)

	store.Reset()

	err = store.Close()
	require.NoError(t, err, "there was an error calling Close")
	store.AssertCalls(t, "Close", 1)
}
