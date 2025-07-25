package enum_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/enum"
)

func TestParseActor(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		tests := []struct {
			input    interface{}
			expected enum.Actor
		}{
			{"", enum.ActorUnknown},
			{"unknown", enum.ActorUnknown},
			{"UNKNOWN", enum.ActorUnknown},
			{"user", enum.ActorUser},
			{"USER", enum.ActorUser},
			{"api_key", enum.ActorAPIKey},
			{"API_KEY", enum.ActorAPIKey},
			{"sunrise", enum.ActorSunrise},
			{"SUNRISE", enum.ActorSunrise},
			{"cli", enum.ActorCLI},
			{"CLI", enum.ActorCLI},
			{uint8(0), enum.ActorUnknown},
			{uint8(1), enum.ActorUser},
			{uint8(2), enum.ActorAPIKey},
			{uint8(3), enum.ActorSunrise},
			{uint8(4), enum.ActorCLI},
			{enum.ActorUnknown, enum.ActorUnknown},
			{enum.ActorUser, enum.ActorUser},
			{enum.ActorAPIKey, enum.ActorAPIKey},
			{enum.ActorSunrise, enum.ActorSunrise},
			{enum.ActorCLI, enum.ActorCLI},
		}

		for i, test := range tests {
			result, err := enum.ParseActor(test.input)
			require.NoError(t, err, "test case %d failed", i)
			require.Equal(t, test.expected, result, "test case %d failed", i)
		}
	})

	t.Run("Errors", func(t *testing.T) {
		tests := []struct {
			input interface{}
			errs  string
		}{
			{"aloha", "invalid actor: \"aloha\""},
			{true, "cannot parse bool into an actor"},
		}

		for i, test := range tests {
			result, err := enum.ParseActor(test.input)
			require.Equal(t, enum.ActorUnknown, result, "test case %d failed", i)
			require.EqualError(t, err, test.errs, "test case %d failed", i)
		}
	})
}

func TestActorString(t *testing.T) {
	tests := []struct {
		actor    enum.Actor
		expected string
	}{
		{enum.ActorUnknown, "unknown"},
		{enum.ActorUser, "user"},
		{enum.ActorAPIKey, "api_key"},
		{enum.ActorSunrise, "sunrise"},
		{enum.ActorCLI, "cli"},
		{enum.Actor(5), "unknown"},
		{enum.Actor(99), "unknown"},
	}

	for i, test := range tests {
		result := test.actor.String()
		require.Equal(t, test.expected, result, "test case %d failed", i)
	}
}

func TestActorJSON(t *testing.T) {
	tests := []enum.Actor{
		enum.ActorUnknown,
		enum.ActorUser,
		enum.ActorAPIKey,
		enum.ActorSunrise,
		enum.ActorCLI,
	}

	for _, actor := range tests {
		data, err := json.Marshal(actor)
		require.NoError(t, err)

		var result enum.Actor
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)
		require.Equal(t, actor, result)
	}
}

func TestActorScan(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected enum.Actor
	}{
		{nil, enum.ActorUnknown},
		{"", enum.ActorUnknown},
		{"unknown", enum.ActorUnknown},
		{"UNKNOWN", enum.ActorUnknown},
		{"user", enum.ActorUser},
		{"USER", enum.ActorUser},
		{"api_key", enum.ActorAPIKey},
		{"API_KEY", enum.ActorAPIKey},
		{"sunrise", enum.ActorSunrise},
		{"SUNRISE", enum.ActorSunrise},
		{"cli", enum.ActorCLI},
		{"CLI", enum.ActorCLI},
		{[]byte(""), enum.ActorUnknown},
		{[]byte("unknown"), enum.ActorUnknown},
		{[]byte("UNKNOWN"), enum.ActorUnknown},
		{[]byte("user"), enum.ActorUser},
		{[]byte("USER"), enum.ActorUser},
		{[]byte("api_key"), enum.ActorAPIKey},
		{[]byte("API_KEY"), enum.ActorAPIKey},
		{[]byte("sunrise"), enum.ActorSunrise},
		{[]byte("SUNRISE"), enum.ActorSunrise},
		{[]byte("cli"), enum.ActorCLI},
		{[]byte("CLI"), enum.ActorCLI},
	}

	for i, test := range tests {
		var actor enum.Actor
		err := actor.Scan(test.input)
		require.NoError(t, err, "test case %d failed", i)
		require.Equal(t, test.expected, actor, "test case %d failed", i)
	}

	var d enum.Actor
	err := d.Scan("aloha")
	require.EqualError(t, err, "invalid actor: \"aloha\"")
	err = d.Scan(true)
	require.EqualError(t, err, "cannot scan bool into an actor")
}

func TestActorValue(t *testing.T) {
	value, err := enum.ActorUnknown.Value()
	require.NoError(t, err)
	require.Equal(t, "unknown", value)

	value, err = enum.ActorUser.Value()
	require.NoError(t, err)
	require.Equal(t, "user", value)

	value, err = enum.ActorAPIKey.Value()
	require.NoError(t, err)
	require.Equal(t, "api_key", value)

	value, err = enum.ActorSunrise.Value()
	require.NoError(t, err)
	require.Equal(t, "sunrise", value)

	value, err = enum.ActorCLI.Value()
	require.NoError(t, err)
	require.Equal(t, "cli", value)
}
