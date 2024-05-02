package trisa

import (
	"context"
	"crypto/rsa"
	"database/sql"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/trisacrypto/envoy/pkg/logger"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/envoy/pkg/trisa/peers"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/trisacrypto/trisa/pkg/ivms101"
	api "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
	"github.com/trisacrypto/trisa/pkg/trisa/crypto"
	"github.com/trisacrypto/trisa/pkg/trisa/crypto/rsaoeap"
	generic "github.com/trisacrypto/trisa/pkg/trisa/data/generic/v1beta1"
	"github.com/trisacrypto/trisa/pkg/trisa/envelope"
	"github.com/trisacrypto/trisa/pkg/trisa/keys"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/anypb"
)

var internalError = status.Error(codes.Internal, "unable to process secure envelope")

//===========================================================================
// Transfer and TransferStream gRPC Handlers
//===========================================================================

func (s *Server) Transfer(ctx context.Context, in *api.SecureEnvelope) (_ *api.SecureEnvelope, err error) {
	// Add tracing from context to the log context.
	log := logger.Tracing(ctx)

	// Identify the counterparty peer from the context.
	var peer peers.Peer
	if peer, err = s.network.FromContext(ctx); err != nil {
		log.Error().Err(err).Msg("could not identify peer from context")
		return nil, status.Error(codes.Unauthenticated, "could not identify remote peer from mTLS certificates")
	}

	// Add peer to the log context and log message.
	log = log.With().Str("peer", peer.String()).Str("envelope_id", in.Id).Logger()
	log.Debug().Msg("trisa transfer received")

	// Handle the incoming transfer
	var outgoing *Outgoing
	incoming := NewIncoming(ctx, peer, in, log)
	if outgoing, err = s.HandleIncoming(incoming); err != nil {
		log.Warn().Err(err).Str("envelope_id", in.Id).Msg("trisa transfer handler failed")
		return nil, err
	}

	log.Debug().Msg("trisa transfer completed")
	return outgoing.env.Proto(), nil
}

// The number of incoming secure envelopes to buffer for handling.
const transferBuffer = 8

func (s *Server) TransferStream(stream api.TRISANetwork_TransferStreamServer) (err error) {
	// Get the stream context for us in handling streaming transfers
	ctx := stream.Context()
	log := logger.Tracing(ctx)

	// Identify the counterparty peer from the context.
	var peer peers.Peer
	if peer, err = s.network.FromContext(ctx); err != nil {
		log.Error().Err(err).Msg("could not identify peer from context")
		return status.Error(codes.Unauthenticated, "could not identify remote peer from mTLS certificates")
	}

	// Add peer to the log context and log message.
	log = log.With().Str("peer", peer.String()).Logger()
	log.Info().Msg("transfer stream opened")

	// Create go routine context
	outgoing := make(chan *api.SecureEnvelope, transferBuffer)
	wg := sync.WaitGroup{}
	wg.Add(2)

	// Receive incoming secure envelopes from the remote client.
	go func(outgoing chan<- *api.SecureEnvelope) {
		defer wg.Done()
		defer close(outgoing)

		for {
			// Check if the context has been closed.
			select {
			case <-ctx.Done():
				if err := ctx.Err(); err != nil {
					log.Debug().Err(err).Msg("context closed")
				}
				return
			default:
			}

			var in *api.SecureEnvelope
			if in, err = stream.Recv(); err != nil {
				if streamClosed(err) {
					log.Debug().Msg("transfer stream closed by client")
					err = nil
					return
				}

				// Set the error message to aborted if we cannot recv a message
				err = status.Error(codes.Aborted, "could not recv event from client")
				log.Warn().Err(err).Msg("transfer stream crashed")
				return
			}

			// Handle the incoming transfer
			incoming := NewIncoming(ctx, peer, in, log.With().Str("envelope_id", in.Id).Logger())

			var out *Outgoing
			if out, err = s.HandleIncoming(incoming); err != nil {
				log.Warn().Err(err).Str("envelope_id", in.Id).Msg("unable to handle transfer request, stream closing")
				return
			}

			// Queue the message to be sent on the outgoing channel
			outgoing <- out.env.Proto()
		}

	}(outgoing)

	// Send outgoing secure envelopes back to the remote client
	go func(outgoing <-chan *api.SecureEnvelope) {
		// Declare an error variable at the top level to ensure that the err managed
		// by the recv stream is not accidentally shadowed by this routine.
		// This prevents race conditions and incorrect return errors.
		var err error
		defer wg.Done()

		for out := range outgoing {
			if err = stream.Send(out); err != nil {
				log.Warn().Err(err).Str("envelope_id", out.Id).Msg("could not send secure envelope back to client")
			}
		}
	}(outgoing)

	// Wait for go routines to close and all remaining transfers handled.
	wg.Wait()

	log.Info().Msg("transfer stream closed")
	return err
}

