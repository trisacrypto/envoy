package postman

import (
	"context"
	"crypto/tls"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/trisa/pkg/openvasp/client"
	"github.com/trisacrypto/trisa/pkg/openvasp/trp/v3"
	trisa "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
	"github.com/trisacrypto/trisa/pkg/trisa/envelope"
	"github.com/trisacrypto/trisa/pkg/trisa/keys"
)

type TRPPacket struct {
	Packet
	info       *trp.Info
	mtls       *tls.ConnectionState
	payload    *trisa.Payload
	message    interface{}
	envelopeID uuid.UUID
}

func ReceiveTRPInquiry(inquiry *trp.Inquiry, mtls *tls.ConnectionState) (packet *TRPPacket, err error) {
	packet = &TRPPacket{
		Packet: Packet{
			In:      &Incoming{},
			Out:     &Outgoing{},
			request: DirectionIncoming,
			reply:   DirectionOutgoing,
		},
		info:    inquiry.Info,
		mtls:    mtls,
		message: inquiry,
	}

	// Make sure the info has a request identifier
	if inquiry.Info.RequestIdentifier == "" {
		return nil, ErrNoRequestIdentifier
	}

	// Make sure the request identifier is a parseable UUID for the database
	if packet.envelopeID, err = uuid.Parse(inquiry.Info.RequestIdentifier); err != nil {
		return nil, ErrInvalidUUID
	}

	// Add parent to submessages
	packet.In.packet = &packet.Packet
	packet.Out.packet = &packet.Packet

	// Create the payload from the inquiry
	if packet.payload, err = PayloadFromInquiry(inquiry); err != nil {
		return nil, err
	}

	// Create the incoming envelope
	opts := []envelope.Option{
		envelope.WithEnvelopeID(inquiry.Info.RequestIdentifier),
		envelope.WithTransferState(trisa.TransferStarted),
	}

	if packet.In.Envelope, err = envelope.New(packet.payload, opts...); err != nil {
		return nil, fmt.Errorf("could not create incoming trp inquiry envelope: %w", err)
	}

	packet.In.original = packet.In.Envelope.Proto()
	packet.Packet.resolver = packet
	return packet, nil
}

func (p *TRPPacket) Resolve(out *trp.Resolution) (err error) {
	// TODO: handle accepted and rejected envelopes better!
	var transferState trisa.TransferState
	switch {
	case out.Approved != nil:
		return errors.New("approved resolution not yet supported")
	case out.Rejected != "":
		return errors.New("rejected resolution not yet supported")
	default:
		transferState = trisa.TransferPending
	}

	// TODO: handle accept and reject envelopes!
	if p.Out.Envelope, err = p.In.Envelope.Update(p.payload, envelope.WithTransferState(transferState)); err != nil {
		p.Log.Debug().Err(err).Msg("could not prepare outgoing payload")
		return fmt.Errorf("could not create outgoing trp resolution envelope: %w", err)
	}
	return nil
}

func (p *TRPPacket) EnvelopeID() uuid.UUID {
	return p.envelopeID
}

func (p *TRPPacket) Payload() *trisa.Payload {
	return p.payload
}

func (p *TRPPacket) CommonName() string {
	if p.Counterparty != nil {
		return p.Counterparty.CommonName
	}
	return ""
}

// Encrypts an unencrypted incoming TRP inquiry, resolution, or confirmation message
// using the specified storage key to ensure secure envelopes are always encrypted in
// the database. This handles both the incoming and outgoing messages and should not
// be used with the secure-trisa-envelope extension is being used.
func (p *TRPPacket) Seal(storageKey keys.PublicKey) (err error) {
	if storageKey == nil {
		return ErrNoSealingKey
	}

	if !p.Out.Envelope.IsError() {
		// Ensure the outgoing message has the same encryption key as the incoming!
		p.Out.StorageKey = storageKey
		p.Out.SealingKey = storageKey

		if _, err = p.Out.Seal(); err != nil {
			return fmt.Errorf("could not encrypt outgoing envelope: %w", err)
		}
	}

	if !p.In.Envelope.IsError() {
		if p.In.Envelope, _, err = p.In.Envelope.Encrypt(); err != nil {
			return fmt.Errorf("could not encrypt trp message: %w", err)
		}

		if p.In.Envelope, _, err = p.In.Envelope.Seal(envelope.WithSealingKey(storageKey)); err != nil {
			return fmt.Errorf("could not seal trp message: %w", err)
		}

		// Store the encrypted and sealed envelope as the "original" message, which will
		// be saved in the database when s.In.Model() is called.
		p.In.original = p.In.Envelope.Proto()
	}

	return nil
}

