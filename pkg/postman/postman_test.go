package postman_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/postman"
	api "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func TestPacketSend(t *testing.T) {
	// Load the payload fixtures
	payload, err := loadPayloadFixture("testdata/identity.pb.json", "testdata/transaction.pb.json")
	require.NoError(t, err, "could not load payload fixtures")

	// Create a new packet
	envelopeID := uuid.MustParse("b3f7e9a4-6f2d-4b5b-9b4b-7f0b7e9f0e5e")
	packet, err := postman.Send(payload, envelopeID, api.TransferState_REVIEW)
	require.NoError(t, err, "could not create a new packet")

	// Check the packet is not nil
	require.NotNil(t, packet, "the packet should not be nil")
}

func TestPacketReady(t *testing.T) {
	// An empty packet should not be ready
	packet := &postman.Packet{}
	require.Error(t, packet.Ready(), "an empty packet should not be ready")
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