//===========================================================================
// Incoming Message Definition
//===========================================================================

// Incoming stores the full context of an incoming transfer message for handling.
type Incoming struct {
	ctx  context.Context            // Context of the request
	peer peers.Peer                 // The peer the message was received from
	env  *envelope.Envelope         // NOTE: this value should not be replaced, only cloned
	log  zerolog.Logger             // Logger updated with RPC details
	db   models.PreparedTransaction // Access to the database model
	hmac sql.NullBool               // Helper to set validated hmac information on
}

func NewIncoming(ctx context.Context, peer peers.Peer, env *api.SecureEnvelope, log zerolog.Logger) *Incoming {
	incoming := &Incoming{
		ctx:  ctx,
		peer: peer,
		log:  log,
	}

	var err error
	if incoming.env, err = envelope.Wrap(env); err != nil {
		panic(fmt.Errorf("could not wrap incoming secure envelope: %w", err))
	}

	return incoming
}

// Helper for retrieving the envelope ID directly from the envelope.
func (i *Incoming) ID() string {
	return i.env.ID()
}

// Helper for retrieving the envelope public key signature if available
func (i *Incoming) PublicKeySignature() string {
	return i.env.Proto().PublicKeySignature
}

// Mark the incoming HMAC as validated (or not).
func (i *Incoming) SetHMACValid(valid bool) {
	i.hmac = sql.NullBool{Valid: true, Bool: valid}
}

// Create a rejection envelope from the incoming envelope
func (i *Incoming) Reject(code api.Error_Code, message string, retry bool) (*Outgoing, error) {
	reject := &api.Error{
		Code:    code,
		Message: message,
		Retry:   retry,
	}
	return i.Error(reject)
}

// Create a rejection envelope from the api error
func (i *Incoming) Error(reject *api.Error) (out *Outgoing, err error) {
	var msg *api.SecureEnvelope
	if msg, err = envelope.Reject(reject, envelope.WithEnvelopeID(i.ID())); err != nil {
		log.Error().Err(err).Msg("could not prepare rejection envelope")
		return nil, status.Error(codes.Internal, "could not complete TRISA transfer")
	}

	if out, err = i.Outgoing(msg); err != nil {
		return nil, err
	}

	i.log.Info().
		Str("code", reject.Code.String()).
		Str("message", reject.Message).
		Bool("retry", reject.Retry).
		Msg("trisa transfer rejected")

	return out, nil
}

// Create an outgoing envelope associated with the incoming envelope
func (i *Incoming) Outgoing(msg *api.SecureEnvelope) (out *Outgoing, err error) {
	var env *envelope.Envelope
	if env, err = envelope.Wrap(msg); err != nil {
		log.Error().Err(err).Msg("could not prepare rejection envelope")
		return nil, status.Error(codes.Internal, "could not complete TRISA transfer")
	}
	return &Outgoing{env: env, log: i.log}, nil
}

