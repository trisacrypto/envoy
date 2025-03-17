package enum_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/enum"
)

func TestParseDirection(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		tests := []struct {
			input    interface{}
			expected enum.Direction
		}{
			{"", enum.DirectionUnknown},
			{"unknown", enum.DirectionUnknown},
			{"in", enum.DirectionIncoming},
			{"IN", enum.DirectionIncoming},
			{"incoming", enum.DirectionIncoming},
			{"out", enum.DirectionOutgoing},
			{"outgoing", enum.DirectionOutgoing},
			{"OUT", enum.DirectionOutgoing},
			{"any", enum.DirectionAny},
			{uint8(0), enum.DirectionUnknown},
			{uint8(1), enum.DirectionIncoming},
			{uint8(2), enum.DirectionOutgoing},
			{uint8(3), enum.DirectionAny},
			{enum.DirectionUnknown, enum.DirectionUnknown},
			{enum.DirectionIncoming, enum.DirectionIncoming},
			{enum.DirectionOutgoing, enum.DirectionOutgoing},
			{enum.DirectionAny, enum.DirectionAny},
		}

		for i, test := range tests {
			result, err := enum.ParseDirection(test.input)
			require.NoError(t, err, "test case %d failed", i)
			require.Equal(t, test.expected, result, "test case %d failed", i)
		}
	})

	t.Run("Errors", func(t *testing.T) {
		tests := []struct {
			input interface{}
			errs  string
		}{
			{"foo", "invalid direction: \"foo\""},
			{true, "cannot parse bool into a direction"},
		}

		for i, test := range tests {
			result, err := enum.ParseDirection(test.input)
			require.Equal(t, enum.DirectionUnknown, result, "test case %d failed", i)
			require.EqualError(t, err, test.errs, "test case %d failed", i)
		}
	})
}

func TestDirectionString(t *testing.T) {
	tests := []struct {
		direction enum.Direction
		expected  string
	}{
		{enum.DirectionUnknown, "unknown"},
		{enum.DirectionIncoming, "in"},
		{enum.DirectionOutgoing, "out"},
		{enum.DirectionAny, "any"},
		{enum.Direction(99), "unknown"},
	}

	for i, test := range tests {
		result := test.direction.String()
		require.Equal(t, test.expected, result, "test case %d failed", i)
	}
}

func TestDirectionVerbose(t *testing.T) {
	tests := []struct {
		direction enum.Direction
		expected  string
	}{
		{enum.DirectionUnknown, "unknown"},
		{enum.DirectionIncoming, "incoming"},
		{enum.DirectionOutgoing, "outgoing"},
		{enum.DirectionAny, "any"},
		{enum.Direction(99), "unknown"},
	}

	for i, test := range tests {
		result := test.direction.Verbose()
		require.Equal(t, test.expected, result, "test case %d failed", i)
	}
}

func TestDirectionJSON(t *testing.T) {
	tests := []enum.Direction{
		enum.DirectionUnknown,
		enum.DirectionIncoming,
		enum.DirectionOutgoing,
		enum.DirectionAny,
	}

	for _, direction := range tests {
		data, err := json.Marshal(direction)
		require.NoError(t, err)

		var result enum.Direction
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)
		require.Equal(t, direction, result)
	}
}

func TestDirectionScan(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected enum.Direction
	}{
		{nil, enum.DirectionUnknown},
		{"", enum.DirectionUnknown},
		{"unknown", enum.DirectionUnknown},
		{"in", enum.DirectionIncoming},
		{"IN", enum.DirectionIncoming},
		{"incoming", enum.DirectionIncoming},
		{"out", enum.DirectionOutgoing},
		{"outgoing", enum.DirectionOutgoing},
		{"OUT", enum.DirectionOutgoing},
		{"any", enum.DirectionAny},
		{[]byte(""), enum.DirectionUnknown},
		{[]byte("unknown"), enum.DirectionUnknown},
		{[]byte("in"), enum.DirectionIncoming},
		{[]byte("IN"), enum.DirectionIncoming},
		{[]byte("incoming"), enum.DirectionIncoming},
		{[]byte("out"), enum.DirectionOutgoing},
		{[]byte("outgoing"), enum.DirectionOutgoing},
		{[]byte("OUT"), enum.DirectionOutgoing},
		{[]byte("any"), enum.DirectionAny},
	}

	for i, test := range tests {
		var direction enum.Direction
		err := direction.Scan(test.input)
		require.NoError(t, err, "test case %d failed", i)
		require.Equal(t, test.expected, direction, "test case %d failed", i)
	}

	var d enum.Direction
	err := d.Scan("foo")
	require.EqualError(t, err, "invalid direction: \"foo\"")
	err = d.Scan(true)
	require.EqualError(t, err, "cannot scan bool into a direction")
}

func TestDirectionValue(t *testing.T) {
	value, err := enum.DirectionIncoming.Value()
	require.NoError(t, err)
	require.Equal(t, "in", value)
}
