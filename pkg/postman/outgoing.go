package postman

import (
	"crypto/rsa"
	"database/sql"
	"fmt"

	"github.com/trisacrypto/envoy/pkg/enum"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"go.rtnl.ai/ulid"

	api "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
	"github.com/trisacrypto/trisa/pkg/trisa/crypto"
	"github.com/trisacrypto/trisa/pkg/trisa/crypto/rsaoeap"
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

// Returns the protocol buffers of the prepared envelope (seal first!)
func (o *Outgoing) Proto() *api.SecureEnvelope {
	return o.Envelope.Proto()
}

// Returns the public key signature from the sealing key to send to the remote.
func (o *Outgoing) PublicKeySignature() string {
	pks, _ := o.SealingKey.PublicKeySignature()
	return pks
}

// Seals the outgoing envelope in preparation for sending or returning to remote.
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

// Creates a secure envelope model to store in the database with all of the information
// that is is in the envelope and in the packet. Note that the PeerInfo is expected to
// be on the packet; and if this is an outgoing reply to an incoming transaction, the
// incoming model must already have been created and have an ID.
//
// This method will Reseal the envelope if it is not an error envelope, encrypting it
// for local storage and requiring the StorageKey in order to reseal.
func (o *Outgoing) Model() *models.SecureEnvelope {
	if o.model == nil {
		se := o.Proto()
		o.model = &models.SecureEnvelope{
			Direction:     enum.DirectionOutgoing,
			Remote:        o.packet.Remote(),
			ReplyTo:       ulid.NullULID{},
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

			if err := o.reseal(o.StorageKey, crypto); err != nil {
				o.packet.Log.Error().Err(err).Msg("could not reseal outgoing envelope for secure storage")
			}

			// Validate the HMAC but only store if its valid or not
			o.model.ValidHMAC.Bool, _ = o.Envelope.ValidateHMAC()
			o.model.ValidHMAC.Valid = true
		}

		o.model.EnvelopeID, _ = o.Envelope.UUID()
		o.model.Timestamp, _ = o.Envelope.Timestamp()

		// This assumes that the incoming model has already been created!
		if o.packet.reply == enum.DirectionOutgoing {
			o.model.ReplyTo = ulid.NullULID{
				Valid: true, ULID: o.packet.In.Model().ID,
			}
		}
	}
	return o.model
}

// Updates the transaction info and status based on the outgoing envelope.
func (o *Outgoing) UpdateTransaction() (err error) {
	// Ensure that we have a counterparty
	if err = o.packet.ResolveCounterparty(); err != nil {
		return err
	}

	// If the transaction on the packet is empty, create a stub.
	if o.packet.Transaction == nil {
		o.packet.Transaction = &models.Transaction{}
	}

	// If the transaction is new and being created by the local node add the
	// counterparty and source; otherwise make sure the same counterparty is involved.
	// TODO: make sure it's the same counterparty or return an error.
	if o.packet.DB.Created() && o.packet.request == enum.DirectionOutgoing {
		// Add the transaction details from the payload of the outgoing message
		if payload := o.Envelope.FindPayload(); payload != nil {
			o.packet.Transaction = TransactionFromPayload(payload)
		}

		if err = o.packet.DB.AddCounterparty(o.packet.Counterparty, &models.ComplianceAuditLog{
			ChangeNotes: sql.NullString{Valid: true, String: "Outgoing.UpdateTransaction()"},
		}); err != nil {
			return fmt.Errorf("could not associate counterparty with transaction: %w", err)
		}

		// Also update the transaction source as local if this is the request
		o.packet.Transaction.Source = enum.SourceLocal
	}

	// Update the status and last update on the transaction
	timestamp, _ := o.Envelope.Timestamp()
	o.packet.Transaction.Status = o.StatusFromTransferState()
	o.packet.Transaction.LastUpdate = sql.NullTime{
		Valid: !timestamp.IsZero(), Time: timestamp,
	}

	// Update the transaction in the database
	if err = o.packet.DB.Update(o.packet.Transaction, &models.ComplianceAuditLog{
		ChangeNotes: sql.NullString{Valid: true, String: "Outgoing.UpdateTransaction()"},
	}); err != nil {
		return fmt.Errorf("could not update transaction in database: %w", err)
	}

	return nil
}

// Reseal the envelope with the specified storage key and cryptography.
func (o *Outgoing) reseal(storageKey keys.PublicKey, sec crypto.Crypto) (err error) {
	// Set the public key signature of the storage key on the model
	if o.model.PublicKey.String, err = storageKey.PublicKeySignature(); err != nil {
		return err
	}

	// Ensure the null value is set to valid
	o.model.PublicKey.Valid = o.model.PublicKey.String != ""

	// Create a cipher to seal the new storage keys
	var (
		pubkey interface{}
		seal   crypto.Cipher
	)

	if pubkey, err = storageKey.SealingKey(); err != nil {
		return err
	}

	switch t := pubkey.(type) {
	case *rsa.PublicKey:
		if seal, err = rsaoeap.New(t); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown key type %T", t)
	}

	// Encrypt the encryption key and hmac secret with the new cipher
	if o.model.EncryptionKey, err = seal.Encrypt(sec.EncryptionKey()); err != nil {
		return err
	}

	if o.model.HMACSecret, err = seal.Encrypt(sec.HMACSecret()); err != nil {
		return err
	}

	return nil
}

// StatusFromTransferState determines what the status should be based on the outgoing
// message transfer state. For example, if the outgoing transfer state is pending, then
// the Transfer should be marked as needing review.
func (o *Outgoing) StatusFromTransferState() enum.Status {
	if o.packet.reply == enum.DirectionOutgoing {
		// If this is a reply to an incoming packet and the response is Pending, then
		// we need to determine what state the incoming message was in to determine
		// what pending action needs to be completed.
		switch ts := o.Envelope.TransferState(); ts {

		// If we sent back pending then we're either in review or repair mode,
		// depending on what the incoming transfer state was (e.g. the request)
		case api.TransferPending:
			switch ints := o.packet.In.original.TransferState; ints {

			// If we were sent started or review, then review action needs to be taken.
			// Also in the case of an unspecified transfer state, we will review.
			case api.TransferStateUnspecified, api.TransferStarted, api.TransferReview:
				return enum.StatusReview

			// If we were sent a repair request, then we need to make a repair.
			case api.TransferRepair:
				return enum.StatusRepair

			// If the incoming message is pending, we should have sent back review.
			// If the incoming messages is accepted, completed, or rejected, we should
			// have echoed those states back to the sender, not pending.
			default:
				panic(fmt.Errorf("unhandled incoming transfer state %q to determine transaction status after pending reply", ints.String()))
			}

		// If we're sending review or repair, we're waiting for the recipient to take action
		case api.TransferReview, api.TransferRepair:
			return enum.StatusPending

		// These are the echo back statuses; we assume the incoming message matches.
		case api.TransferAccepted:
			return enum.StatusAccepted
		case api.TransferCompleted:
			return enum.StatusCompleted
		case api.TransferRejected:
			return enum.StatusRejected
		default:
			panic(fmt.Errorf("unhandled outgoing transfer state %q for reply to incoming message", ts.String()))
		}
	}

	// These states will likely be temporarily set on the transaction as a request, but
	// then will be overridden depending on the state of the incoming reply to our message.
	switch ts := o.Envelope.TransferState(); ts {
	case api.TransferStateUnspecified:
		return enum.StatusUnspecified
	case api.TransferStarted:
		return enum.StatusDraft
	case api.TransferPending:
		return enum.StatusReview
	case api.TransferReview:
		return enum.StatusPending
	case api.TransferRepair:
		return enum.StatusPending
	case api.TransferAccepted:
		return enum.StatusAccepted
	case api.TransferCompleted:
		return enum.StatusCompleted
	case api.TransferRejected:
		return enum.StatusRejected
	default:
		panic(fmt.Errorf("unknown transfer state %s", ts.String()))
	}
}