// Returns the remote information for storage in the database using the underlying
// resolver if available, otherwise a NULL string is returned.
func (p *TRPPacket) Remote() sql.NullString {
	commonName := p.CommonName()
	return sql.NullString{Valid: commonName != "", String: commonName}
}

// Resolve counterparty through the following methods:
//
// 1. Try to lookup the counterparty via the mTLS hostname
// 2. Try to lookup the counterparty using the callback hostname
// 3. Try to lookup the counterparty using the originator name
// 4. Try to lookup the counterparty using the beneficiary name
//
// If a hostname is available, perform an identity lookup.
// Note: name matches must be exact; they are not fuzzy searches.
func (p *TRPPacket) ResolveCounterparty() (err error) {
	// Attempt to resolve the counterparty from the incoming mTLS connection
	if p.mtls != nil {
		if len(p.mtls.PeerCertificates) > 0 {
			cert := p.mtls.PeerCertificates[0]
			hostnames := make([]string, 0, len(cert.DNSNames)+1)
			hostnames = append(hostnames, cert.Subject.CommonName)
			hostnames = append(hostnames, cert.DNSNames...)

			if p.Counterparty, err = p.resolveCounterpartyHostname(hostnames...); err == nil {
				return nil
			}
		}
	}

	if p.payload != nil {
		if inquiry, ok := p.message.(*trp.Inquiry); ok {
			// Attempt to resolve the counterparty from the callback hostname
			if callback := inquiry.Callback; callback != "" {
				if uri, err := url.Parse(callback); err == nil {
					if p.Counterparty, err = p.resolveCounterpartyHostname(uri.Host); err == nil {
						return nil
					}
				}
			}

			// TODO: Attempt to resolve the counterparty from the originator name

			// TODO: Attempt to resolve the counterparty from the beneficiary name
		}
	}

	// If we get to this point and we were unable to resolve a counterparty then only
	// return an error if this transaction is being created (a counterparty is required
	// to associate with the tranaction). Otherwise return no error in the case of an
	// update because of resolution or confirmation.
	if p.DB.Created() {
		return ErrNoCounterpartyInfo
	}
	return nil
}

func (p *TRPPacket) resolveCounterpartyHostname(hostnames ...string) (counterparty *models.Counterparty, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var trpClient *client.Client
	if trpClient, err = client.New(); err != nil {
		return nil, err
	}

	for _, host := range hostnames {
		uri := url.URL{Scheme: "https", Host: host}

		// Attempt to lookup the counterparty in the database by common name
		if counterparty, err = p.DB.LookupCounterparty(models.FieldCommonName, uri.Hostname()); err == nil {
			return counterparty, nil
		} else {
			p.Log.Debug().Err(err).Str("hostname", uri.Hostname()).Msg("counterparty not found by hostname")
		}

		// Attempt to lookup the counterparty identity from the host
		var identity *trp.Identity
		if identity, err = trpClient.Identity(ctx, uri.String()); err != nil {
			p.Log.Debug().Err(err).Str("host", host).Msg("could not resolve counterparty identity from host")
			continue
		}

		// Lookup identity in database by LEI
		// This occurs when the hostname is different than the common name stored in the database
		if identity.LEI != "" {
			if counterparty, err = p.DB.LookupCounterparty(models.FieldLEI, identity.LEI); err == nil {
				return counterparty, nil
			} else {
				p.Log.Debug().Err(err).Str("lei", identity.LEI).Msg("counterparty not found by lei")
			}
		}

		// If the identity lookup was successful: create a new counterparty from the returned identity
		p.Log.Debug().Str("host", host).Str("lei", identity.LEI).Str("name", identity.Name).Msg("trp counterparty identity resolved from peer hostname")
		return &models.Counterparty{
			Source:     models.SourcePeer,
			Protocol:   models.ProtocolTRP,
			Endpoint:   uri.String(),
			CommonName: uri.Hostname(),
			Name:       identity.Name,
			LEI:        sql.NullString{Valid: identity.LEI != "", String: identity.LEI},
			Country:    sql.NullString{Valid: false},
		}, nil
	}

	return nil, ErrCounterpartyNotFound
}
