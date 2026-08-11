package postman_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/postman"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/envoy/pkg/webhook"
	trisa "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
)

func TestSunriseWebhookRequest(t *testing.T) {
	// Use the same payload shape that a Sunrise recipient submits during review.
	payload, err := loadPayloadFixture("testdata/identity.pb.json", "testdata/transaction.pb.json")
	require.NoError(t, err)

	// An accepted Sunrise review becomes an accepted incoming webhook message
	// containing the identity and transaction payload.
	t.Run("Accept", func(t *testing.T) {
		packet, err := postman.ReceiveSunriseAccept(uuid.New(), payload)
		require.NoError(t, err)

		packet.Counterparty = &models.Counterparty{Name: "Example VASP"}
		request, err := packet.WebhookRequest()
		require.NoError(t, err)

		var received *webhook.Request
		mock := webhook.NewMock()
		mock.OnCallback = func(_ context.Context, request *webhook.Request) (*webhook.Reply, error) {
			received = request
			return &webhook.Reply{TransactionID: request.TransactionID}, nil
		}

		_, err = mock.Callback(context.Background(), request)
		require.NoError(t, err)
		require.Equal(t, 1, mock.Callbacks)
		require.Same(t, request, received)
		require.Equal(t, trisa.TransferAccepted.String(), received.TransferState)
		require.NotNil(t, received.Payload)
		require.NotNil(t, received.Payload.Identity)
		require.NotNil(t, received.Payload.Transaction)
	})

	// A rejected review is delivered as an error-only webhook message, with
	// the transfer state determined by the rejection's retry flag.
	t.Run("Reject", func(t *testing.T) {
		reject := &trisa.Error{
			Code:    trisa.ComplianceCheckFail,
			Message: "transaction rejected",
			Retry:   false,
		}
		packet, err := postman.ReceiveSunriseReject(uuid.New(), reject)
		require.NoError(t, err)

		packet.Counterparty = &models.Counterparty{Name: "Example VASP"}
		request, err := packet.WebhookRequest()
		require.NoError(t, err)

		mock := webhook.NewMock()
		var received *webhook.Request
		mock.OnCallback = func(_ context.Context, request *webhook.Request) (*webhook.Reply, error) {
			received = request
			return &webhook.Reply{TransactionID: request.TransactionID}, nil
		}

		_, err = mock.Callback(context.Background(), request)
		require.NoError(t, err)
		require.Equal(t, 1, mock.Callbacks)
		require.Same(t, request, received)
		require.Equal(t, trisa.TransferRejected.String(), received.TransferState)
		require.Equal(t, reject, received.Error)
		require.Nil(t, received.Payload)
	})
}
