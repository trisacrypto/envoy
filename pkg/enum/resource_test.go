package enum_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/enum"
)

func TestParseResource(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		tests := []struct {
			input    interface{}
			expected enum.Resource
		}{
			{"", enum.ResourceUnknown},
			{"unknown", enum.ResourceUnknown},
			{"UNKNOWN", enum.ResourceUnknown},
			{"transaction", enum.ResourceTransaction},
			{"TRANSACTION", enum.ResourceTransaction},
			{"user", enum.ResourceUser},
			{"USER", enum.ResourceUser},
			{"api_key", enum.ResourceAPIKey},
			{"API_KEY", enum.ResourceAPIKey},
			{"counterparty", enum.ResourceCounterparty},
			{"COUNTERPARTY", enum.ResourceCounterparty},
			{"account", enum.ResourceAccount},
			{"ACCOUNT", enum.ResourceAccount},
			{"sunrise", enum.ResourceSunrise},
			{"SUNRISE", enum.ResourceSunrise},
			{uint8(0), enum.ResourceUnknown},
			{uint8(1), enum.ResourceTransaction},
			{uint8(2), enum.ResourceUser},
			{uint8(3), enum.ResourceAPIKey},
			{uint8(4), enum.ResourceCounterparty},
			{uint8(5), enum.ResourceAccount},
			{uint8(6), enum.ResourceSunrise},
			{enum.ResourceUnknown, enum.ResourceUnknown},
			{enum.ResourceTransaction, enum.ResourceTransaction},
			{enum.ResourceUser, enum.ResourceUser},
			{enum.ResourceAPIKey, enum.ResourceAPIKey},
			{enum.ResourceCounterparty, enum.ResourceCounterparty},
			{enum.ResourceAccount, enum.ResourceAccount},
			{enum.ResourceSunrise, enum.ResourceSunrise},
		}

		for i, test := range tests {
			result, err := enum.ParseResource(test.input)
			require.NoError(t, err, "test case %d failed", i)
			require.Equal(t, test.expected, result, "test case %d failed", i)
		}
	})

	t.Run("Errors", func(t *testing.T) {
		tests := []struct {
			input interface{}
			errs  string
		}{
			{"aloha", "invalid resource: \"aloha\""},
			{true, "cannot parse bool into a resource"},
		}

		for i, test := range tests {
			result, err := enum.ParseResource(test.input)
			require.Equal(t, enum.ResourceUnknown, result, "test case %d failed", i)
			require.EqualError(t, err, test.errs, "test case %d failed", i)
		}
	})
}

func TestResourceString(t *testing.T) {
	tests := []struct {
		resource enum.Resource
		expected string
	}{
		{enum.ResourceUnknown, "unknown"},
		{enum.ResourceTransaction, "transaction"},
		{enum.ResourceUser, "user"},
		{enum.ResourceAPIKey, "api_key"},
		{enum.ResourceCounterparty, "counterparty"},
		{enum.ResourceAccount, "account"},
		{enum.ResourceSunrise, "sunrise"},
		{enum.Resource(7), "unknown"},
		{enum.Resource(99), "unknown"},
	}

	for i, test := range tests {
		result := test.resource.String()
		require.Equal(t, test.expected, result, "test case %d failed", i)
	}
}

func TestResourceJSON(t *testing.T) {
	tests := []enum.Resource{
		enum.ResourceUnknown,
		enum.ResourceTransaction,
		enum.ResourceUser,
		enum.ResourceAPIKey,
		enum.ResourceCounterparty,
		enum.ResourceAccount,
		enum.ResourceSunrise,
	}

	for _, resource := range tests {
		data, err := json.Marshal(resource)
		require.NoError(t, err)

		var result enum.Resource
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)
		require.Equal(t, resource, result)
	}
}

func TestResourceScan(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected enum.Resource
	}{
		{nil, enum.ResourceUnknown},
		{"", enum.ResourceUnknown},
		{"unknown", enum.ResourceUnknown},
		{"UNKNOWN", enum.ResourceUnknown},
		{"transaction", enum.ResourceTransaction},
		{"TRANSACTION", enum.ResourceTransaction},
		{"user", enum.ResourceUser},
		{"USER", enum.ResourceUser},
		{"api_key", enum.ResourceAPIKey},
		{"API_KEY", enum.ResourceAPIKey},
		{"counterparty", enum.ResourceCounterparty},
		{"COUNTERPARTY", enum.ResourceCounterparty},
		{"account", enum.ResourceAccount},
		{"ACCOUNT", enum.ResourceAccount},
		{"sunrise", enum.ResourceSunrise},
		{"SUNRISE", enum.ResourceSunrise},
		{[]byte(""), enum.ResourceUnknown},
		{[]byte("unknown"), enum.ResourceUnknown},
		{[]byte("UNKNOWN"), enum.ResourceUnknown},
		{[]byte("transaction"), enum.ResourceTransaction},
		{[]byte("TRANSACTION"), enum.ResourceTransaction},
		{[]byte("user"), enum.ResourceUser},
		{[]byte("USER"), enum.ResourceUser},
		{[]byte("api_key"), enum.ResourceAPIKey},
		{[]byte("API_KEY"), enum.ResourceAPIKey},
		{[]byte("counterparty"), enum.ResourceCounterparty},
		{[]byte("COUNTERPARTY"), enum.ResourceCounterparty},
		{[]byte("account"), enum.ResourceAccount},
		{[]byte("ACCOUNT"), enum.ResourceAccount},
		{[]byte("sunrise"), enum.ResourceSunrise},
		{[]byte("SUNRISE"), enum.ResourceSunrise},
	}

	for i, test := range tests {
		var resource enum.Resource
		err := resource.Scan(test.input)
		require.NoError(t, err, "test case %d failed", i)
		require.Equal(t, test.expected, resource, "test case %d failed", i)
	}

	var d enum.Resource
	err := d.Scan("aloha")
	require.EqualError(t, err, "invalid resource: \"aloha\"")
	err = d.Scan(true)
	require.EqualError(t, err, "cannot scan bool into a resource")
}

func TestResourceValue(t *testing.T) {
	value, err := enum.ResourceUnknown.Value()
	require.NoError(t, err)
	require.Equal(t, "unknown", value)

	value, err = enum.ResourceTransaction.Value()
	require.NoError(t, err)
	require.Equal(t, "transaction", value)

	value, err = enum.ResourceUser.Value()
	require.NoError(t, err)
	require.Equal(t, "user", value)

	value, err = enum.ResourceAPIKey.Value()
	require.NoError(t, err)
	require.Equal(t, "api_key", value)

	value, err = enum.ResourceCounterparty.Value()
	require.NoError(t, err)
	require.Equal(t, "counterparty", value)

	value, err = enum.ResourceAccount.Value()
	require.NoError(t, err)
	require.Equal(t, "account", value)

	value, err = enum.ResourceSunrise.Value()
	require.NoError(t, err)
	require.Equal(t, "sunrise", value)

}
