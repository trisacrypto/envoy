package scene_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/web/scene"
)

func TestToastMessages(t *testing.T) {

	// Test the MarshalCookie and UnmarshalCookie methods
	toastMessages := scene.ToastMessages{
		{Heading: "Test Heading", Message: "Test Message", Type: "success"},
		{Heading: "Test Warning", Message: "Something went wrong", Type: "danger"},
	}

	marshaled := toastMessages.MarshalCookie()
	require.NotEmpty(t, marshaled, "marshaled cookie should not be empty")

	var unmarshaled scene.ToastMessages
	unmarshaled.UnmarshalCookie(marshaled)
	require.Len(t, unmarshaled, 2, "unmarshaled cookie should contain 2 messages")
	require.Equal(t, toastMessages, unmarshaled, "unmarshaled messages should match original")
}
