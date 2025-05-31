package mock_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/store/dsn"
	"github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/mock"
)

func TestOpenStore(t *testing.T) {
	store, err := mock.Open(&dsn.DSN{Scheme: dsn.Mock})
	require.NoError(t, err)
	require.NotNil(t, store)
}

func TestCloseStore(t *testing.T) {

	t.Run("Success", func(t *testing.T) {
		// setup
		store, err := mock.Open(&dsn.DSN{Scheme: dsn.Mock})
		require.NoError(t, err)
		require.NotNil(t, store)

		// test
		err = store.Close()
		require.NoError(t, err)
		store.AssertCalls(t, "Close", 1)
	})

	t.Run("CallbackWithError", func(t *testing.T) {
		// setup
		store, err := mock.Open(&dsn.DSN{Scheme: dsn.Mock})
		require.NoError(t, err)
		require.NotNil(t, store)
		store.OnClose(func() error { return errors.ErrInternal }) // the error type doesn't matter

		// test
		err = store.Close()
		require.Error(t, err, "expected an error calling Close()")
		store.AssertCalls(t, "Close", 1)
	})
}

func TestMultipleCallbacksWithReset(t *testing.T) {
	// setup
	store, err := mock.Open(&dsn.DSN{Scheme: dsn.Mock})
	require.NoError(t, err)
	require.NotNil(t, store)

	// test
	err = store.Close()
	require.NoError(t, err, "there was an error calling Close()")
	store.AssertCalls(t, "Close", 1)

	err = store.Close()
	require.NoError(t, err, "there was an error calling Close()")
	store.AssertCalls(t, "Close", 2)

	store.Reset()

	err = store.Close()
	require.NoError(t, err, "there was an error calling Close()")
	store.AssertCalls(t, "Close", 1)
}
