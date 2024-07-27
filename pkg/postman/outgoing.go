package postman

import (
	"database/sql"

	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/envoy/pkg/ulids"

	api "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
	"github.com/trisacrypto/trisa/pkg/trisa/envelope"
	"github.com/trisacrypto/trisa/pkg/trisa/keys"
)

// Outgoing messages are sent to the remote party, either replies to transfers coming
// into the TRISA or TRP server or original messages if the transfer is initiated by
// a user or the API.
//
// Outgoing messages imply that they need to be resealed for decryption by the local host.
type Outgoing struct {
	Envelope   *envelope.Envelope
	StorageKey keys.PublicKey
	SealingKey keys.PublicKey
	packet     *Packet
	model      *models.SecureEnvelope
}

func (o *Outgoing) Proto() *api.SecureEnvelope {
	return o.Envelope.Proto()
}

func (o *Outgoing) PublicKeySignature() string {
	pks, _ := o.SealingKey.PublicKeySignature()
	return pks
}

func (o *Outgoing) Seal() (reject *api.Error, err error) {
	if o.SealingKey == nil {
		return nil, ErrNoSealingKey
	}

	if o.Envelope, reject, err = o.Envelope.Encrypt(); err != nil {
		if reject != nil {
			return reject, err
		}

		o.packet.Log.Error().Err(err).Str("pks", o.PublicKeySignature()).Msg("could not encrypt outgoing secure envelope")
		return nil, err
	}

	if o.Envelope, reject, err = o.Envelope.Seal(envelope.WithSealingKey(o.SealingKey)); err != nil {
		if reject != nil {
			return reject, err
		}

		o.packet.Log.Error().Err(err).Str("pks", o.PublicKeySignature()).Msg("could not seal outgoing secure envelope")
		return nil, err
	}

	return nil, nil
}

func (o *Outgoing) Model() *models.SecureEnvelope {
	if o.model == nil {
		se := o.Proto()
		o.model = &models.SecureEnvelope{
			Direction:     models.DirectionOutgoing,
			Remote:        sql.NullString{Valid: o.packet.PeerInfo.CommonName != "", String: o.packet.PeerInfo.CommonName},
			ReplyTo:       ulids.NullULID{},
			IsError:       o.Envelope.IsError(),
			EncryptionKey: nil,
			HMACSecret:    nil,
			ValidHMAC:     sql.NullBool{},
			PublicKey:     sql.NullString{},
			TransferState: int32(se.TransferState),
			Envelope:      se,
		}

		if !o.model.IsError {
			// Handle cryptography if this is an encrypted envelope
			if o.StorageKey == nil {
				panic("cannot encrypt outgoing envelope without storage key")
			}

			crypto := o.Envelope.Crypto()
			if crypto == nil {
				panic("cannot encrypt outgoing envelope without crypto on envelope")
			}

			if err := o.model.Reseal(o.StorageKey, crypto); err != nil {
				o.packet.Log.Error().Err(err).Msg("could not reseal outgoing envelope for secure storage")
			}

			// Validate the HMAC but only store if its valid or not
			o.model.ValidHMAC.Bool, _ = o.Envelope.ValidateHMAC()
			o.model.ValidHMAC.Valid = true
		}

		o.model.EnvelopeID, _ = o.Envelope.UUID()
		o.model.Timestamp, _ = o.Envelope.Timestamp()

		// This assumes that the incoming model has already been created!
		if o.packet.Reply == DirectionOutgoing {
			o.model.ReplyTo = ulids.NullULID{
				Valid: true, ULID: o.packet.In.Model().ID,
			}
		}
	}
	return o.model
}