// Converts the incoming message into a database model for storage. This method assumes
// that the envelopeID has already been parsed as a uuid and panics if the envelopeID is
// not a uuid. Since this is an incoming message, the encryption key and hmac secret are
// assumed to be sealed using a public key of the local TRISA node, identified by the
// public key signature.
func (i *Incoming) Model() *models.SecureEnvelope {
	// Create the incoming secure envelope model
	se := i.env.Proto()
	model := &models.SecureEnvelope{
		Direction:     models.DirectionIncoming,
		IsError:       i.env.IsError(),
		EncryptionKey: se.EncryptionKey,
		HMACSecret:    se.HmacSecret,
		ValidHMAC:     i.hmac,
		PublicKey:     sql.NullString{Valid: se.PublicKeySignature != "", String: se.PublicKeySignature},
		Envelope:      se,
	}

	model.EnvelopeID, _ = i.env.UUID()
	model.Timestamp, _ = i.env.Timestamp()
	return model
}

// Helper method to update the transaction with the incoming record details
func (i *Incoming) UpdateRecord() (err error) {
	// Update counterparty information from remote peer info (e.g. from the certificates)
	var info *peers.Info
	if info, err = i.peer.Info(); err != nil {
		return fmt.Errorf("could not identify counterparty in transaction: %w", err)
	}

	// If the transaction is new and is being created, add the counterparty
	// TODO: Make sure it's the same counterparty or return an error
	if i.db.Created() {
		if err = i.db.AddCounterparty(info.Model()); err != nil {
			return fmt.Errorf("could not identify or create counterparty in database: %w", err)
		}
	}

	timestamp, _ := i.env.Timestamp()

	// Update the transaction with available information
	// TODO: identify if this is a completed transaction
	transaction := &models.Transaction{
		Status:     models.StatusPending,
		LastUpdate: sql.NullTime{Valid: true, Time: timestamp},
	}

	if i.db.Created() {
		transaction.Source = models.SourceRemote
	}

	if err = i.db.Update(transaction); err != nil {
		return fmt.Errorf("could not update transaction in database: %w", err)
	}

	// Save the incoming envelope to disk
	if err = i.db.AddEnvelope(i.Model()); err != nil {
		return fmt.Errorf("could not store incoming secure envelope in database: %w", err)
	}

	return nil
}

//===========================================================================
// Outgoing Message Definition
//===========================================================================

type Outgoing struct {
	log        zerolog.Logger
	env        *envelope.Envelope
	storageKey keys.PublicKey
	crypto     crypto.Crypto
	seal       crypto.Cipher
}

func (o *Outgoing) SetStorageCrypto(k keys.PublicKey, c crypto.Crypto) (err error) {
	o.storageKey = k
	o.crypto = c

	var skey interface{}
	if skey, err = k.SealingKey(); err != nil {
		o.log.Error().Err(err).Msg("could not retrieve public key for storage encryption")
		return internalError
	}

	switch t := skey.(type) {
	case *rsa.PublicKey:
		if o.seal, err = rsaoeap.New(t); err != nil {
			o.log.Error().Err(err).Msg("could not create new rsa-oeap sealing cipher")
			return internalError
		}
	default:
		o.log.Error().Type("type", t).Msg("unknown cipher type for storage encryption")
		return internalError
	}

	return nil
}

