package audit_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/audit"
	"github.com/trisacrypto/envoy/pkg/store/mock"
	"github.com/trisacrypto/envoy/pkg/trisa/keychain"
	"github.com/trisacrypto/trisa/pkg/trisa/keys"
	"github.com/trisacrypto/trisa/pkg/trust"
)

func TestSignVerify(t *testing.T) {
	//setup
	log := mock.GetComplianceAuditLog(true, false)
	loadAuditKeyChainFixture(t)

	// tests
	err := audit.Sign(log)
	require.NoError(t, err, "couldn't sign the data")
	require.NotNil(t, log.Signature, "signature shouldn't be nil")
	require.NotZero(t, log.Algorithm, "algorithm shouldn't be the empty string")
	require.NotZero(t, log.KeyID, "key id shouldn't be the empty string")

	err = audit.Verify(log)
	require.NoError(t, err, "data was not verified")
}

// ===========================================================================
// Helpers
// ===========================================================================

func loadAuditKeyChainFixture(t *testing.T) {
	// Load Certificate fixture with private keys
	sz, err := trust.NewSerializer(false)
	require.NoError(t, err, "could not create serializer to load fixture")

	provider, err := sz.ReadFile("testdata/certs.pem")
	require.NoError(t, err, "could not read test fixture")

	certs, err := keys.FromProvider(provider)
	require.NoError(t, err, "could not create Key from provider")
	require.True(t, certs.IsPrivate(), "expected test certs fixture to be private")

	// Setup a mock KeyChain
	kc, err := keychain.New(keychain.WithCacheDuration(1*time.Hour), keychain.WithDefaultKey(certs))
	require.NoError(t, err, "could not create a KeyChain")
	audit.UseKeyChain(kc)
}
