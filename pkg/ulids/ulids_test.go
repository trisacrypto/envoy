package ulids_test

import (
	"sync"
	"testing"

	. "github.com/trisacrypto/envoy/pkg/ulids"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestIsZero(t *testing.T) {
	testCases := []struct {
		input  ulid.ULID
		assert require.BoolAssertionFunc
	}{
		{ulid.ULID{}, require.True},
		{ulid.ULID{0x00}, require.True},
		{ulid.ULID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, require.True},
		{ulid.Make(), require.False},
	}

	for _, tc := range testCases {
		tc.assert(t, IsZero(tc.input))
	}
}

func TestCheckIDMatch(t *testing.T) {
	alpha := New()
	bravo := New()

	testCases := []struct {
		id     ulid.ULID
		target ulid.ULID
		err    error
	}{
		{Null, alpha, ErrMissingID},
		{Null, Null, ErrMissingID},
		{alpha, Null, ErrIDMismatch},
		{alpha, bravo, ErrIDMismatch},
		{alpha, alpha, nil},
		{bravo, bravo, nil},
	}

	for i, tc := range testCases {
		require.ErrorIs(t, CheckIDMatch(tc.id, tc.target), tc.err, "test case %d failed", i)
	}
}

func TestNew(t *testing.T) {
	// Should be able to concurrently create 100
	var wg sync.WaitGroup
	wg.Add(100)
	for i := 0; i < 100; i++ {
		go func() {
			defer wg.Done()
			uid := New()
			require.False(t, IsZero(uid))
		}()
	}
	wg.Wait()
}

func TestParse(t *testing.T) {
	example := New()

	testCases := []struct {
		input    any
		expected ulid.ULID
		err      error
	}{
		{example.String(), example, nil},
		{example.Bytes(), example, nil},
		{example, example, nil},
		{[16]byte(example), example, nil},
		{"", Null, nil},
		{uint64(14), Null, ErrUnknownType},
		{"foo", Null, ulid.ErrDataSize},
		{[]byte{0x14, 0x21}, Null, ulid.ErrDataSize},
		{Null.String(), Null, nil},
	}

	for i, tc := range testCases {
		actual, err := Parse(tc.input)
		require.ErrorIs(t, err, tc.err, "could not compare error on test case %d", i)
		require.Equal(t, tc.expected, actual, "expected result not returned")

		if tc.err != nil {
			require.Panics(t, func() { MustParse(tc.input) })
		} else {
			require.Equal(t, tc.expected, MustParse(tc.input))
		}
	}
}
