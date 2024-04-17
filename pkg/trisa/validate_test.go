package trisa_test

import (
	"fmt"
	"testing"

	"github.com/trisacrypto/envoy/pkg/trisa"

	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/trisa/pkg/ivms101"
	api "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
	generic "github.com/trisacrypto/trisa/pkg/trisa/data/generic/v1beta1"
	"google.golang.org/protobuf/types/known/anypb"
)

func TestValidation(t *testing.T) {
	var (
		identity    *ivms101.IdentityPayload
		transaction *generic.Transaction
		pending     *generic.Pending

		identityPayload    *anypb.Any
		transactionPayload *anypb.Any
		pendingPayload     *anypb.Any
	)

	// Load and setup fixtures for tests.
	identity = &ivms101.IdentityPayload{}
	err := loadFixture("testdata/fixtures/payloads/identity.pb.json", identity)
	require.NoError(t, err, "could not load identity payload")

	transaction = &generic.Transaction{}
	err = loadFixture("testdata/fixtures/payloads/transaction.pb.json", transaction)
	require.NoError(t, err, "could not load transaction payload")

	pending = &generic.Pending{}
	err = loadFixture("testdata/fixtures/payloads/pending.pb.json", pending)
	require.NoError(t, err, "could not load pending payload")

	identityPayload, err = anypb.New(identity)
	require.NoError(t, err, "could not wrap identity payload in anypb")

	transactionPayload, err = anypb.New(transaction)
	require.NoError(t, err, "could not wrap transaction payload in anypb")

	pendingPayload, err = anypb.New(pending)
	require.NoError(t, err, "could not wrap pending payload in anypb")

	t.Run("Valid", func(t *testing.T) {
		t.Parallel()

		testCases := []*api.Payload{
			{
				Identity:    identityPayload,
				Transaction: transactionPayload,
				SentAt:      "2024-03-29T16:18:14Z",
				ReceivedAt:  "",
			},
			{
				Identity:    identityPayload,
				Transaction: pendingPayload,
				SentAt:      "2024-03-29T16:18:14Z",
				ReceivedAt:  "2024-03-29T16:18:16Z",
			},
		}

		for i, payload := range testCases {
			require.Nil(t, trisa.Validate(payload), "expected payload %d to be valid", i)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		t.Parallel()

		// Create an "unparseable fixture"
		unparseable, _ := anypb.New(&api.SecureEnvelope{})

		testCases := []struct {
			payload *api.Payload
			err     *api.Error
		}{
			{
				&api.Payload{
					Identity:    nil,
					Transaction: transactionPayload,
					SentAt:      "2024-03-29T16:18:14Z",
				},
				trisa.ErrMissingIdentity,
			},
			{
				&api.Payload{
					Identity:    unparseable,
					Transaction: transactionPayload,
					SentAt:      "2024-03-29T16:18:14Z",
				},
				&api.Error{
					Code:    api.Error_UNPARSEABLE_IDENTITY,
					Message: fmt.Sprintf("unknown identity payload type %q", unparseable.TypeUrl),
					Retry:   true,
				},
			},
			{
				&api.Payload{
					Identity:    identityPayload,
					Transaction: nil,
					SentAt:      "2024-03-29T16:18:14Z",
				},
				trisa.ErrMissingTransaction,
			},
			{
				&api.Payload{
					Identity:    identityPayload,
					Transaction: unparseable,
					SentAt:      "2024-03-29T16:18:14Z",
				},
				&api.Error{
					Code:    api.Error_UNPARSEABLE_TRANSACTION,
					Message: fmt.Sprintf("unknown transaction payload type %q", unparseable.TypeUrl),
					Retry:   true,
				},
			},
			{
				&api.Payload{
					Identity:    identityPayload,
					Transaction: transactionPayload,
					SentAt:      "",
				},
				trisa.ErrMissingSentAt,
			},
			{
				&api.Payload{
					Identity:    identityPayload,
					Transaction: transactionPayload,
					SentAt:      "invalid",
				},
				trisa.ErrInvalidTimestamp,
			},
			{
				&api.Payload{
					Identity:    identityPayload,
					Transaction: transactionPayload,
					SentAt:      "2024-03-29T16:18:14Z",
					ReceivedAt:  "invalid",
				},
				trisa.ErrInvalidTimestamp,
			},
		}

		for i, tc := range testCases {
			err := trisa.Validate(tc.payload)
			require.Equal(t, tc.err, err, "test case %d failed", i)
		}
	})
}
