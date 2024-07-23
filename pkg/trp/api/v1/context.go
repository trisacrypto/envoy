package api

import "context"

// API-specific context keys for passing values to requests via the context. These keys
// are unexported to reduce the size of the public interface an prevent incorrect handling.
type contextKey uint8

// Allocate context keys to simplify context key usage in helper functions.
const (
	contextKeyUnknown contextKey = iota
	contextKeyRequestID
)

// Adds a request ID to the context which is sent with the request in the X-Request-ID header.
func ContextWithRequestID(parent context.Context, requestID string) context.Context {
	return context.WithValue(parent, contextKeyRequestID, requestID)
}

// Extracts a request ID from the context.
func RequestIDFromContext(ctx context.Context) (string, bool) {
	requestID, ok := ctx.Value(contextKeyRequestID).(string)
	return requestID, ok
}

var contextKeyNames = []string{"unknown", "requestID"}

// String returns a human readable representation of the context key for easier debugging.
func (c contextKey) String() string {
	if int(c) < len(contextKeyNames) {
		return contextKeyNames[c]
	}
	return contextKeyNames[0]
}