// Creates an model to save an outgoing secure envelope to disk. The complicated thing
// about outgoing secure envelopes is that they're encrypted with the recipient's public
// keys, so instead, original envelope is kept intact and the encryption key and hmac
// secret are saved with the keys used to decrypt the associated incoming envelope.
func (o *Outgoing) Model() (model *models.SecureEnvelope, err error) {
	se := o.env.Proto()
	model = &models.SecureEnvelope{
		Direction:     models.DirectionOutgoing,
		IsError:       o.env.IsError(),
		EncryptionKey: nil,
		HMACSecret:    nil,
		ValidHMAC:     sql.NullBool{Valid: true, Bool: se.Sealed},
		PublicKey:     sql.NullString{Valid: false},
		Envelope:      se,
	}

	if !o.env.IsError() {
		// Encrypt the outgoing envelope
		if o.storageKey == nil || o.crypto == nil {
			o.log.Error().Msg("missing storage key or crypto reference to encrypt outgoing envelope locally")
			return nil, internalError
		}

		// Store the public key signature used to encrypt the locally stored envelope
		if model.PublicKey.String, err = o.storageKey.PublicKeySignature(); err != nil {
			o.log.Error().Err(err).Msg("unknown storage key public key signature")
		}
		model.PublicKey.Valid = model.PublicKey.String != ""

		if model.EncryptionKey, err = o.seal.Encrypt(o.crypto.EncryptionKey()); err != nil {
			o.log.Error().Err(err).Msg("unable to encrypt locally stored envelope encryption key")
			return nil, internalError
		}

		if model.HMACSecret, err = o.seal.Encrypt(o.crypto.HMACSecret()); err != nil {
			o.log.Error().Err(err).Msg("unable to encrypt locally stored envelope hmac secret")
			return nil, internalError
		}
	}

	model.EnvelopeID, _ = o.env.UUID()
	model.Timestamp, _ = o.env.Timestamp()
	return model, nil
}

//===========================================================================
// TRISA Transfer Handler Methods
//===========================================================================

func (s *Server) HandleIncoming(in *Incoming) (out *Outgoing, err error) {
	// Validate the incoming message
	if err = in.env.ValidateMessage(); err != nil {
		in.log.Debug().Err(err).Bool("stored_to_database", false).Msg("received invalid secure envelope, no secure envelopes saved to the database")
		return in.Reject(api.BadRequest, err.Error(), true)
	}

	// Parse the envelope ID
	var envelopeID uuid.UUID
	if envelopeID, err = in.env.UUID(); err != nil {
		in.log.Warn().Err(err).Bool("stored_to_database", false).Msg("received invalid secure envelope id, no secure envelopes saved to the database")
		return in.Reject(api.BadRequest, "could not parse envelope id as UUID", true)
	}

	// Create the prepared transaction to handle envelope storage
	if in.db, err = s.store.PrepareTransaction(in.ctx, envelopeID); err != nil {
		in.log.Warn().Err(err).Bool("stored_to_database", false).Msg("could not prepare transaction for database storage")
		return nil, internalError
	}

	// Rollback the prepared transaction if there are any errors.
	defer in.db.Rollback()

	// Handle storing basic transaction details and counterparty information
	if err = in.UpdateRecord(); err != nil {
		in.log.Error().Err(err).Bool("stored_to_database", false).Msg("could not store basic transaction details and counterparty information")
		return nil, internalError
	}

	// Handle the envelope, depending on the incoming envelope state.
	// NOTE: it is up to the handler to store the incoming secure envelope
	switch in.env.State() {
	case envelope.Sealed:
		out, err = s.HandleSealed(in)
	case envelope.Error:
		// If the envelope only contains an error, handle it without decryption
		out, err = s.HandleIncomingError(in)
	case envelope.Corrupted:
		out, err = in.Reject(api.BadRequest, "received envelope in corrupted state", false)
	default:
		out, err = in.Reject(api.BadRequest, "received envelope in unhandled state", true)
	}

	// Return any errors directly to the user, with a warning that no envelopes were stored
	if err != nil {
		in.log.Warn().Err(err).Bool("stored_to_database", false).Msg("could not process incoming trisa transfer, no secure envelopes saved to the database")
		return nil, err
	}

	// Store the outgoing message to the database
	var outse *models.SecureEnvelope
	if outse, err = out.Model(); err != nil {
		in.log.Error().Err(err).Bool("stored_to_database", false).Msg("could not create outgoing envelope for storage")
		return nil, internalError
	}

	if err = in.db.AddEnvelope(outse); err != nil {
		in.log.Error().Err(err).Bool("stored_to_database", false).Msg("could not store outgoing envelope")
		return nil, internalError
	}

	// Commit the transaction to the database
	if err = in.db.Commit(); err != nil {
		in.log.Warn().Err(err).Bool("stored_to_database", false).Msg("could not commit incoming transfer and response to database")
		return nil, internalError
	}

	in.log.Info().Bool("stored_to_database", true).Msg("incoming transfer handling complete")
	return out, nil
}

