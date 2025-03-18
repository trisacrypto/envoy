package enum_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/enum"
)

func TestValidSources(t *testing.T) {
	tests := []struct {
		input  interface{}
		assert require.BoolAssertionFunc
	}{
		{"", require.True},
		{"unknown", require.True},
		{"gds", require.True},
		{"user", require.True},
		{"peer", require.True},
		{"local", require.True},
		{"remote", require.True},
		{uint8(0), require.True},
		{uint8(1), require.True},
		{uint8(2), require.True},
		{uint8(3), require.True},
		{uint8(4), require.True},
		{uint8(5), require.True},
		{enum.SourceUnknown, require.True},
		{enum.SourceDirectorySync, require.True},
		{enum.SourceUserEntry, require.True},
		{enum.SourcePeer, require.True},
		{enum.SourceLocal, require.True},
		{enum.SourceRemote, require.True},
		{"foo", require.False},
		{true, require.False},
		{uint8(99), require.False},
	}

	for i, tc := range tests {
		tc.assert(t, enum.ValidSource(tc.input), "test case %d failed", i)
	}
}

func TestCheckSource(t *testing.T) {
	tests := []struct {
		input   interface{}
		targets []enum.Source
		assert  require.BoolAssertionFunc
		err     error
	}{
		{"", []enum.Source{enum.SourceUnknown, enum.SourceLocal, enum.SourceRemote}, require.True, nil},
		{"unknown", []enum.Source{enum.SourceLocal, enum.SourceRemote, enum.SourceUnknown}, require.True, nil},
		{"gds", []enum.Source{enum.SourceLocal, enum.SourceRemote}, require.False, nil},
		{"foo", []enum.Source{enum.SourceLocal, enum.SourceRemote}, require.False, errors.New(`invalid source: "foo"`)},
		{"", []enum.Source{enum.SourceLocal, enum.SourceRemote}, require.False, nil},
		{"unknown", []enum.Source{enum.SourceLocal, enum.SourceRemote}, require.False, nil},
		{"gds", []enum.Source{enum.SourceLocal, enum.SourceRemote}, require.False, nil},
	}

	for i, tc := range tests {
		result, err := enum.CheckSource(tc.input, tc.targets...)
		tc.assert(t, result, "test case %d failed", i)

		if tc.err != nil {
			require.Equal(t, tc.err, err, "test case %d failed", i)
		} else {
			require.NoError(t, err, "test case %d failed", i)
		}
	}
}

func TestParseSource(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		tests := []struct {
			input    interface{}
			expected enum.Source
		}{
			{"", enum.SourceUnknown},
			{"unknown", enum.SourceUnknown},
			{"gds", enum.SourceDirectorySync},
			{"user", enum.SourceUserEntry},
			{"peer", enum.SourcePeer},
			{"local", enum.SourceLocal},
			{"remote", enum.SourceRemote},
			{uint8(0), enum.SourceUnknown},
			{uint8(1), enum.SourceDirectorySync},
			{uint8(2), enum.SourceUserEntry},
			{uint8(3), enum.SourcePeer},
			{uint8(4), enum.SourceLocal},
			{uint8(5), enum.SourceRemote},
			{enum.SourceUnknown, enum.SourceUnknown},
			{enum.SourceDirectorySync, enum.SourceDirectorySync},
			{enum.SourceUserEntry, enum.SourceUserEntry},
			{enum.SourcePeer, enum.SourcePeer},
			{enum.SourceLocal, enum.SourceLocal},
			{enum.SourceRemote, enum.SourceRemote},
		}

		for i, test := range tests {
			result, err := enum.ParseSource(test.input)
			require.NoError(t, err, "test case %d failed", i)
			require.Equal(t, test.expected, result, "test case %d failed", i)
		}
	})

	t.Run("Errors", func(t *testing.T) {
		tests := []struct {
			input interface{}
			errs  string
		}{
			{"foo", "invalid source: \"foo\""},
			{true, "cannot parse bool into a source"},
		}

		for i, test := range tests {
			result, err := enum.ParseSource(test.input)
			require.Equal(t, enum.SourceUnknown, result, "test case %d failed", i)
			require.EqualError(t, err, test.errs, "test case %d failed", i)
		}
	})
}

func TestSourceString(t *testing.T) {
	tests := []struct {
		source   enum.Source
		expected string
	}{
		{enum.SourceUnknown, "unknown"},
		{enum.SourceDirectorySync, "gds"},
		{enum.SourceUserEntry, "user"},
		{enum.SourcePeer, "peer"},
		{enum.SourceLocal, "local"},
		{enum.SourceRemote, "remote"},
		{enum.Source(99), "unknown"},
	}

	for i, test := range tests {
		result := test.source.String()
		require.Equal(t, test.expected, result, "test case %d failed", i)
	}
}

func TestSourceJSON(t *testing.T) {
	tests := []enum.Source{
		enum.SourceUnknown, enum.SourceDirectorySync, enum.SourceUserEntry,
		enum.SourcePeer, enum.SourceLocal, enum.SourceRemote,
	}

	for _, source := range tests {
		data, err := json.Marshal(source)
		require.NoError(t, err)

		var result enum.Source
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)
		require.Equal(t, source, result)
	}
}

func TestSourceScan(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected enum.Source
	}{
		{nil, enum.SourceUnknown},
		{"", enum.SourceUnknown},
		{"unknown", enum.SourceUnknown},
		{"gds", enum.SourceDirectorySync},
		{"user", enum.SourceUserEntry},
		{"peer", enum.SourcePeer},
		{"local", enum.SourceLocal},
		{"remote", enum.SourceRemote},
		{[]byte(""), enum.SourceUnknown},
		{[]byte("unknown"), enum.SourceUnknown},
		{[]byte("gds"), enum.SourceDirectorySync},
		{[]byte("user"), enum.SourceUserEntry},
		{[]byte("peer"), enum.SourcePeer},
		{[]byte("local"), enum.SourceLocal},
		{[]byte("remote"), enum.SourceRemote},
	}

	for i, test := range tests {
		var source enum.Source
		err := source.Scan(test.input)
		require.NoError(t, err, "test case %d failed", i)
		require.Equal(t, test.expected, source, "test case %d failed", i)
	}

	var d enum.Source
	err := d.Scan("foo")
	require.EqualError(t, err, "invalid source: \"foo\"")
	err = d.Scan(true)
	require.EqualError(t, err, "cannot scan bool into a source")
}

func TestSourceValue(t *testing.T) {
	value, err := enum.SourceDirectorySync.Value()
	require.NoError(t, err)
	require.Equal(t, "gds", value)
}
