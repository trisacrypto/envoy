package emails

import (
	"fmt"
	"net/mail"

	"github.com/jordan-wright/email"

	sgmail "github.com/sendgrid/sendgrid-go/helpers/mail"
)

type Email struct {
	Sender   string
	To       []string
	Subject  string
	Template string
	Data     interface{}
}

// New creates a new email template with the currently configured sender attached. If
// the sender is not configured, then it is left empty; otherwise if the module has
// been configured and there is no sender, an error is returned.
func New(recipient, subject, template string, data interface{}) (*Email, error) {
	msg := &Email{
		To:       []string{recipient},
		Subject:  subject,
		Template: template,
		Data:     data,
	}

	if initialized {
		msg.Sender = config.Sender
	}
	return msg, nil
}

// Validate that all required data is present to assemble a sendable email.
func (e *Email) Validate() error {
	switch {
	case e.Subject == "":
		return ErrMissingSubject
	case e.Sender == "":
		return ErrMissingSender
	case len(e.To) == 0:
		return ErrMissingRecipient
	case e.Template == "":
		return ErrMissingTemplate
	}

	if _, err := mail.ParseAddress(e.Sender); err != nil {
		return fmt.Errorf("invalid sender email address %q: %w", e.Sender, ErrIncorrectEmail)
	}

	for _, to := range e.To {
		if _, err := mail.ParseAddress(to); err != nil {
			return fmt.Errorf("invalid recipient email address %q: %w", to, ErrIncorrectEmail)
		}
	}

	return nil
}

// Helper method to send an email using the emails.Send package function.
func (e *Email) Send() error {
	return Send(e)
}

// Return an email struct that can be sent via SMTP
func (e *Email) ToSMTP() (msg *email.Email, err error) {
	if err = e.Validate(); err != nil {
		return nil, err
	}

	msg = email.NewEmail()
	msg.From = e.Sender
	msg.To = e.To
	msg.Subject = e.Subject

	if msg.Text, msg.HTML, err = Render(e.Template, e.Data); err != nil {
		return nil, err
	}

	return msg, nil
}

// Return an email struct that can be sent via SendGrid
func (e *Email) ToSendGrid() (msg *sgmail.SGMailV3, err error) {
	if err = e.Validate(); err != nil {
		return nil, err
	}

	// See: https://github.com/sendgrid/sendgrid-go/blob/16f25e4d92886b2733473a19977ccf1aa625a89b/helpers/mail/mail_v3.go#L186-L195
	msg = new(sgmail.SGMailV3)
	msg.Subject = e.Subject
	msg.SetFrom(MustNewSGEmail(e.Sender))

	p := sgmail.NewPersonalization()
	p.AddTos(MustNewSGEmails(e.To)...)
	msg.AddPersonalizations(p)

	var (
		text string
		html string
	)

	if text, html, err = RenderString(e.Template, e.Data); err != nil {
		return nil, err
	}

	msg.AddContent(
		sgmail.NewContent("text/plain", text),
		sgmail.NewContent("text/html", html),
	)

	return msg, nil
}
