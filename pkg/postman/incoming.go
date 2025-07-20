package postman

import (
	"database/sql"
	"encoding/base64"
	"fmt"

	"github.com/trisacrypto/envoy/pkg/enum"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/envoy/pkg/web/api/v1"
	"github.com/trisacrypto/envoy/pkg/webhook"
	"go.rtnl.ai/ulid"

	trisa "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
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
	original     *trisa.SecureEnvelope
}

// Returns the original protocol buffers that was wrapped by the incoming message.
func (i *Incoming) Proto() *trisa.SecureEnvelope {
	return i.original
}

// Returns the public key signature from the original message.
func (i *Incoming) PublicKeySignature() string {
	return i.original.PublicKeySignature
}

// Returns the original transfer state on the envelope.
func (i *Incoming) TransferState() trisa.TransferState {
	return i.original.TransferState
}

// Opens the incoming envelope, unsealing and decrypting it for handling.
func (i *Incoming) Open() (reject *trisa.Error, err error) {
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
			Direction:     enum.DirectionIncoming,
			Remote:        i.packet.Remote(),
			ReplyTo:       ulid.NullULID{},
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
		if i.packet.reply == enum.DirectionIncoming {
			i.model.ReplyTo = ulid.NullULID{
				Valid: true, ULID: i.packet.Out.Model().ID,
			}
		}
	}
	return i.model
}

// Updates the transaction info and status based on the incoming envelope.
func (i *Incoming) UpdateTransaction() (err error) {
	// Ensure that we have a counterparty
	if err = i.packet.ResolveCounterparty(); err != nil {
		return err
	}

	// If the transaction on the packet is empty, create a stub.
	if i.packet.Transaction == nil {
		i.packet.Transaction = &models.Transaction{}
	}

	// If the transaction is new and being created by the remote, add the counterparty.
	// Otherwise make sure it's the same counterparty or return an error.
	// TODO: Make sure it's the same counterparty or return an error
	if i.packet.DB.Created() && i.packet.request == enum.DirectionIncoming {
		//FIXME: COMPLETE AUDIT LOG
		if err = i.packet.DB.AddCounterparty(i.packet.Counterparty, &models.ComplianceAuditLog{}); err != nil {
			return fmt.Errorf("could not associate counterparty with transaction: %w", err)
		}

		// Also update the transaction source as remote if this is the request
		i.packet.Transaction.Source = enum.SourceRemote
	}

	// Update the status and last update on the transaction.
	timestamp, _ := i.Envelope.Timestamp()
	i.packet.Transaction.Status = i.StatusFromTransferState()
	i.packet.Transaction.LastUpdate = sql.NullTime{
		Valid: !timestamp.IsZero(), Time: timestamp,
	}

	// Update the transaction in the database
	//FIXME: COMPLETE AUDIT LOG
	if err = i.packet.DB.Update(i.packet.Transaction, &models.ComplianceAuditLog{}); err != nil {
		return fmt.Errorf("could not update transaction in database: %w", err)
	}

	return nil
}

// Creates a webhook callback request from the incoming envelope. Note that the packet
// must have the counterparty set and that the envelope UUID has been validated.
func (i *Incoming) WebhookRequest() *webhook.Request {
	request := &webhook.Request{
		Timestamp:     i.original.Timestamp,
		HMAC:          base64.RawURLEncoding.EncodeToString(i.original.Hmac),
		PKS:           i.original.PublicKeySignature,
		TransferState: i.original.TransferState.String(),
		Error:         i.original.Error,
		Payload:       nil,
	}

	// Ignore any errors: we expect that this has been validated already
	// TODO: configure the webhook to specify the encoding and format of the IVMS record
	request.TransactionID, _ = i.Envelope.UUID()
	request.Counterparty, _ = api.NewCounterparty(i.packet.Counterparty, nil)

	return request
}

// StatusFromTransferState determines what the status should be based on the incoming
// message transfer state. For example, if the incoming transfer state is accepted, then
// the Transfer can be marked as completed.
func (i *Incoming) StatusFromTransferState() enum.Status {
	switch ts := i.original.TransferState; ts {
	case trisa.TransferStateUnspecified:
		return enum.StatusUnspecified
	case trisa.TransferStarted:
		return enum.StatusReview
	case trisa.TransferPending:
		return enum.StatusPending
	case trisa.TransferReview:
		return enum.StatusReview
	case trisa.TransferRepair:
		return enum.StatusRepair
	case trisa.TransferAccepted:
		return enum.StatusAccepted
	case trisa.TransferCompleted:
		return enum.StatusCompleted
	case trisa.TransferRejected:
		return enum.StatusRejected
	default:
		panic(fmt.Errorf("unknown transfer state %s", ts.String()))
	}
}