func (s *Server) HandleSealed(in *Incoming) (out *Outgoing, err error) {
	// Identify the sealing key of the counterparty to return an encrypted response, if
	// it's not available, perform a side-channel RPC to fetch the keys.
	var sealingKey keys.PublicKey
	if sealingKey, err = s.network.SealingKey(in.peer.Name()); err != nil {
		in.log.Info().Msg("conducting mid-transfer key exchange")
		if sealingKey, err = s.network.KeyExchange(in.ctx, in.peer); err != nil {
			// If we cannot exchange keys, return a TRISA rejection error for retry
			in.log.Warn().Err(err).Msg("cannot complete transfer without counterparty sealing keys")
			return in.Reject(api.NoSigningKey, "unable to identify sender's sealing keys to complete transfer", true)
		}
	}

	// Identify local unsealing keys to decrypt the incoming envelope
	var unsealingKey keys.PrivateKey
	if unsealingKey, err = s.network.UnsealingKey(in.PublicKeySignature(), in.peer.Name()); err != nil {
		// Return TRISA rejection message if we cannot unseal the envelope
		in.log.Warn().Err(err).Str("pks", in.PublicKeySignature()).Msg("could not identify unsealing key for envelope")
		return in.Reject(api.InvalidKey, "unknown public key signature", true)
	}

	// Identify the local keys to store the outgoing envelope (usually the public key component of the unsealing key)
	var storageKey keys.PublicKey
	if storageKey, err = s.network.StorageKey(in.PublicKeySignature(), in.peer.Name()); err != nil {
		in.log.Warn().Err(err).Str("pks", in.PublicKeySignature()).Msg("could not identify storage key for envelope")
		return in.Reject(api.InvalidKey, "unknown public key signature", true)
	}

	// Decryption and validation
	var (
		reject    *api.Error
		payload   *api.Payload
		unseal    interface{}
		unsealed  *envelope.Envelope
		decrypted *envelope.Envelope
	)

	if unseal, err = unsealingKey.UnsealingKey(); err != nil {
		in.log.Error().Err(err).Str("pks", in.PublicKeySignature()).Msg("unsealing private key not available")
		return nil, internalError
	}

	if unsealed, reject, err = in.env.Unseal(envelope.WithUnsealingKey(unseal)); err != nil {
		if reject != nil {
			return in.Error(reject)
		}

		in.log.Error().Err(err).Str("pks", in.PublicKeySignature()).Msg("could not unseal incoming secure envelope")
		return nil, internalError
	}

	if decrypted, reject, err = unsealed.Decrypt(); err != nil {
		if reject != nil {
			// Record if the HMAC was not valid
			if reject.Code == api.InvalidSignature {
				in.SetHMACValid(false)
			}
			return in.Error(reject)
		}

		in.log.Error().Err(err).Str("pks", in.PublicKeySignature()).Msg("could not decrypt incoming secure envelope")
		return nil, internalError
	}

	// At this point if we've successfully decrypted the message, we know the HMAC is valid
	in.SetHMACValid(true)

	if payload, err = decrypted.Payload(); err != nil {
		in.log.Error().Err(err).Str("pks", in.PublicKeySignature()).Msg("could not decrypt incoming secure envelope")
		return nil, internalError
	}

	if reject = Validate(payload); reject != nil {
		return in.Error(reject)
	}

	// Update transaction with decrypted details if available
	if err = in.db.Update(transactionFromPayload(payload)); err != nil {
		in.log.Error().Err(err).Msg("could not update transaction in database with decrypted details")
		return nil, internalError
	}

	// TODO: load auto approve/reject policies for counterparty to determine response
	// TODO: send message to callback server on backend

	// NOTE: for now, the server will always simply return a pending response
	if payload, err = pendingPayload(payload, in.ID()); err != nil {
		in.log.Error().Err(err).Msg("could not create outgoing payload")
		return nil, internalError
	}

	var seal interface{}
	if seal, err = sealingKey.SealingKey(); err != nil {
		in.log.Error().Err(err).Msg("sealing public key not available")
		return nil, internalError
	}

	var msg *api.SecureEnvelope
	if msg, reject, err = envelope.Seal(payload, envelope.WithSealingKey(seal), envelope.WithEnvelopeID(in.ID()), envelope.WithCrypto(decrypted.Crypto())); err != nil {
		if reject != nil {
			return in.Error(reject)
		}

		pks, _ := sealingKey.PublicKeySignature()
		in.log.Error().Err(err).Str("pks", pks).Msg("could not seal outgoing envelope")
		return nil, internalError
	}

	// Create outgoing message
	if out, err = in.Outgoing(msg); err != nil {
		return nil, err
	}

	// Set the ougoing message cryptography
	if err = out.SetStorageCrypto(storageKey, decrypted.Crypto()); err != nil {
		return nil, err
	}

	return out, nil
}

