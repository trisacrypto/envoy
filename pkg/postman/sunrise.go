package postman

import (
	"database/sql"
	"time"

	"github.com/google/uuid"

	"github.com/trisacrypto/envoy/pkg/emails"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/envoy/pkg/sunrise"
	trisa "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
)

const defaultSunriseExpiration = 14 * 24 * time.Hour

type SunrisePacket struct {
	Packet
}

func SendSunrise(envelopeID uuid.UUID, payload *trisa.Payload) (packet *SunrisePacket, err error) {
	var parent *Packet
	if parent, err = Send(envelopeID, payload, trisa.TransferStarted); err != nil {
		return nil, err
	}

	packet = &SunrisePacket{
		Packet: *parent,
	}

	// Add parent to submessages
	packet.In.packet = &packet.Packet
	packet.Out.packet = &packet.Packet

	return packet, nil
}

func (s *SunrisePacket) Contacts() (contacts []*models.Contact, err error) {
	if contacts, err = s.Counterparty.Contacts(); err != nil {
		return nil, err
	}

	if len(contacts) == 0 {
		return nil, ErrNoContacts
	}

	return contacts, nil
}

func (s *SunrisePacket) SendEmail(contact *models.Contact, invite emails.SunriseInviteData) (err error) {
	// Create a sunrise record for database storage
	record := &models.Sunrise{
		EnvelopeID: uuid.MustParse(s.EnvelopeID()),
		Email:      contact.Email,
		Expiration: time.Now().Add(defaultSunriseExpiration),
		Status:     models.StatusDraft,
	}

	// Create the ID in the database of the sunrise record.
	if err = s.DB.CreateSunrise(record); err != nil {
		return err
	}

	// Create the HMAC verification token for the contact
	verification := sunrise.NewToken(record.ID, record.Expiration)

	// Sign the verification token
	if invite.Token, record.Signature, err = verification.Sign(); err != nil {
		return err
	}

	// Send the email to the contact
	var email *emails.Email
	if email, err = emails.NewSunriseInvite(contact.Address().String(), invite); err != nil {
		return err
	}

	if err = email.Send(); err != nil {
		return err
	}

	// Update the sunrise record in the database with the token and sent on timestamp
	record.SentOn = sql.NullTime{Valid: true, Time: time.Now()}
	record.Status = models.StatusPending

	if err = s.DB.UpdateSunrise(record); err != nil {
		return err
	}

	s.Log.Info().Str("email", contact.Email).Msg("sunrise verification token sent")
	return nil
}
