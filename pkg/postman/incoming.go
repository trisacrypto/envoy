package postman

import (
	"database/sql"
	"fmt"

	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/envoy/pkg/ulids"

	api "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
	"github.com/trisacrypto/trisa/pkg/trisa/envelope"
	"github.com/trisacrypto/trisa/pkg/trisa/keys"
)

// Incoming messages are received from the remote; they can be replies to transfers
// initiated by the local node or they can be incoming messages that require a reply
// from the local node (e.g. received by the TRISA or TRP servers).
//
// Incoming messages imply that they are sealed for decryption by the local host.
type Incoming struct {
	Envelope     *envelope.Envelope
	UnsealingKey keys.PrivateKey
	packet       *Packet
	model        *models.SecureEnvelope
	original     *api.SecureEnvelope
}

// Returns the original protocol buffers that was wrapped by the incoming message.
func (i *Incoming) Proto() *api.SecureEnvelope {
	return i.original
}

// Returns the public key signature from the original message
func (i *Incoming) PublicKeySignature() string {
	return i.original.PublicKeySignature
}

func (i *Incoming) Open() (reject *api.Error, err error) {
	if i.UnsealingKey == nil {
		return nil, ErrNoUnsealingKey
	}

	if i.Envelope, reject, err = i.Envelope.Unseal(envelope.WithUnsealingKey(i.UnsealingKey)); err != nil {
		if reject != nil {
			return reject, err
		}

		i.packet.Log.Error().Err(err).Str("pks", i.PublicKeySignature()).Msg("could not unseal incoming secure envelope")
		return nil, err
	}

	if i.Envelope, reject, err = i.Envelope.Decrypt(); err != nil {
		if reject != nil {
			return reject, err
		}

		i.packet.Log.Error().Err(err).Str("pks", i.PublicKeySignature()).Msg("could not decrypt incoming secure envelope")
		return nil, err
	}

	return nil, nil
}

// Creates a secure envelope model to store in the database with all of the information
// that is in the envelope and in the packet. Note that the PeerInfo is expected to be
// on the packet; and if this is an incoming reply to an outgoing transaction, the
// outgoing model must already have been created and have an ID.
//
// TODO: we need to store public key information about the key that was actually used
// to decrypt the model, so that we can decrypt the model in the future.
func (i *Incoming) Model() *models.SecureEnvelope {
	if i.model == nil {
		// Create the incoming secure envelope model
		i.model = &models.SecureEnvelope{
			Direction:     models.DirectionIncoming,
			Remote:        sql.NullString{Valid: i.packet.PeerInfo.CommonName != "", String: i.packet.PeerInfo.CommonName},
			ReplyTo:       ulids.NullULID{},
			IsError:       i.Envelope.IsError(),
			EncryptionKey: i.original.EncryptionKey,
			HMACSecret:    i.original.HmacSecret,
			ValidHMAC:     sql.NullBool{},
			PublicKey:     sql.NullString{Valid: i.original.PublicKeySignature != "", String: i.original.PublicKeySignature},
			TransferState: int32(i.original.TransferState),
			Envelope:      i.original,
		}

		if !i.model.IsError {
			// Validate the HMAC but only store if its valid or not
			i.model.ValidHMAC.Bool, _ = i.Envelope.ValidateHMAC()
			i.model.ValidHMAC.Valid = true
		}

		i.model.EnvelopeID, _ = i.Envelope.UUID()
		i.model.Timestamp, _ = i.Envelope.Timestamp()

		// This assumes that the outgoing model has already been created!
		if i.packet.Reply == DirectionIncoming {
			i.model.ReplyTo = ulids.NullULID{
				Valid: true, ULID: i.packet.Out.Model().ID,
			}
		}
	}
	return i.model
}

func (i *Incoming) UpdateTransaction() (err error) {
	// Ensure that we have a counterparty
	if err = i.packet.ResolveCounterparty(); err != nil {
		return err
	}

	// If the transaction on the packet is empty, create a stub; though this indicates
	// that the incoming message may not have been propertly instantiated.
	if i.packet.Transaction == nil {
		i.packet.Transaction = &models.Transaction{}
	}

	// If the transaction is new and being created by the remote, add the counterparty.
	// Otherwise make sure it's the same counterparty or return an error.
	// TODO: Make sure it's the same counterparty or return an error
	if i.packet.DB.Created() && i.packet.Request == DirectionIncoming {
		if err = i.packet.DB.AddCounterparty(i.packet.Counterparty); err != nil {
			return fmt.Errorf("could not associate counterparty with transaction: %w", err)
		}

		// Also update the transaction source as remote if this is the request
		i.packet.Transaction.Source = models.SourceRemote
	}

	// Update the status, last update, and source if necessary
	timestamp, _ := i.Envelope.Timestamp()
	i.packet.Transaction.Status = models.StatusFromTransferState(i.original.TransferState)
	i.packet.Transaction.LastUpdate = sql.NullTime{
		Valid: !timestamp.IsZero(), Time: timestamp,
	}

	// Update the transaction in the database
	if err = i.packet.DB.Update(i.packet.Transaction); err != nil {
		return fmt.Errorf("could not update transaction in database: %w", err)
	}

	return nil
}
