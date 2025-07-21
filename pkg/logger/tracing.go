package logger

import (
	"context"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/trisacrypto/envoy/pkg/contextkey"
)

func Tracing(ctx context.Context) zerolog.Logger {
	requestID, _ := RequestID(ctx)
	return log.With().Str("request_id", requestID).Logger()
}

func WithRequestID(parent context.Context, requestID string) context.Context {
	return context.WithValue(parent, contextkey.KeyRequestID, requestID)
}

func RequestID(ctx context.Context) (string, bool) {
	requestID, ok := ctx.Value(contextkey.KeyRequestID).(string)
	return requestID, ok
}
