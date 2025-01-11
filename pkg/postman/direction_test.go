package postman_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/postman"
)

func TestDirectionString(t *testing.T) {
	tests := []struct {
		direction postman.Direction
		expected  string
	}{
		{postman.Unknown, "unknown"},
		{postman.DirectionIncoming, "incoming"},
		{postman.DirectionOutgoing, "outgoing"},
	}

	for _, test := range tests {
		require.Equal(t, test.expected, test.direction.String())
	}
}

func TestDirectionValid(t *testing.T) {
	tests := []struct {
		direction postman.Direction
		expected  bool
	}{
		{postman.Unknown, false},
		{postman.DirectionIncoming, true},
		{postman.DirectionOutgoing, true},
		{postman.Direction(3), false},
	}

	for _, test := range tests {
		require.Equal(t, test.expected, test.direction.Valid())
	}
}
