package api_test

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	mrand "math/rand"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/web/api/v1"
	"github.com/trisacrypto/trisa/pkg/ivms101"
	"google.golang.org/protobuf/proto"
)

func TestEncodingQuery(t *testing.T) {
	t.Run("Validate", func(t *testing.T) {
		tests := []*api.EncodingQuery{
			{Encoding: "", Format: ""},
			{Encoding: "base64", Format: ""},
			{Encoding: "none", Format: ""},
			{Encoding: "", Format: "json"},
			{Encoding: "", Format: "pb"},
			{Encoding: "none", Format: "json"},
			{Encoding: "base64", Format: "json"},
			{Encoding: "base64", Format: "pb"},
			{Encoding: "NONE", Format: "JSON"},
			{Encoding: "BASE64", Format: "PB"},
		}

		for i, q := range tests {
			require.NoError(t, q.Validate(), "test case %d failed", i)
		}
	})

	t.Run("Invalid", func(t *testing.T) {
		tests := []struct {
			q   *api.EncodingQuery
			err error
		}{
			{
				&api.EncodingQuery{Encoding: "foo", Format: ""},
				api.IncorrectField("encoding", "specify either 'none' or 'base64'"),
			},
			{
				&api.EncodingQuery{Encoding: "", Format: "foo"},
				api.IncorrectField("format", "specify either 'pb' or 'json'"),
			},
			{
				&api.EncodingQuery{Encoding: "none", Format: "pb"},
				api.IncorrectField("format", "when format is 'pb' encoding must be 'base64'"),
			},
		}

		for i, tc := range tests {
			require.EqualError(t, tc.q.Validate(), tc.err.Error(), "test case %d failed", i)
		}
	})
}

func TestEncodingQuerySerialization(t *testing.T) {

	legal := &ivms101.LegalPerson{}
	require.NoError(t, loadFixture("testdata/legal_person.json", legal), "could not load fixture testdata/legal_person.json")

	natural := &ivms101.NaturalPerson{}
	require.NoError(t, loadFixture("testdata/natural_person.json", natural), "could not load fixture testdata/natural_person.json")

	t.Run("PB", func(t *testing.T) {
		testCases := []struct {
			b64 string
			obj proto.Message
		}{
			{
				serializePB(t, legal, base64.StdEncoding),
				&ivms101.LegalPerson{},
			},
			{
				serializePB(t, legal.Person(), base64.StdEncoding),
				&ivms101.Person{},
			},
			{
				serializePB(t, natural, base64.StdEncoding),
				&ivms101.NaturalPerson{},
			},
			{
				serializePB(t, natural.Person(), base64.StdEncoding),
				&ivms101.Person{},
			},
			{
				serializePB(t, legal, base64.URLEncoding),
				&ivms101.LegalPerson{},
			},
			{
				serializePB(t, legal.Person(), base64.URLEncoding),
				&ivms101.Person{},
			},
			{
				serializePB(t, natural, base64.URLEncoding),
				&ivms101.NaturalPerson{},
			},
			{
				serializePB(t, natural.Person(), base64.URLEncoding),
				&ivms101.Person{},
			},
		}

		q := &api.EncodingQuery{Format: "pb", Encoding: "base64"}
		for i, tc := range testCases {
			// Unmarshal data
			err := q.Unmarshal(tc.b64, tc.obj)
			require.NoError(t, err, "test case %d: unmarshal of %q failed", i, tc.b64)

			out, err := q.Marshal(tc.obj)
			require.NoError(t, err, "test case %d: marshal failed", i)
			require.Equal(t, "base64", q.DetectEncoding(out), "test case %d: incorrect detect encoding", i)
			require.Equal(t, tc.b64, out, "test case %d: unexpected output marshaled", i)
		}
	})

	t.Run("JSON", func(t *testing.T) {

		testCases := []struct {
			b64 string
			str string
			obj any
		}{
			{
				serializeJSON(t, legal, base64.StdEncoding),
				serializeJSON(t, legal, nil),
				&ivms101.LegalPerson{},
			},
			{
				serializeJSON(t, legal.Person(), base64.StdEncoding),
				serializeJSON(t, legal.Person(), nil),
				&ivms101.Person{},
			},
			{
				serializeJSON(t, natural, base64.StdEncoding),
				serializeJSON(t, natural, nil),
				&ivms101.NaturalPerson{},
			},
			{
				serializeJSON(t, natural.Person(), base64.StdEncoding),
				serializeJSON(t, natural.Person(), nil),
				&ivms101.Person{},
			},
			{
				serializeJSON(t, legal, base64.URLEncoding),
				serializeJSON(t, legal, nil),
				&ivms101.LegalPerson{},
			},
			{
				serializeJSON(t, legal.Person(), base64.URLEncoding),
				serializeJSON(t, legal.Person(), nil),
				&ivms101.Person{},
			},
			{
				serializeJSON(t, natural, base64.URLEncoding),
				serializeJSON(t, natural, nil),
				&ivms101.NaturalPerson{},
			},
			{
				serializeJSON(t, natural.Person(), base64.URLEncoding),
				serializeJSON(t, natural.Person(), nil),
				&ivms101.Person{},
			},
		}

		t.Run("B64", func(t *testing.T) {

			q := &api.EncodingQuery{Format: "json", Encoding: "base64"}
			for i, tc := range testCases {
				// Unmarshal data
				err := q.Unmarshal(tc.b64, tc.obj)
				require.NoError(t, err, "test case %d: unmarshal failed", i)

				out, err := q.Marshal(tc.obj)
				require.NoError(t, err, "test case %d: marshal failed", i)
				require.Equal(t, "base64", q.DetectEncoding(out), "test case %d: incorrect detect encoding", i)
				require.Equal(t, tc.b64, out, "test case %d: unexpected output marshaled", i)
			}
		})

		t.Run("None", func(t *testing.T) {

			q := &api.EncodingQuery{Format: "json", Encoding: "none"}
			for i, tc := range testCases {
				// Unmarshal data
				err := q.Unmarshal(tc.str, tc.obj)
				require.NoError(t, err, "test case %d: unmarshal failed", i)

				out, err := q.Marshal(tc.obj)
				require.NoError(t, err, "test case %d: marshal failed", i)
				require.Equal(t, "none", q.DetectEncoding(out), "test case %d: incorrect detect encoding", i)
				require.Equal(t, tc.str, out, "test case %d: unexpected output marshaled", i)
			}
		})
	})
}

