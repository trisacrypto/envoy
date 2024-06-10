package auth_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/web/auth"
)

func TestIsLocalhost(t *testing.T) {
	testCases := []struct {
		domain string
		assert require.BoolAssertionFunc
	}{
		{
			"localhost",
			require.True,
		},
		{
			"envoy.local",
			require.True,
		},
		{
			"counterparty.local",
			require.True,
		},
		{
			"beneficiary",
			require.False,
		},
		{
			"beneficiary.com",
			require.False,
		},
		{
			"trisa.example.tr-envoy.com",
			require.False,
		},
		{
			"beneficiary.local.example.io",
			require.False,
		},
	}

	for i, tc := range testCases {
		tc.assert(t, auth.IsLocalhost(tc.domain), "test case %d failed", i)
	}
}
