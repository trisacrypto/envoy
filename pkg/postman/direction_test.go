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
		{postman.DirectionUnknown, "unknown"},
		{postman.DirectionIncoming, "incoming"},
		{postman.DirectionOutgoing, "outgoing"},
	}

	for _, test := range tests {
		result := test.direction.String()
		require.Equal(t, test.expected, result)
	}
}