func TestDetectEncoding(t *testing.T) {
	datagen := func() []byte {
		size := mrand.Intn(65536) + 32
		data := make([]byte, size)
		_, err := rand.Read(data)
		require.NoError(t, err, "could not generate random data")
		return data
	}

	t.Run("Std", func(t *testing.T) {
		for i := 0; i < 64; i++ {
			q := &api.EncodingQuery{}
			data := base64.StdEncoding.EncodeToString(datagen())
			require.Equal(t, "base64", q.DetectEncoding(data))
		}
	})

	t.Run("URL", func(t *testing.T) {
		for i := 0; i < 64; i++ {
			q := &api.EncodingQuery{}
			data := base64.URLEncoding.EncodeToString(datagen())
			require.Equal(t, "base64", q.DetectEncoding(data))
		}
	})

	t.Run("None", func(t *testing.T) {
		tests := []string{
			``,
			`{"fruit": "apple", "age": 42}`,
			`null`,
			`[1, 23, 1, 32, 1, 42, 1, 24, 1]`,
		}

		for i, tc := range tests {
			q := &api.EncodingQuery{}
			require.Equal(t, "none", q.DetectEncoding(tc), "test case %d failed", i)
		}
	})

	t.Run("Unknown", func(t *testing.T) {
		tests := []string{
			`"apple"`,
			`+-/_abcded1341`,
		}

		for i, tc := range tests {
			q := &api.EncodingQuery{}
			require.Equal(t, "unknown", q.DetectEncoding(tc), "test case %d failed", i)
		}
	})
}

type MockEncoder interface {
	EncodeToString(src []byte) string
}

func serializePB(t *testing.T, v proto.Message, encoder MockEncoder) string {
	data, err := proto.Marshal(v)
	require.NoError(t, err, "could not marshal protocol buffer")
	return encoder.EncodeToString(data)
}

func serializeJSON(t *testing.T, v any, encoder MockEncoder) string {
	data, err := json.Marshal(v)
	require.NoError(t, err, "could not marshal json")

	if encoder != nil {
		return encoder.EncodeToString(data)
	}
	return string(data)
}