// Handles envelopes that only contain errors and require no decryption. The error is
// stored locally and to complete the transfer, the error is echoed back to the sender.
func (s *Server) HandleIncomingError(in *Incoming) (out *Outgoing, err error) {
	// If the transaction doesn't exist, why are we receiving an error?
	if in.db.Created() {
		return nil, status.Error(codes.NotFound, "transaction does not exist")
	}

	// Fetch the error and log it
	trisaError := in.env.Error()
	in.log.Debug().
		Str("code", trisaError.Code.String()).
		Str("message", trisaError.Message).
		Bool("retry", trisaError.Retry).
		Msg("received trisa rejection")

	// Construct a reply that simply echos back the received error.
	// NOTE: do not use in.Error() as that logs that we are sending a rejection response
	var msg *api.SecureEnvelope
	if msg, err = envelope.Reject(trisaError, envelope.WithEnvelopeID(in.ID())); err != nil {
		log.Error().Err(err).Msg("could not prepare rejection envelope")
		return nil, status.Error(codes.Internal, "could not complete TRISA transfer")
	}

	return in.Outgoing(msg)
}

//===========================================================================
// Helper Methods
//===========================================================================

func pendingPayload(in *api.Payload, envelopeID string) (out *api.Payload, err error) {
	ts := time.Now().UTC()
	out = &api.Payload{
		Identity:   in.Identity,
		SentAt:     in.SentAt,
		ReceivedAt: ts.Format(time.RFC3339),
	}

	// TODO: populate pending from configuration
	pending := &generic.Pending{
		EnvelopeId:     envelopeID,
		ReceivedBy:     "TRISA Self Hosted Node",
		ReceivedAt:     ts.Format(time.RFC3339),
		Message:        "We are reviewing your travel rule exchange request and will reply once we have completed our internal compliance checks",
		ReplyNotAfter:  ts.Add(24 * time.Hour).Format(time.RFC3339),
		ReplyNotBefore: ts.Add(5 * time.Minute).Format(time.RFC3339),
		Transaction:    &generic.Transaction{},
	}

	// If we've received a transaction, add it to the pending response
	// NOTE: ignoring errors here, expecting transaction to be nil if we didn't receive
	// an incoming transaction (e.g. if we received another pending message).
	in.Transaction.UnmarshalTo(pending.Transaction)

	// Add the pending payload to the transaction
	if out.Transaction, err = anypb.New(pending); err != nil {
		return nil, err
	}
	return out, nil
}

