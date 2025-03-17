package enum_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/enum"
)

func TestDirectionString(t *testing.T) {
	tests := []struct {
		direction enum.Direction
		expected  string
	}{
		{enum.DirectionUnknown, "unknown"},
		{enum.DirectionIncoming, "in"},
		{enum.DirectionOutgoing, "out"},
	}

	for _, test := range tests {
		result := test.direction.String()
		require.Equal(t, test.expected, result)
	}
}
