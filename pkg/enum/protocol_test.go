package enum_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/enum"
)

func TestValidProtocols(t *testing.T) {
	tests := []struct {
		input  interface{}
		assert require.BoolAssertionFunc
	}{
		{"", require.True},
		{"unknown", require.True},
		{"trisa", require.True},
		{"trp", require.True},
		{"sunrise", require.True},
		{uint8(0), require.True},
		{uint8(1), require.True},
		{uint8(2), require.True},
		{uint8(3), require.True},
		{enum.ProtocolUnknown, require.True},
		{enum.ProtocolTRISA, require.True},
		{enum.ProtocolTRP, require.True},
		{enum.ProtocolSunrise, require.True},
		{"foo", require.False},
		{true, require.False},
		{uint8(99), require.False},
	}

	for i, tc := range tests {
		tc.assert(t, enum.ValidProtocol(tc.input), "test case %d failed", i)
	}
}

func TestParseProtocol(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		tests := []struct {
			input    interface{}
			expected enum.Protocol
		}{
			{"", enum.ProtocolUnknown},
			{"unknown", enum.ProtocolUnknown},
			{"trisa", enum.ProtocolTRISA},
			{"trp", enum.ProtocolTRP},
			{"sunrise", enum.ProtocolSunrise},
			{uint8(0), enum.ProtocolUnknown},
			{uint8(1), enum.ProtocolTRISA},
			{uint8(2), enum.ProtocolTRP},
			{uint8(3), enum.ProtocolSunrise},
			{enum.ProtocolUnknown, enum.ProtocolUnknown},
			{enum.ProtocolTRISA, enum.ProtocolTRISA},
			{enum.ProtocolTRP, enum.ProtocolTRP},
			{enum.ProtocolSunrise, enum.ProtocolSunrise},
		}

		for i, test := range tests {
			result, err := enum.ParseProtocol(test.input)
			require.NoError(t, err, "test case %d failed", i)
			require.Equal(t, test.expected, result, "test case %d failed", i)
		}
	})

	t.Run("Errors", func(t *testing.T) {
		tests := []struct {
			input interface{}
			errs  string
		}{
			{"foo", "invalid protocol: \"foo\""},
			{true, "cannot parse bool into a protocol"},
		}

		for i, test := range tests {
			result, err := enum.ParseProtocol(test.input)
			require.Equal(t, enum.ProtocolUnknown, result, "test case %d failed", i)
			require.EqualError(t, err, test.errs, "test case %d failed", i)
		}
	})
}

func TestProtocolString(t *testing.T) {
	tests := []struct {
		protocol enum.Protocol
		expected string
	}{
		{enum.ProtocolUnknown, "unknown"},
		{enum.ProtocolTRISA, "trisa"},
		{enum.ProtocolTRP, "trp"},
		{enum.ProtocolSunrise, "sunrise"},
		{enum.Protocol(99), "unknown"},
	}

	for i, test := range tests {
		result := test.protocol.String()
		require.Equal(t, test.expected, result, "test case %d failed", i)
	}
}

func TestProtocolJSON(t *testing.T) {
	tests := []enum.Protocol{
		enum.ProtocolUnknown, enum.ProtocolTRISA, enum.ProtocolTRP, enum.ProtocolSunrise,
	}

	for _, protocol := range tests {
		data, err := json.Marshal(protocol)
		require.NoError(t, err)

		var result enum.Protocol
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)
		require.Equal(t, protocol, result)
	}
}

func TestProtocolScan(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected enum.Protocol
	}{
		{nil, enum.ProtocolUnknown},
		{"", enum.ProtocolUnknown},
		{"unknown", enum.ProtocolUnknown},
		{"trisa", enum.ProtocolTRISA},
		{"trp", enum.ProtocolTRP},
		{"sunrise", enum.ProtocolSunrise},
		{[]byte(""), enum.ProtocolUnknown},
		{[]byte("unknown"), enum.ProtocolUnknown},
		{[]byte("trisa"), enum.ProtocolTRISA},
		{[]byte("trp"), enum.ProtocolTRP},
		{[]byte("sunrise"), enum.ProtocolSunrise},
	}

	for i, test := range tests {
		var protocol enum.Protocol
		err := protocol.Scan(test.input)
		require.NoError(t, err, "test case %d failed", i)
		require.Equal(t, test.expected, protocol, "test case %d failed", i)
	}

	var d enum.Protocol
	err := d.Scan("foo")
	require.EqualError(t, err, "invalid protocol: \"foo\"")
	err = d.Scan(true)
	require.EqualError(t, err, "cannot scan bool into a protocol")
}

func TestProtocolValue(t *testing.T) {
	value, err := enum.ProtocolTRISA.Value()
	require.NoError(t, err)
	require.Equal(t, "trisa", value)
}
