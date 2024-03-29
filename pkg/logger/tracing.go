package logger

import "context"

type contextKey uint8

const (
	KeyUnknown contextKey = iota
	KeyRequestID
)

func WithRequestID(parent context.Context, requestID string) context.Context {
	return context.WithValue(parent, KeyRequestID, requestID)
}

func RequestID(ctx context.Context) (string, bool) {
	requestID, ok := ctx.Value(KeyRequestID).(string)
	return requestID, ok
}

var contextKeyNames = []string{"unknown", "requestID"}

func (c contextKey) String() string {
	if int(c) < len(contextKeyNames) {
		return contextKeyNames[c]
	}
	return contextKeyNames[0]
}
