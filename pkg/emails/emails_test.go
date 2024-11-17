package emails_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/emails"
)

func TestEmailValidate(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		testCases := []*emails.Email{
			{
				"admin@server.com",
				[]string{"test@example.com"},
				"This is a test email",
				"test",
				nil,
			},
			{
				"admin@server.com",
				[]string{"test@example.com"},
				"This is a test email",
				"test",
				map[string]interface{}{"count": 4},
			},
		}

		for i, email := range testCases {
			require.NoError(t, email.Validate(), "test case %d failed", i)
		}
	})

	t.Run("Invalid", func(t *testing.T) {
		testCases := []struct {
			email *emails.Email
			err   error
		}{
			{
				&emails.Email{
					Sender:   "admin@server.com",
					To:       []string{"test@example.com"},
					Template: "test",
					Scene:    nil,
				},
				emails.ErrMissingSubject,
			},
			{
				&emails.Email{
					To:       []string{"test@example.com"},
					Subject:  "This is a test email",
					Template: "test",
					Scene:    nil,
				},
				emails.ErrMissingSender,
			},
			{
				&emails.Email{
					Sender:   "admin@server.com",
					Subject:  "This is a test email",
					Template: "test",
					Scene:    nil,
				},
				emails.ErrMissingRecipient,
			},
			{
				&emails.Email{
					Sender:  "admin@server.com",
					To:      []string{"test@example.com"},
					Subject: "This is a test email",
					Scene:   nil,
				},
				emails.ErrMissingTemplate,
			},
			{
				&emails.Email{
					Sender:   "admin@@server",
					To:       []string{"test@example.com"},
					Subject:  "This is a test email",
					Template: "test",
					Scene:    nil,
				},
				emails.ErrIncorrectEmail,
			},
			{
				&emails.Email{
					Sender:   "admin@server.com",
					To:       []string{"@example.com"},
					Subject:  "This is a test email",
					Template: "test",
					Scene:    nil,
				},
				emails.ErrIncorrectEmail,
			},
		}

		for i, tc := range testCases {
			require.ErrorIs(t, tc.email.Validate(), tc.err, "test case %d failed", i)
		}
	})

}
