package emails_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/emails"
	"github.com/trisacrypto/envoy/pkg/verification"
)

func TestVerifyURL(t *testing.T) {
	invite := emails.SunriseInviteData{
		BaseURL: &url.URL{
			Scheme: "https",
			Host:   "sunrise.example.com",
			Path:   "/v1/sunrise/verify",
		},
		Token: verification.VerificationToken("abc123"),
	}

	require.Equal(t, "https://sunrise.example.com/v1/sunrise/verify?token=YWJjMTIz", invite.VerifyURL())
}
