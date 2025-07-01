package enum_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/enum"
)

func TestParseAction(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		tests := []struct {
			input    interface{}
			expected enum.Action
		}{
			{"", enum.ActionUnknown},
			{"unknown", enum.ActionUnknown},
			{"UNKNOWN", enum.ActionUnknown},
			{"create", enum.ActionCreate},
			{"CREATE", enum.ActionCreate},
			{"update", enum.ActionUpdate},
			{"UPDATE", enum.ActionUpdate},
			{"delete", enum.ActionDelete},
			{"DELETE", enum.ActionDelete},
			{uint8(0), enum.ActionUnknown},
			{uint8(1), enum.ActionCreate},
			{uint8(2), enum.ActionUpdate},
			{uint8(3), enum.ActionDelete},
			{enum.ActionUnknown, enum.ActionUnknown},
			{enum.ActionCreate, enum.ActionCreate},
			{enum.ActionUpdate, enum.ActionUpdate},
			{enum.ActionDelete, enum.ActionDelete},
		}

		for i, test := range tests {
			result, err := enum.ParseAction(test.input)
			require.NoError(t, err, "test case %d failed", i)
			require.Equal(t, test.expected, result, "test case %d failed", i)
		}
	})

	t.Run("Errors", func(t *testing.T) {
		tests := []struct {
			input interface{}
			errs  string
		}{
			{"aloha", "invalid action: \"aloha\""},
			{true, "cannot parse bool into an action"},
		}

		for i, test := range tests {
			result, err := enum.ParseAction(test.input)
			require.Equal(t, enum.ActionUnknown, result, "test case %d failed", i)
			require.EqualError(t, err, test.errs, "test case %d failed", i)
		}
	})
}

func TestActionString(t *testing.T) {
	tests := []struct {
		action   enum.Action
		expected string
	}{
		{enum.ActionUnknown, "unknown"},
		{enum.ActionCreate, "create"},
		{enum.ActionUpdate, "update"},
		{enum.ActionDelete, "delete"},
		{enum.Action(4), "unknown"},
		{enum.Action(99), "unknown"},
	}

	for i, test := range tests {
		result := test.action.String()
		require.Equal(t, test.expected, result, "test case %d failed", i)
	}
}

func TestActionJSON(t *testing.T) {
	tests := []enum.Action{
		enum.ActionUnknown,
		enum.ActionCreate,
		enum.ActionUpdate,
		enum.ActionDelete,
	}

	for _, action := range tests {
		data, err := json.Marshal(action)
		require.NoError(t, err)

		var result enum.Action
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)
		require.Equal(t, action, result)
	}
}

func TestActionScan(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected enum.Action
	}{
		{nil, enum.ActionUnknown},
		{"", enum.ActionUnknown},
		{"unknown", enum.ActionUnknown},
		{"UNKNOWN", enum.ActionUnknown},
		{"create", enum.ActionCreate},
		{"CREATE", enum.ActionCreate},
		{"update", enum.ActionUpdate},
		{"UPDATE", enum.ActionUpdate},
		{"delete", enum.ActionDelete},
		{"DELETE", enum.ActionDelete},
		{[]byte(""), enum.ActionUnknown},
		{[]byte("unknown"), enum.ActionUnknown},
		{[]byte("UNKNOWN"), enum.ActionUnknown},
		{[]byte("create"), enum.ActionCreate},
		{[]byte("CREATE"), enum.ActionCreate},
		{[]byte("update"), enum.ActionUpdate},
		{[]byte("UPDATE"), enum.ActionUpdate},
		{[]byte("delete"), enum.ActionDelete},
		{[]byte("DELETE"), enum.ActionDelete},
	}

	for i, test := range tests {
		var action enum.Action
		err := action.Scan(test.input)
		require.NoError(t, err, "test case %d failed", i)
		require.Equal(t, test.expected, action, "test case %d failed", i)
	}

	var d enum.Action
	err := d.Scan("aloha")
	require.EqualError(t, err, "invalid action: \"aloha\"")
	err = d.Scan(true)
	require.EqualError(t, err, "cannot scan bool into an action")
}

func TestActionValue(t *testing.T) {
	value, err := enum.ActionUnknown.Value()
	require.NoError(t, err)
	require.Equal(t, "unknown", value)

	value, err = enum.ActionCreate.Value()
	require.NoError(t, err)
	require.Equal(t, "create", value)

	value, err = enum.ActionUpdate.Value()
	require.NoError(t, err)
	require.Equal(t, "update", value)

	value, err = enum.ActionDelete.Value()
	require.NoError(t, err)
	require.Equal(t, "delete", value)
}
