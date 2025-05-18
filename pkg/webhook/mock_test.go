package webhook_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/webhook"
)

func TestMockURL(t *testing.T) {
	require.Equal(t, "mock", webhook.MockConfig.Endpoint().Scheme)
}

func TestMock(t *testing.T) {
	cb, _ := webhook.New(webhook.MockConfig)
	require.IsType(t, &webhook.Mock{}, cb, "expected mock handler to be returned")

	req, err := loadRequest("transaction_payload.json")
	require.NoError(t, err, "could not load request fixture")

	mock := cb.(*webhook.Mock)
	mock.UseFixture("testdata/reply.json")

	rep, err := cb.Callback(context.Background(), req)
	require.NoError(t, err, "expected no error on the callback response")
	require.Equal(t, "d0a3dcb5-589c-4450-a84d-54ef0b74ae78", rep.TransactionID.String())

	mock.UseError(errors.New("could not connect to mock webhook"))

	rep, err = cb.Callback(context.Background(), req)
	require.EqualError(t, err, "could not connect to mock webhook")
	require.Nil(t, rep)

	require.Equal(t, 2, mock.Callbacks)
	mock.Reset()
	require.Equal(t, 0, mock.Callbacks)
}

func TestMockReply(t *testing.T) {
	cb, _ := webhook.New(webhook.MockConfig)
	require.IsType(t, &webhook.Mock{}, cb, "expected mock handler to be returned")

	mock := cb.(*webhook.Mock)

	req, err := loadRequest("transaction_payload.json")
	require.NoError(t, err, "could not load request fixture")

	t.Run("Pending", func(t *testing.T) {
		mock.OnCallback = webhook.MockPendingReply
		rep, err := cb.Callback(context.Background(), req)
		require.NoError(t, err)

		require.Equal(t, req.TransactionID, rep.TransactionID)
		require.Nil(t, rep.Error)
		require.NotNil(t, rep.Payload)
	})

	t.Run("Error", func(t *testing.T) {
		mock.OnCallback = webhook.MockErrorReply
		rep, err := cb.Callback(context.Background(), req)
		require.NoError(t, err)

		require.Equal(t, req.TransactionID, rep.TransactionID)
		require.NotNil(t, rep.Error)
		require.Nil(t, rep.Payload)
	})
}
