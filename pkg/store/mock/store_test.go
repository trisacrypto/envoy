package mock_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/store/dsn"
	"github.com/trisacrypto/envoy/pkg/store/mock"
)

func TestOpenStore(t *testing.T) {
	store, err := mock.Open(&dsn.DSN{Scheme: dsn.Mock})
	require.NoError(t, err)
	require.NotNil(t, store)
}
