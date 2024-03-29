package logger_test

import (
	"context"
	"testing"

	"self-hosted-node/pkg/logger"
	"self-hosted-node/pkg/ulids"

	"github.com/stretchr/testify/require"
)

func TestRequestIDContext(t *testing.T) {
	requestID := ulids.New().String()
	parent, cancel := context.WithCancel(context.Background())
	ctx := logger.WithRequestID(parent, requestID)

	cmp, ok := logger.RequestID(ctx)
	require.True(t, ok)
	require.Equal(t, requestID, cmp)

	cancel()
	require.ErrorIs(t, ctx.Err(), context.Canceled)
}
