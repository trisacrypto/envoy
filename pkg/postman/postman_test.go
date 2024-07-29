package postman_test

import (
	"os"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

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
