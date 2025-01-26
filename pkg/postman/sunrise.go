package postman

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/trisacrypto/envoy/pkg/emails"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/envoy/pkg/sunrise"
	trisa "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
	generic "github.com/trisacrypto/trisa/pkg/trisa/data/generic/v1beta1"
	"github.com/trisacrypto/trisa/pkg/trisa/envelope"
	"github.com/trisacrypto/trisa/pkg/trisa/keys"
)

const defaultSunriseExpiration = 14 * 24 * time.Hour

type SunrisePacket struct {
	Packet
	Messages []*generic.SunriseMessage
	payload  *trisa.Payload
}

// Initiates a Sunrise packet for handling interactions between secure envelopes and
// sending messages via email to compliance contacts that they have a new compliance
// message for review.
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

	// Keep track of the original payload
	packet.payload = payload
	return packet, nil
}

// Returns the email contacts of the compliance officers associated with the counterparty.
func (s *SunrisePacket) Contacts() (contacts []*models.Contact, err error) {
	if contacts, err = s.Counterparty.Contacts(); err != nil {
		return nil, err
	}

	if len(contacts) == 0 {
		return nil, ErrNoContacts
	}

	return contacts, nil
}

// Send a sunrise invitation email to the contact and create a verification token.
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
	invite.ContactName = contact.Name
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

	// Store the message info for the Sunrise "reply" message
	s.Messages = append(s.Messages, &generic.SunriseMessage{
		Recipient:      contact.Name,
		Email:          contact.Email,
		Channel:        "email",
		SentAt:         record.SentOn.Time.Format(time.RFC3339),
		ReplyNotBefore: record.Expiration.Format(time.RFC3339),
	})

	s.Log.Info().Str("email", contact.Email).Msg("sunrise verification token sent")
	return nil
}

// Creates the "reply" pending message for the sunrise messages that were sent.
func (s *SunrisePacket) Pending() (err error) {
	if len(s.Messages) == 0 {
		return ErrNoMessages
	}

	// Fetch the original payload from the outgoing message
	payload := &trisa.Payload{
		Identity:   s.payload.Identity,
		SentAt:     s.payload.SentAt,
		ReceivedAt: s.payload.ReceivedAt,
	}

	transaction := &generic.Sunrise{
		EnvelopeId:   s.EnvelopeID(),
		Counterparty: s.Counterparty.Name,
		Messages:     s.Messages,
	}

	transaction.Transaction = &generic.Transaction{}
	if err = s.payload.Transaction.UnmarshalTo(transaction.Transaction); err != nil {
		return fmt.Errorf("could not unmarshal original transaction: %w", err)
	}

	if payload.Transaction, err = anypb.New(transaction); err != nil {
		return fmt.Errorf("could not wrap sunrise transaction: %w", err)
	}

	opts := []envelope.Option{
		envelope.WithEnvelopeID(s.EnvelopeID()),
		envelope.WithTransferState(trisa.TransferPending),
	}

	if s.In.Envelope, err = envelope.New(payload, opts...); err != nil {
		return fmt.Errorf("could not create envelope for sunrise messages: %w", err)
	}

	return nil
}

// Encrypts the the "incoming" sunrise message for secure storage in the database with
// the specifrified storage key. and also passes this key to the "outgoing" secure
// envelope for secure encryption when that envlepe is turned into a model.
func (s *SunrisePacket) Seal(storageKey keys.PublicKey) (err error) {
	if storageKey == nil {
		return ErrNoSealingKey
	}

	// Ensure the outgoing message has the same encryption key as the incoming!
	s.Out.StorageKey = storageKey
	s.Out.SealingKey = storageKey
	if _, err = s.Out.Seal(); err != nil {
		return fmt.Errorf("could not encrypt outgoing envelope: %w", err)
	}

	// The sunrise incoming message must be encrypted for local storage in the database
	// since it is not encrypted by the "sender" (it's just a record of sent emails).
	if !s.In.Envelope.IsError() {
		if s.In.Envelope, _, err = s.In.Envelope.Encrypt(); err != nil {
			return fmt.Errorf("could not encrypt sunrise message: %w", err)
		}

		if s.In.Envelope, _, err = s.In.Envelope.Seal(envelope.WithSealingKey(storageKey)); err != nil {
			return fmt.Errorf("could not seal sunrise message: %w", err)
		}

		// Store the encrypted and sealed envelope as the "original" message, which will
		// be saved in the database when s.In.Model() is called.
		s.In.original = s.In.Envelope.Proto()
	}

	return nil
}

// This method creates the pending message to represent the "incoming" response; e.g. a
// record of the email messages that were sent and then encrypts both the outgoing and
// incoming messages using the storage key before saving the envelopes in the database.
// Finally, this method updates the transaction state and refreshes the local
// transaction so that the information is visible to the API.
func (s *SunrisePacket) Save(storageKey keys.PublicKey) (err error) {
	// Ensure that we have a counterparty
	if err = s.ResolveCounterparty(); err != nil {
		return err
	}

	// Add the counterparty to the transaction
	if err = s.DB.AddCounterparty(s.Counterparty); err != nil {
		return fmt.Errorf("could not associate counterparty with transaction: %w", err)
	}

	// Refresh the transaction to get the counterparty info before update.
	if err = s.RefreshTransaction(); err != nil {
		return err
	}

	// Add the transaction details from the payload of the outgoing message
	if payload := s.Out.Envelope.FindPayload(); payload != nil {
		s.Transaction = TransactionFromPayload(payload)
	}

	// Set transaction values
	s.Transaction.Source = models.SourceLocal
	s.Transaction.Status = models.StatusPending
	s.Transaction.LastUpdate = sql.NullTime{Valid: true, Time: time.Now()}

	// Update the transaction in the database
	if err = s.DB.Update(s.Transaction); err != nil {
		return fmt.Errorf("could not update transaction in database: %w", err)
	}

	// Create the incoming secure envelope with the sunrise message
	if err = s.Pending(); err != nil {
		return err
	}

	// Seal the incoming and outgoing messages with the storage key
	if err = s.Seal(storageKey); err != nil {
		return err
	}

	// Add envelopes to the database
	if err = s.DB.AddEnvelope(s.Out.Model()); err != nil {
		return fmt.Errorf("could not store outgoing sunrise message: %w", err)
	}

	if err = s.DB.AddEnvelope(s.In.Model()); err != nil {
		return fmt.Errorf("could not store incoming sunrise message: %w", err)
	}

	// Refresh to respond with the latest transaction info to the API request.
	if err = s.RefreshTransaction(); err != nil {
		return err
	}

	return nil
}
