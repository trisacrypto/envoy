package ulids_test

import (
	"bytes"
	"encoding/json"
	"testing"

	. "self-hosted-node/pkg/ulids"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestNullULIDScan(t *testing.T) {
	var u ulid.ULID
	var nu NullULID

	uNilErr := u.Scan(nil)
	nuNilErr := nu.Scan(nil)
	require.Equal(t, uNilErr, nuNilErr, "expected errors to be equal, got %s, %s", uNilErr, nuNilErr)

	uInvalidStringErr := u.Scan("test")
	nuInvalidStringErr := nu.Scan("test")
	require.Equal(t, uInvalidStringErr, nuInvalidStringErr, "expected errors to be equal, got %s, %s", uInvalidStringErr, nuInvalidStringErr)

	valid := "01HTNMW2JAW89YSBG7NFPHABA4"
	uValidErr := u.Scan(valid)
	nuValidErr := nu.Scan(valid)
	require.Equal(t, uValidErr, nuValidErr, "expected errors to be equal, got %s, %s", uValidErr, nuValidErr)
}

func TestNullULIDValue(t *testing.T) {
	var u ulid.ULID
	var nu NullULID

	nuValue, nuErr := nu.Value()
	require.NoError(t, nuErr, "expected nil err, got err %s", nuErr)
	require.Nil(t, nuValue, "expected nil value, got non-nil %s", nuValue)

	u = MustParse("01HTNMW2JAW89YSBG7NFPHABA4")
	nu = NullULID{
		ULID:  MustParse("01HTNMW2JAW89YSBG7NFPHABA4"),
		Valid: true,
	}

	uValue, uErr := u.Value()
	nuValue, nuErr = nu.Value()
	require.NoError(t, uErr)
	require.NoError(t, nuErr)
	require.Equal(t, uValue, nuValue, "expected ulid %s and nullulid %s to be equal ", uValue, nuValue)
}

func TestNullULIDMarshalText(t *testing.T) {
	tests := []struct {
		nullULID NullULID
	}{
		{
			nullULID: NullULID{},
		},
		{
			nullULID: NullULID{
				ULID:  MustParse("01HTNMW2JAW89YSBG7NFPHABA4"),
				Valid: true,
			},
		},
	}
	for _, test := range tests {
		var uText []byte
		var uErr error
		nuText, nuErr := test.nullULID.MarshalText()
		if test.nullULID.Valid {
			uText, uErr = test.nullULID.ULID.MarshalText()
		} else {
			uText = []byte("null")
		}

		require.Equal(t, nuErr, uErr, "expected error %e, got %e", nuErr, uErr)
		require.True(t, bytes.Equal(nuText, uText), "expected text data %s, got %s", string(nuText), string(uText))
	}
}

func TestNullULIDMarshalBinary(t *testing.T) {
	tests := []struct {
		nullULID NullULID
	}{
		{
			nullULID: NullULID{},
		},
		{
			nullULID: NullULID{
				ULID:  MustParse("01HTNMW2JAW89YSBG7NFPHABA4"),
				Valid: true,
			},
		},
	}
	for _, test := range tests {
		var uBinary []byte
		var uErr error
		nuBinary, nuErr := test.nullULID.MarshalBinary()
		if test.nullULID.Valid {
			uBinary, uErr = test.nullULID.ULID.MarshalBinary()
		} else {
			uBinary = []byte(nil)
		}

		require.Equal(t, nuErr, uErr, "expected error %e, got %e", nuErr, uErr)
		require.True(t, bytes.Equal(nuBinary, uBinary), "expected binary data %s, got %s", string(nuBinary), string(uBinary))
	}
}

func TestNullULIDMarshalJSON(t *testing.T) {
	jsonNull, _ := json.Marshal(nil)
	tests := []struct {
		nullULID    NullULID
		expected    []byte
		expectedErr error
	}{
		{
			nullULID:    NullULID{},
			expected:    jsonNull,
			expectedErr: nil,
		},
		{
			nullULID: NullULID{
				ULID:  MustParse("01HTNMW2JAW89YSBG7NFPHABA4"),
				Valid: true,
			},
			expected:    []byte(`"01HTNMW2JAW89YSBG7NFPHABA4"`),
			expectedErr: nil,
		},
	}
	for _, test := range tests {
		data, err := json.Marshal(&test.nullULID)
		require.Equal(t, test.expectedErr, err, "expected error %e, got %e", test.expectedErr, err)
		require.True(t, bytes.Equal(data, test.expected), "expected json data %s, got %s", string(test.expected), string(data))
	}
}

func TestNullULIDUnmarshalJSON(t *testing.T) {
	jsonNull, _ := json.Marshal(nil)
	jsonULID, _ := json.Marshal(MustParse("01HTNMW2JAW89YSBG7NFPHABA4"))

	var nu NullULID
	err := json.Unmarshal(jsonNull, &nu)
	require.NoError(t, err)
	require.False(t, nu.Valid)

	err = json.Unmarshal(jsonULID, &nu)
	require.NoError(t, err)
	require.True(t, nu.Valid)
}
