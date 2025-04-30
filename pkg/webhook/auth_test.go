package webhook_test

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/webhook"
)

func TestHMAC(t *testing.T) {
	key, _ := hex.DecodeString("cfbabc4715b4759d45ba26953dd2fc0bfc2344ef70a2005432e7f16b5081610d")
	keyID := "01JT4B3R5Z6AHJXV87QHPPKRBM"

	mac := webhook.NewHMAC(keyID, key)
	mac.AddHeader("Host", "example.com")
	mac.AddHeader("X-Transaction-ID", "f17c9693-c280-4836-b544-245d832a11e0")

	auth, err := mac.Authorization()
	require.NoError(t, err, "could not create authorization header")
	require.NotEmpty(t, auth, "authorization header is empty")
}
