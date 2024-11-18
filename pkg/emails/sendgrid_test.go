package emails_test

import (
	"testing"

	sgmail "github.com/sendgrid/sendgrid-go/helpers/mail"
	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/emails"
)

func TestNewSGEmail(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		testCases := []struct {
			email    string
			expected *sgmail.Email
		}{
			{
				"jlong@example.com",
				&sgmail.Email{Name: "", Address: "jlong@example.com"},
			},
			{
				"Jersey Long <jlong@example.com>",
				&sgmail.Email{Name: "Jersey Long", Address: "jlong@example.com"},
			},
		}

		for i, tc := range testCases {
			sgm, err := emails.NewSGEmail(tc.email)
			require.NoError(t, err, "test case %d errored", i)
			require.Equal(t, tc.expected, sgm, "test case %d mismatch", i)
			require.Equal(t, tc.expected, emails.MustNewSGEmail(tc.email), "test case %d panic", i)
		}
	})

	t.Run("Invalid", func(t *testing.T) {
		testCases := []string{
			"foo",
			"Lacy Credence <foo>",
			"foo@@foo",
		}

		for i, email := range testCases {
			sgm, err := emails.NewSGEmail(email)
			require.Error(t, err, "test case %d did not error", i)
			require.Nil(t, sgm, "test case %d message was not nil", i)
			require.Panics(t, func() { emails.MustNewSGEmail(email) }, "test case %d did not panic", i)
		}
	})
}

func TestNewSGEmails(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		testCases := []struct {
			emails   []string
			expected []*sgmail.Email
		}{
			{
				nil,
				[]*sgmail.Email{},
			},
			{
				[]string{"jlong@example.com"},
				[]*sgmail.Email{{Name: "", Address: "jlong@example.com"}},
			},
			{
				[]string{"Jersey Long <jlong@example.com>"},
				[]*sgmail.Email{{Name: "Jersey Long", Address: "jlong@example.com"}},
			},
			{
				[]string{"jlong@example.com", "Frieda Short <fshort@example.com>"},
				[]*sgmail.Email{{Name: "", Address: "jlong@example.com"}, {Name: "Frieda Short", Address: "fshort@example.com"}},
			},
		}

		for i, tc := range testCases {
			sgm, err := emails.NewSGEmails(tc.emails)
			require.NoError(t, err, "test case %d errored", i)
			require.Equal(t, tc.expected, sgm, "test case %d mismatch", i)
			require.Equal(t, tc.expected, emails.MustNewSGEmails(tc.emails), "test case %d panic", i)
		}
	})

	t.Run("Invalid", func(t *testing.T) {
		testCases := [][]string{
			{
				"foo",
			},
			{
				"Larry Helmand <lh@example.com>", "lh@example.com", "bad",
			},
			{
				"foo",
				"Lacy Credence <foo>",
				"foo@@foo",
			},
		}

		for i, email := range testCases {
			sgm, err := emails.NewSGEmails(email)
			require.Error(t, err, "test case %d did not error", i)
			require.Nil(t, sgm, "test case %d message was not nil", i)
			require.Panics(t, func() { emails.MustNewSGEmails(email) }, "test case %d did not panic", i)
		}
	})
}
