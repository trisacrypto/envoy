package audit_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/audit"
	"github.com/trisacrypto/envoy/pkg/enum"
	"go.rtnl.ai/ulid"
)

func TestActorContext(t *testing.T) {
	// fresh context
	ctx := context.Background()

	// actor metadata should not be present
	_, ok := audit.ActorID(ctx)
	require.False(t, ok, "expected no actor id")
	_, ok = audit.ActorType(ctx)
	require.False(t, ok, "expected no actor type")

	// add actor metadata
	actorID := ulid.MakeSecure()
	ctx = audit.WithActor(ctx, actorID.Bytes(), enum.ActorAPIKey)

	// actor metadata should be present
	storedID, ok := audit.ActorID(ctx)
	require.True(t, ok, "expected an actor id")
	require.Equal(t, actorID.Bytes(), storedID, "stored id not equal to expected id")

	storedType, ok := audit.ActorType(ctx)
	require.True(t, ok, "expected an actor type")
	require.Equal(t, enum.ActorAPIKey, storedType, "stored type not equal to expected type")
}