func transactionFromPayload(in *api.Payload) *models.Transaction {
	var (
		err                error
		originator         string
		originatorAddress  string
		beneficiary        string
		beneficiaryAddress string
		virtualAsset       string
		amount             float64
	)

	data := &generic.Transaction{}
	if err = in.Transaction.UnmarshalTo(data); err == nil {
		switch {
		case data.Network != "" && data.AssetType != "":
			virtualAsset = fmt.Sprintf("%s (%s)", data.Network, data.AssetType)
		case data.Network != "":
			virtualAsset = data.Network
		case data.AssetType != "":
			virtualAsset = data.AssetType
		}

		amount = data.Amount
		originatorAddress = data.Originator
		beneficiaryAddress = data.Beneficiary
	}

	identity := &ivms101.IdentityPayload{}
	if err = in.Identity.UnmarshalTo(identity); err == nil {
		if identity.Originator != nil {
			originator = findName(identity.Originator.OriginatorPersons...)
		}

		if identity.Beneficiary != nil {
			beneficiary = findName(identity.Beneficiary.BeneficiaryPersons...)
		}

		if originatorAddress == "" {
			originatorAddress = findAccount(identity.Originator)
		}

		if beneficiaryAddress == "" {
			beneficiaryAddress = findAccount(identity.Beneficiary)
		}
	}

	return &models.Transaction{
		Originator:         sql.NullString{Valid: originator != "", String: originator},
		OriginatorAddress:  sql.NullString{Valid: originatorAddress != "", String: originatorAddress},
		Beneficiary:        sql.NullString{Valid: beneficiary != "", String: beneficiary},
		BeneficiaryAddress: sql.NullString{Valid: beneficiaryAddress != "", String: beneficiaryAddress},
		VirtualAsset:       virtualAsset,
		Amount:             amount,
	}

}

func findName(persons ...*ivms101.Person) (name string) {
	// Search all persons for the first legal name available. Use the last available
	// non-zero name for any other name identifier types.
	for _, person := range persons {
		switch t := person.Person.(type) {
		case *ivms101.Person_LegalPerson:
			if t.LegalPerson.Name != nil {
				for _, identifier := range t.LegalPerson.Name.NameIdentifiers {
					// Set the name found to the current legal person name
					if identifier.LegalPersonName != "" {
						name = identifier.LegalPersonName

						// If this is the legal name, short circuit and return it.
						if identifier.LegalPersonNameIdentifierType == ivms101.LegalPersonLegal {
							return name
						}
					}
				}
			}
		case *ivms101.Person_NaturalPerson:
			if t.NaturalPerson.Name != nil {
				for _, identifier := range t.NaturalPerson.Name.NameIdentifiers {
					// Set the name found to the current natural person name
					if identifier.PrimaryIdentifier != "" {
						name = strings.TrimSpace(fmt.Sprintf("%s %s", identifier.SecondaryIdentifier, identifier.PrimaryIdentifier))

						// If this is the legal name of the person, short circuit and return it.
						if identifier.NameIdentifierType == ivms101.NaturalPersonLegal {
							return name
						}
					}
				}
			}
		}

	}

	// Return whatever non-zero name we found, or empty string if we found nothing.
	return name
}

func findAccount(person any) (account string) {
	if person == nil {
		return ""
	}

	switch t := person.(type) {
	case *ivms101.Originator:
		for _, account = range t.AccountNumbers {
			if account != "" {
				return account
			}
		}
	case *ivms101.Beneficiary:
		for _, account = range t.AccountNumbers {
			if account != "" {
				return account
			}
		}
	}

	// Return whatever non-zero account we found, or empty string if we found nothing.
	return account
}

func streamClosed(err error) bool {
	if err == io.EOF {
		return true
	}

	if serr, ok := status.FromError(err); ok {
		if serr.Code() == codes.Canceled {
			return true
		}
	}

	return false
}
