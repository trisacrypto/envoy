package postman

import (
	"context"
	"crypto/tls"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/trisa/pkg/openvasp/client"
	"github.com/trisacrypto/trisa/pkg/openvasp/trp/v3"
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

			// Attempt to resolve the counterparty from the originator name

			// Attempt to resolve the counterparty from the beneficiary name
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

func (p *TRPPacket) resolveCounterpartyHostname(hostnames ...string) (_ *models.Counterparty, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var trpClient *client.Client
	if trpClient, err = client.New(); err != nil {
		return nil, err
	}

	for _, host := range hostnames {
		uri := url.URL{Scheme: "https", Host: host}

		var identity *trp.Identity
		if identity, err = trpClient.Identity(ctx, uri.String()); err != nil {
			p.Log.Debug().Err(err).Str("host", host).Msg("could not resolve counterparty identity from host")
			continue
		}

		// TODO: look up identity in database
		if identity.LEI != "" {
			p.Log.Info().Str("lei", identity.LEI).Msg("resolved counterparty identity from host")
		}
		// if p.Counterparty, err = p.DB.FindCounterpartyByName(identity.Name); err != nil {
		// }

	}

	return nil, nil
}
