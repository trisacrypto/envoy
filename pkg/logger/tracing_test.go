package logger_test

import (
	"context"
	"testing"

	"github.com/trisacrypto/envoy/pkg/logger"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/ulid"
)

func TestRequestIDContext(t *testing.T) {
	requestID := ulid.Make().String()
	parent, cancel := context.WithCancel(context.Background())
	ctx := logger.WithRequestID(parent, requestID)

	cmp, ok := logger.RequestID(ctx)
	require.True(t, ok)
	require.Equal(t, requestID, cmp)

	cancel()
	require.ErrorIs(t, ctx.Err(), context.Canceled)
}
