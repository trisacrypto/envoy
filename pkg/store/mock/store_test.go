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

// Setup a read-only mock store for tests.
func setupReadOnlyStore() (store *mock.Store) {
	var err error
	if store, err = mock.Open(&dsn.DSN{ReadOnly: true, Scheme: dsn.Mock}); err != nil {
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
		store.OnClose = func() error { return errors.ErrInternal }

		// test
		err := store.Close()
		require.Error(t, err, "expected an error calling Close")
		store.AssertCalls(t, "Close", 1)
	})
}

func TestStoreMultipleCallbacksWithReset(t *testing.T) {
	// setup
	store := setupStore()

	// test call 1
	err := store.Close()
	require.NoError(t, err, "there was an error calling Close")
	store.AssertCalls(t, "Close", 1)

	// test call 2
	err = store.Close()
	require.NoError(t, err, "there was an error calling Close")
	store.AssertCalls(t, "Close", 2)

	// set a callback and test call 3
	store.OnClose = func() error { return errors.ErrInternal }
	require.NotNil(t, store.OnClose, "OnClose is not supposed to be nil")
	err = store.Close()
	require.Error(t, err, "expected an error calling Close")
	store.AssertCalls(t, "Close", 3)

	// reset
	store.Reset()

	// ensure callback is gone now
	require.Nil(t, store.OnClose, "OnClose is supposed to be nil")

	// call once again to ensure the counter was reset
	err = store.Close()
	require.NoError(t, err, "there was an error calling Close")
	store.AssertCalls(t, "Close", 1)
}
