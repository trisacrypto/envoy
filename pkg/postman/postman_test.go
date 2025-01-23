package postman_test

import (
	"fmt"
	"os"

	api "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

//===========================================================================
// Helper Functions
//===========================================================================

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
