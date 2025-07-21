package audit

import (
	"context"

	"github.com/trisacrypto/envoy/pkg/contextkey"
	"github.com/trisacrypto/envoy/pkg/enum"
)

// Adds the actor id and type to the given context that can be retrieved later
// with ActorID() and ActorType().
func WithActor(parent context.Context, actorID []byte, actorType enum.Actor) (ctx context.Context) {
	ctx = context.WithValue(parent, contextkey.KeyActorID, actorID)
	ctx = context.WithValue(ctx, contextkey.KeyActorType, actorType)
	return ctx
}

// Returns the context's actor ID and true, if present, otherwise returns
// false for the second value (first value not useful).
func ActorID(ctx context.Context) (requestID []byte, ok bool) {
	requestID, ok = ctx.Value(contextkey.KeyActorID).([]byte)
	return requestID, ok
}

// Returns the context's actor type and true, if present, otherwise returns
// false for the second value (first value not useful).
func ActorType(ctx context.Context) (actorType enum.Actor, ok bool) {
	actorType, ok = ctx.Value(contextkey.KeyActorType).(enum.Actor)
	return actorType, ok
}
