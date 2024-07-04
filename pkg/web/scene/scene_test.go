package scene_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/web/scene"
)

func TestUpdate(t *testing.T) {
	alpha := scene.New(nil)
	alpha["Fruit"] = "Orange"
	alpha["Age"] = 42

	bravo := scene.Scene{
		scene.Version: "v0.0.1",
		"Fruit":       "Orange",
		"Name":        "Roger",
	}

	// Assert original
	require.Len(t, alpha, 3)
	require.Len(t, bravo, 3)

	// Update alpha from bravo
	alpha.Update(bravo)
	require.Len(t, alpha, 4)
	require.Len(t, bravo, 3)

	// Check the update happened correctly
	expected := scene.Scene{
		scene.Version: "v0.0.1",
		"Fruit":       "Orange",
		"Name":        "Roger",
		"Age":         42,
	}
	require.Equal(t, expected, alpha)
}
