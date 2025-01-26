package emails_test

import (
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/joho/godotenv"
	"github.com/rotationalio/confire"
	"github.com/stretchr/testify/require"
	. "github.com/trisacrypto/envoy/pkg/emails"
	"github.com/trisacrypto/envoy/pkg/sunrise"
)

func TestLiveEmails(t *testing.T) {
	// Load local .env if it exists to make setting envvars easier.
	godotenv.Load()

	// This test will send actual emails to an account as configured by the environment.
	// The $TEST_LIVE_EMAILS environment variable must be set to true to not skip.
	SkipByEnvVar(t, "TEST_LIVE_EMAILS")
	CheckEnvVars(t, "TEST_LIVE_EMAIL_RECIPIENT")

	// Configure email sending from the environment. See .env.template for requirements.
	conf := Config{}
	err := confire.Process("trisa_email", &conf)
	require.NoError(t, err, "environment not setup to send live emails; see .env.template")
	require.True(t, conf.Available(), "no backend setup to send live emails; see .env.template")
	require.NoError(t, Configure(conf), "could not configure email sending")

	recipient := os.Getenv("TEST_LIVE_EMAIL_RECIPIENT")

	t.Run("Invite", func(t *testing.T) {
		data := SunriseInviteData{
			ContactName:     "Charlie Brown",
			ComplianceName:  "Testing Compliance",
			OriginatorName:  "Alice Duncan",
			BeneficiaryName: "Benedict Smith",
			BaseURL:         &url.URL{Scheme: "http", Host: "envoy.local:8000", Path: "/sunrise/verify"},
			Token:           sunrise.VerificationToken("abc123"),
			SupportEmail:    "support@example.com",
			ComplianceEmail: "compliance@example.com",
		}

		email, err := NewSunriseInvite(recipient, data)
		require.NoError(t, err, "could not create sunrise invite email")

		err = email.Send()
		require.NoError(t, err, "could not send sunrise invite email")
	})

	t.Run("Verify", func(t *testing.T) {
		data := VerifyEmailData{
			Code:           "ABC123",
			ComplianceName: "Testing Compliance",
			SupportEmail:   "support@example.com",
		}

		email, err := NewVerifyEmail(recipient, data)
		require.NoError(t, err, "could not create sunrise verify email")

		err = email.Send()
		require.NoError(t, err, "could not send sunrise verify email")
	})
}

func CheckEnvVars(t *testing.T, envs ...string) {
	for _, env := range envs {
		require.NotEmpty(t, os.Getenv(env), "required environment variable $%s not set", env)
	}
}

func SkipByEnvVar(t *testing.T, env string) {
	val := strings.ToLower(strings.TrimSpace(os.Getenv(env)))
	switch val {
	case "1", "t", "true":
		return
	default:
		t.Skipf("this test depends on the $%s envvar to run", env)
	}
}
