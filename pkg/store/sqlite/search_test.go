package sqlite_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	. "github.com/trisacrypto/envoy/pkg/store/sqlite"
)

func TestIsURLCandidate(t *testing.T) {
	t.Run("True", func(t *testing.T) {
		tests := []string{
			"https://example.com",
			"http://example.com",
			"example.com",
			"www.example.com",
			"testnet.travel-rule.example.com",
			"example.com/path/to/resource",
			"envoy.local",
			"trisa.example.tr-envoy.com:443",
			"envoy.local:8100",
			"https://envoy.local:8100",
		}

		for _, test := range tests {
			require.True(t, IsURLCandidate(test), "expected %q to be a URL candidate", test)
		}
	})

	t.Run("False", func(t *testing.T) {
		tests := []string{
			"Rotational Labs",
			"rotational",
			"bit4x",
			"",
			"https://",
			"foo:80",
		}

		for _, test := range tests {
			require.False(t, IsURLCandidate(test), "expected %q to not be a URL candidate", test)
		}
	})
}

func TestNormURL(t *testing.T) {
	tests := []struct {
		in       string
		expected string
	}{
		{"", ""},
		{"https://example.com", "example.com"},
		{"http://example.com", "example.com"},
		{"example.com", "example.com"},
		{"www.example.com", "www.example.com"},
		{"testnet.travel-rule.example.com", "testnet.travel-rule.example.com"},
		{"example.com/path/to/resource", "example.com"},
		{"envoy.local", "envoy.local"},
		{"trisa.example.tr-envoy.com:443", "trisa.example.tr-envoy.com:443"},
		{"envoy.local:8100", "envoy.local:8100"},
		{"https://envoy.local:8100", "envoy.local:8100"},
		{"foo", ""},
	}

	for _, test := range tests {
		require.Equal(t, test.expected, NormURL(test.in), "expected %q to normalize to %q", test.in, test.expected)
	}
}
