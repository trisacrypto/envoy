package enum_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/enum"
)

func TestValidStatus(t *testing.T) {
	tests := []struct {
		input  interface{}
		assert require.BoolAssertionFunc
	}{
		{"", require.True},
		{"unspecified", require.True},
		{"draft", require.True},
		{"pending", require.True},
		{"review", require.True},
		{"repair", require.True},
		{"accepted", require.True},
		{"completed", require.True},
		{"rejected", require.True},
		{uint8(0), require.True},
		{uint8(1), require.True},
		{uint8(2), require.True},
		{uint8(3), require.True},
		{uint8(4), require.True},
		{uint8(5), require.True},
		{uint8(6), require.True},
		{uint8(7), require.True},
		{enum.StatusUnspecified, require.True},
		{enum.StatusDraft, require.True},
		{enum.StatusPending, require.True},
		{enum.StatusReview, require.True},
		{enum.StatusRepair, require.True},
		{enum.StatusAccepted, require.True},
		{enum.StatusCompleted, require.True},
		{enum.StatusRejected, require.True},
		{"foo", require.False},
		{true, require.False},
		{uint8(99), require.False},
	}

	for i, tc := range tests {
		tc.assert(t, enum.ValidStatus(tc.input), "test case %d failed", i)
	}
}

func TestCheckStatus(t *testing.T) {
	tests := []struct {
		input   interface{}
		targets []enum.Status
		assert  require.BoolAssertionFunc
		err     error
	}{
		{"", []enum.Status{enum.StatusUnspecified, enum.StatusRepair, enum.StatusAccepted}, require.True, nil},
		{"unspecified", []enum.Status{enum.StatusRepair, enum.StatusAccepted, enum.StatusUnspecified}, require.True, nil},
		{"accepted", []enum.Status{enum.StatusRepair, enum.StatusAccepted, enum.StatusUnspecified}, require.True, nil},
		{"draft", []enum.Status{enum.StatusRepair, enum.StatusAccepted}, require.False, nil},
		{"foo", []enum.Status{enum.StatusRepair, enum.StatusAccepted}, require.False, errors.New(`invalid status: "foo"`)},
		{"", []enum.Status{enum.StatusRepair, enum.StatusAccepted}, require.False, nil},
		{"unspecified", []enum.Status{enum.StatusRepair, enum.StatusAccepted}, require.False, nil},
		{"pending", []enum.Status{enum.StatusRepair, enum.StatusAccepted}, require.False, nil},
	}

	for i, tc := range tests {
		result, err := enum.CheckStatus(tc.input, tc.targets...)
		tc.assert(t, result, "test case %d failed", i)

		if tc.err != nil {
			require.Equal(t, tc.err, err, "test case %d failed", i)
		} else {
			require.NoError(t, err, "test case %d failed", i)
		}
	}
}

func TestParseStatus(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		tests := []struct {
			input    interface{}
			expected enum.Status
		}{
			{"", enum.StatusUnspecified},
			{"unspecified", enum.StatusUnspecified},
			{"draft", enum.StatusDraft},
			{"pending", enum.StatusPending},
			{"review", enum.StatusReview},
			{"repair", enum.StatusRepair},
			{"accepted", enum.StatusAccepted},
			{"completed", enum.StatusCompleted},
			{"rejected", enum.StatusRejected},
			{uint8(0), enum.StatusUnspecified},
			{uint8(1), enum.StatusDraft},
			{uint8(2), enum.StatusPending},
			{uint8(3), enum.StatusReview},
			{uint8(4), enum.StatusRepair},
			{uint8(5), enum.StatusAccepted},
			{uint8(6), enum.StatusCompleted},
			{uint8(7), enum.StatusRejected},
			{enum.StatusUnspecified, enum.StatusUnspecified},
			{enum.StatusDraft, enum.StatusDraft},
			{enum.StatusPending, enum.StatusPending},
			{enum.StatusReview, enum.StatusReview},
			{enum.StatusRepair, enum.StatusRepair},
			{enum.StatusAccepted, enum.StatusAccepted},
			{enum.StatusCompleted, enum.StatusCompleted},
			{enum.StatusRejected, enum.StatusRejected},
		}

		for i, test := range tests {
			result, err := enum.ParseStatus(test.input)
			require.NoError(t, err, "test case %d failed", i)
			require.Equal(t, test.expected, result, "test case %d failed", i)
		}
	})

	t.Run("Errors", func(t *testing.T) {
		tests := []struct {
			input interface{}
			errs  string
		}{
			{"foo", "invalid status: \"foo\""},
			{true, "cannot parse bool into a status"},
		}

		for i, test := range tests {
			result, err := enum.ParseStatus(test.input)
			require.Equal(t, enum.StatusUnspecified, result, "test case %d failed", i)
			require.EqualError(t, err, test.errs, "test case %d failed", i)
		}
	})
}

func TestStatusString(t *testing.T) {
	tests := []struct {
		status   enum.Status
		expected string
	}{
		{enum.StatusUnspecified, "unspecified"},
		{enum.StatusDraft, "draft"},
		{enum.StatusPending, "pending"},
		{enum.StatusReview, "review"},
		{enum.StatusRepair, "repair"},
		{enum.StatusAccepted, "accepted"},
		{enum.StatusCompleted, "completed"},
		{enum.StatusRejected, "rejected"},
		{enum.Status(99), "unspecified"},
	}

	for i, test := range tests {
		result := test.status.String()
		require.Equal(t, test.expected, result, "test case %d failed", i)
	}
}

func TestStatusJSON(t *testing.T) {
	tests := []enum.Status{
		enum.StatusUnspecified, enum.StatusDraft, enum.StatusPending,
		enum.StatusReview, enum.StatusRepair, enum.StatusAccepted,
		enum.StatusCompleted, enum.StatusRejected,
	}

	for _, status := range tests {
		data, err := json.Marshal(status)
		require.NoError(t, err)

		var result enum.Status
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)
		require.Equal(t, status, result)
	}
}

func TestStatusScan(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected enum.Status
	}{
		{nil, enum.StatusUnspecified},
		{"", enum.StatusUnspecified},
		{"unspecified", enum.StatusUnspecified},
		{"draft", enum.StatusDraft},
		{"pending", enum.StatusPending},
		{"review", enum.StatusReview},
		{"repair", enum.StatusRepair},
		{"accepted", enum.StatusAccepted},
		{"completed", enum.StatusCompleted},
		{"rejected", enum.StatusRejected},
		{[]byte(""), enum.StatusUnspecified},
		{[]byte("unspecified"), enum.StatusUnspecified},
		{[]byte("draft"), enum.StatusDraft},
		{[]byte("pending"), enum.StatusPending},
		{[]byte("review"), enum.StatusReview},
		{[]byte("repair"), enum.StatusRepair},
		{[]byte("accepted"), enum.StatusAccepted},
		{[]byte("completed"), enum.StatusCompleted},
		{[]byte("rejected"), enum.StatusRejected},
	}

	for i, test := range tests {
		var status enum.Status
		err := status.Scan(test.input)
		require.NoError(t, err, "test case %d failed", i)
		require.Equal(t, test.expected, status, "test case %d failed", i)
	}

	var d enum.Status
	err := d.Scan("foo")
	require.EqualError(t, err, "invalid status: \"foo\"")
	err = d.Scan(true)
	require.EqualError(t, err, "cannot scan bool into a status")
}

func TestStatusValue(t *testing.T) {
	value, err := enum.StatusDraft.Value()
	require.NoError(t, err)
	require.Equal(t, "draft", value)
}
