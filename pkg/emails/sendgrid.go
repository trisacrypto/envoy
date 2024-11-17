package emails

import (
	"net/mail"

	sgmail "github.com/sendgrid/sendgrid-go/helpers/mail"
)

func NewSGEmail(email string) (_ *sgmail.Email, err error) {
	var parsed *mail.Address
	if parsed, err = mail.ParseAddress(email); err != nil {
		return nil, err
	}
	return sgmail.NewEmail(parsed.Name, parsed.Address), nil
}

func NewSGEmails(emails []string) (out []*sgmail.Email, err error) {
	out = make([]*sgmail.Email, 0, len(emails))
	for _, email := range emails {
		var addr *sgmail.Email
		if addr, err = NewSGEmail(email); err != nil {
			return nil, err
		}
		out = append(out, addr)
	}
	return out, nil
}

func MustNewSGEmail(email string) *sgmail.Email {
	addr, err := NewSGEmail(email)
	if err != nil {
		panic(err)
	}
	return addr
}

func MustNewSGEmails(emails []string) []*sgmail.Email {
	addrs, err := NewSGEmails(emails)
	if err != nil {
		panic(err)
	}
	return addrs
}
