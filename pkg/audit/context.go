package audit

import (
	"context"

	"github.com/trisacrypto/envoy/pkg/enum"
)

// ###########################################################################
// Audit log context.Context tools
// ###########################################################################

// Adds the actor id and type to the given context that can be retrieved later
// with ActorID() and ActorType().
func WithActor(parent context.Context, actorID []byte, actorType enum.Actor) (ctx context.Context) {
	ctx = context.WithValue(parent, KeyActorID, actorID)
	ctx = context.WithValue(ctx, KeyActorType, actorType)
	return ctx
}

// Returns the context's actor ID and true, if present, otherwise returns
// false for the second value (first value not useful).
func ActorID(ctx context.Context) (actorID []byte, ok bool) {
	actorID, ok = ctx.Value(KeyActorID).([]byte)
	return actorID, ok
}

// Returns the context's actor type and true, if present, otherwise returns
// false for the second value (first value not useful).
func ActorType(ctx context.Context) (actorType enum.Actor, ok bool) {
	actorType, ok = ctx.Value(KeyActorType).(enum.Actor)
	return actorType, ok
}

// ###########################################################################
// Context keys for audit log
// ###########################################################################

type contextKey uint8

const (
	KeyUnknown contextKey = iota
	KeyActorID
	KeyActorType
)

var contextKeyNames = []string{"unknown", "actorID", "actorType"}

func (c contextKey) String() string {
	if int(c) < len(contextKeyNames) {
		return contextKeyNames[c]
	}
	return contextKeyNames[0]
}
