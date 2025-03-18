package postman_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/enum"
	"github.com/trisacrypto/envoy/pkg/postman"
	api "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
)

func TestSendTRISA(t *testing.T) {
	// Should be able to create an outgoing envelope for sending a message.
	payload, err := loadPayloadFixture("testdata/identity.pb.json", "testdata/transaction.pb.json")
	require.NoError(t, err, "could not load payload fixture")

	envelopeID := uuid.New()
	log := log.With().Str("envelope_id", envelopeID.String()).Logger()
	transferState := api.TransferStarted

	packet, err := postman.SendTRISA(envelopeID, payload, transferState)
	require.NoError(t, err, "could not create packet with valid payload and envelope")

	packet.Log = log

	// Ensure packet has been instantiated correctly
	require.NotNil(t, packet.In, "the packet needs to have an instantiated incoming message")
	require.NotNil(t, packet.Out, "the packet needs to have an instantiated outgoing message")
	require.NotNil(t, packet.Out.Envelope, "the packet needs to have an instantiated envelope")
	require.Equal(t, log, packet.Log, "expected the log to be set correctly")
	require.Equal(t, enum.DirectionOutgoing, packet.Request(), "on send the request direction should be outgoing")
	require.Equal(t, enum.DirectionIncoming, packet.Reply(), "on send the reply direction should be incoming")

	require.Equal(t, envelopeID.String(), packet.Out.Envelope.ID())
	require.Equal(t, transferState, packet.Out.Envelope.TransferState())
}

func TestSendTRISAReject(t *testing.T) {
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

			packet, err := postman.SendTRISAReject(envelopeID, msg)
			require.NoError(t, err, "could not create packet with valid rejection and envelope")

			packet.Log = log

			// Ensure packet has been instantiated correctly
			require.NotNil(t, packet.In, "the packet needs to have an instantiated incoming message")
			require.NotNil(t, packet.Out, "the packet needs to have an instantiated outgoing message")
			require.NotNil(t, packet.Out.Envelope, "the packet needs to have an instantiated envelope")
			require.Equal(t, log, packet.Log, "expected the log to be set correctly")
			require.Equal(t, enum.DirectionOutgoing, packet.Request(), "on send the request direction should be outgoing")
			require.Equal(t, enum.DirectionIncoming, packet.Reply(), "on send the reply direction should be incoming")

			require.Equal(t, envelopeID.String(), packet.Out.Envelope.ID())
			require.Equal(t, expected, packet.Out.Envelope.TransferState())
		}
	}

	t.Run("Reject", makeSendRejectTest(reject, api.TransferRejected))
	t.Run("Repair", makeSendRejectTest(repair, api.TransferRepair))
}
