package postman

import (
	"context"
	"crypto/tls"
	"database/sql"
	"fmt"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/trisa/pkg/openvasp/client"
	"github.com/trisacrypto/trisa/pkg/openvasp/trp/v3"
	trisa "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
	generic "github.com/trisacrypto/trisa/pkg/trisa/data/generic/v1beta1"
	"github.com/trisacrypto/trisa/pkg/trisa/envelope"
	"google.golang.org/protobuf/types/known/anypb"
)

type TRPPacket struct {
	Packet
	info    *trp.Info
	mtls    *tls.ConnectionState
	payload interface{}
}

func ReceiveTRPInquiry(inquiry *trp.Inquiry, mtls *tls.ConnectionState) (packet *TRPPacket, err error) {
	// Make sure the info has a request identifier
	if inquiry.Info.RequestIdentifier == "" {
		return nil, ErrNoRequestIdentifier
	}

	// Make sure the request identifier is a parseable UUID for the database
	if _, err = uuid.Parse(inquiry.Info.RequestIdentifier); err != nil {
		return nil, ErrInvalidUUID
	}

	packet = &TRPPacket{
		Packet: Packet{
			In:      &Incoming{},
			Out:     &Outgoing{},
			request: DirectionIncoming,
			reply:   DirectionOutgoing,
		},
		info:    inquiry.Info,
		mtls:    mtls,
		payload: inquiry,
	}

	// Add parent to submessages
	packet.In.packet = &packet.Packet
	packet.Out.packet = &packet.Packet

	// Create the payload from the inquiry
	// TODO: push this code back into the trp package
	var payload *trisa.Payload
	if payload, err = payloadFromInquiry(inquiry); err != nil {
		return nil, err
	}

	// Create the incoming envelope
	opts := []envelope.Option{
		envelope.WithEnvelopeID(inquiry.Info.RequestIdentifier),
		envelope.WithTransferState(trisa.TransferStarted),
	}

	if packet.In.Envelope, err = envelope.New(payload, opts...); err != nil {
		return nil, fmt.Errorf("could not create incoming trp inquiry envelope: %w", err)
	}

	packet.In.original = packet.In.Envelope.Proto()
	packet.Packet.resolver = packet
	return packet, nil
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
		if inquiry, ok := p.payload.(*trp.Inquiry); ok {
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

// TODO: push this code to the TRP package
func payloadFromInquiry(inquiry *trp.Inquiry) (payload *trisa.Payload, err error) {
	payload = &trisa.Payload{
		SentAt: time.Now().Format(time.RFC3339), // The TRP inquiry is the first message, so sent at is now.
	}

	if payload.Identity, err = anypb.New(inquiry.IVMS101); err != nil {
		return nil, err
	}

	transaction := &generic.TRP{}

	if payload.Transaction, err = anypb.New(transaction); err != nil {
		return nil, err
	}

	return payload, nil
}
