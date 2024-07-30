package postman_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/postman"
	api "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func TestSend(t *testing.T) {
	// Should be able to create an outgoing envelope for sending a message.
	payload, err := loadPayloadFixture("testdata/identity.pb.json", "testdata/transaction.pb.json")
	require.NoError(t, err, "could not load payload fixture")

	envelopeID := uuid.New()
	transferState := api.TransferStarted
	log := log.With().Str("envelope_id", envelopeID.String()).Logger()

	packet, err := postman.Send(payload, envelopeID, transferState, log)
	require.NoError(t, err, "could not create packet with valid payload and envelope")

	// Ensure packet has been instantiated correctly
	require.NotNil(t, packet.In, "the packet needs to have an instantiated incoming message")
	require.NotNil(t, packet.Out, "the packet needs to have an instantiated outgoing message")
	require.NotNil(t, packet.Out.Envelope, "the packet needs to have an instantiated envelope")
	require.Equal(t, log, packet.Log, "expected the log to be set correctly")
	require.Equal(t, postman.DirectionOutgoing, packet.Request, "on send the request direction should be outgoing")
	require.Equal(t, postman.DirectionIncoming, packet.Reply, "on send the reply direction should be incoming")

	require.Equal(t, envelopeID.String(), packet.Out.Envelope.ID())
	require.Equal(t, transferState, packet.Out.Envelope.TransferState())
}

func TestSendReject(t *testing.T) {
	// Should be able to create an outgoing message for sending a rejection or repair.
	reject := &api.Error{
		Code:    api.BeneficiaryNameUnmatched,
		Message: "no beneficiary with the specified name exists in our system",
		Retry:   false,
	}

	repair := &api.Error{
		Code:    api.MissingFields,
		Message: "the date of birth of the originator is required for our jurisdiction",
		Retry:   true,
	}

	makeSendRejectTest := func(msg *api.Error, expected api.TransferState) func(t *testing.T) {
		return func(t *testing.T) {
			envelopeID := uuid.New()
			log := log.With().Str("envelope_id", envelopeID.String()).Logger()

			packet, err := postman.SendReject(msg, envelopeID, log)
			require.NoError(t, err, "could not create packet with valid rejection and envelope")

			// Ensure packet has been instantiated correctly
			require.NotNil(t, packet.In, "the packet needs to have an instantiated incoming message")
			require.NotNil(t, packet.Out, "the packet needs to have an instantiated outgoing message")
			require.NotNil(t, packet.Out.Envelope, "the packet needs to have an instantiated envelope")
			require.Equal(t, log, packet.Log, "expected the log to be set correctly")
			require.Equal(t, postman.DirectionOutgoing, packet.Request, "on send the request direction should be outgoing")
			require.Equal(t, postman.DirectionIncoming, packet.Reply, "on send the reply direction should be incoming")

			require.Equal(t, envelopeID.String(), packet.Out.Envelope.ID())
			require.Equal(t, expected, packet.Out.Envelope.TransferState())
		}
	}

	t.Run("Reject", makeSendRejectTest(reject, api.TransferRejected))
	t.Run("Repair", makeSendRejectTest(repair, api.TransferRepair))
}

func loadFixture(path string, obj proto.Message) (err error) {
	var data []byte
	if data, err = os.ReadFile(path); err != nil {
		return err
	}

	json := protojson.UnmarshalOptions{
		AllowPartial:   true,
		DiscardUnknown: true,
	}

	return json.Unmarshal(data, obj)
}

func loadPayloadFixture(identityFixture, txnFixture string) (payload *api.Payload, err error) {
	payload = &api.Payload{
		Transaction: &anypb.Any{},
		Identity:    &anypb.Any{},
		SentAt:      "2024-07-28T07:41:42-05:00",
		ReceivedAt:  "2024-07-28T12:22:19-05:00",
	}

	if err = loadFixture(identityFixture, payload.Identity); err != nil {
		return nil, fmt.Errorf("could not load %s identity fixture: %w", identityFixture, err)
	}

	if err = loadFixture(txnFixture, payload.Transaction); err != nil {
		return nil, fmt.Errorf("could not load %s transaction fixture: %w", txnFixture, err)
	}

	return payload, nil
}
