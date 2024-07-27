package trisa

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/trisacrypto/envoy/pkg/logger"
	"github.com/trisacrypto/envoy/pkg/postman"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/envoy/pkg/trisa/peers"

	"github.com/trisacrypto/trisa/pkg/ivms101"
	api "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
	generic "github.com/trisacrypto/trisa/pkg/trisa/data/generic/v1beta1"
	"github.com/trisacrypto/trisa/pkg/trisa/envelope"
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

	// Add peer to the log context and log message received.
	log = log.With().Str("peer", peer.String()).Str("envelope_id", in.Id).Logger()

	// Create a packet to handle the incoming request
	var packet *postman.Packet
	if packet, err = postman.Receive(in, log, peer); err != nil {
		log.Error().Err(err).Msg("could not start trisa transfer")
		return nil, internalError
	}

	// Log that the incoming transfer has been received
	packet.Log.Debug().Msg("trisa transfer received")

	// Handle the incoming transfer
	if err = s.Handle(ctx, packet); err != nil {
		packet.Log.Warn().Err(err).Msg("trisa transfer handler failed")
		return nil, err
	}

	packet.Log.Debug().Msg("trisa transfer completed")
	return packet.Out.Proto(), nil
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

			// Create a packet to handle the incoming request
			var packet *postman.Packet
			if packet, err = postman.Receive(in, log, peer); err != nil {
				log.Error().Err(err).Msg("could not handle stream message, stream closing")
				return
			}

			if err = s.Handle(ctx, packet); err != nil {
				packet.Log.Warn().Err(err).Msg("unable to handle transfer request, stream closing")
				return
			}

			// Queue the message to be sent on the outgoing channel
			outgoing <- packet.Out.Proto()
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
// TRISA Transfer Handler Methods
//===========================================================================

func (s *Server) Handle(ctx context.Context, p *postman.Packet) (err error) {
	// Ensure that the error returned from this function is a gRPC error
	defer func() {
		if err != nil {
			if _, ok := status.FromError(err); !ok {
				err = internalError
			}
		}
	}()

	// Validate the incoming message
	if err = p.In.Envelope.ValidateMessage(); err != nil {
		p.Log.Debug().Err(err).Bool("stored_to_database", false).Msg("received invalid secure envelope, no secure envelopes saved to the database")
		return p.Reject(api.BadRequest, err.Error(), false)
	}

	// Parse the envelope ID
	var envelopeID uuid.UUID
	if envelopeID, err = p.In.Envelope.UUID(); err != nil {
		p.Log.Warn().Err(err).Bool("stored_to_database", false).Msg("received invalid secure envelope id, no secure envelopes saved to the database")
		return p.Reject(api.BadRequest, "could not parse envelope id as UUID", false)
	}

	// Create the prepared transaction to handle envelope storage
	if p.DB, err = s.store.PrepareTransaction(ctx, envelopeID); err != nil {
		p.Log.Warn().Err(err).Bool("stored_to_database", false).Msg("could not prepare transaction for database storage")
		return internalError
	}

	// Rollback the prepared transaction if there are any errors in processing
	defer p.DB.Rollback()

	// Update the transaction record and add counterparty information and status
	// TODO: this may return an invalid counterparty error, which should return a different status error
	if err = p.In.UpdateTransaction(); err != nil {
		p.Log.Warn().Err(err).Bool("stored_to_database", false).Msg("could not update transaction details and counterparty information")
		return internalError
	}

	// Handle the envelope, depending on the incoming envelope's state.
	switch state := p.In.Envelope.State(); state {
	case envelope.Sealed:
		err = s.HandleSealed(ctx, p)
	case envelope.Error:
		err = s.HandleError(ctx, p)
	case envelope.Corrupted:
		p.Log.Warn().Str("state", state.String()).Msg("received envelope in corrupted state")
		return status.Error(codes.InvalidArgument, "received envelope in corrupted state")
	default:
		p.Log.Warn().Str("state", state.String()).Msg("received envelope in unhandled state")
		return status.Error(codes.InvalidArgument, "received envelope in unhandled state")
	}

	if err != nil {
		p.Log.Warn().Err(err).Bool("stored_to_database", false).Msg("could not process incoming trisa transfer")
		return err
	}

	// Store Incoming Message
	if err = p.DB.AddEnvelope(p.In.Model()); err != nil {
		p.Log.Error().Err(err).Bool("stored_to_database", false).Msg("could not store incoming trisa envelope in database")
		return internalError
	}

	// Store Outgoing message
	if err = p.DB.AddEnvelope(p.Out.Model()); err != nil {
		p.Log.Error().Err(err).Bool("stored_to_database", false).Msg("could not store outgoing trisa envelope in database")
		return internalError
	}

	// Commit the transaction to the database (success!)
	if err = p.DB.Commit(); err != nil {
		p.Log.Warn().Err(err).Bool("stored_to_database", false).Msg("could not commit incoming trisa transfer to database")
		return internalError
	}

	p.Log.Info().Bool("stored_to_database", true).Msg("incoming trisa transfer handling complete")
	return nil
}

func (s *Server) HandleSealed(ctx context.Context, p *postman.Packet) (err error) {
	// Identify the unsealing keys to decrypt the incoming envelope
	if p.In.UnsealingKey, err = s.network.UnsealingKey(p.In.PublicKeySignature(), p.Peer.Name()); err != nil {
		// Return TRISA rejection message if we cannot unseal the envelope
		p.Log.Warn().Err(err).Str("pks", p.In.PublicKeySignature()).Msg("could not identify unsealing key for envelope")
		return p.Reject(api.InvalidKey, "unknown public key signature", true)
	}

	// Identify the sealing key of the counterparty to return an encrypted response, if
	// it's not available, perform a side-channel RPC to fetch the keys.
	if p.Out.SealingKey, err = s.network.SealingKey(p.Peer.Name()); err != nil {
		p.Log.Info().Msg("conducting mid-transfer key exchange")
		if p.Out.SealingKey, err = s.network.KeyExchange(ctx, p.Peer); err != nil {
			// If we cannot exchange keys, return a TRISA rejection error for retry
			p.Log.Warn().Err(err).Msg("cannot complete transfer without counterparty sealing keys")
			return p.Reject(api.NoSigningKey, "unable to identify sender's sealing keys to complete transfer", true)
		}
	}

	// Identify the local keys used to store the outgoing envelope
	// These are usually the public key component of the unsealing keys
	if p.Out.StorageKey, err = s.network.StorageKey(p.In.PublicKeySignature(), p.Peer.Name()); err != nil {
		p.Log.Warn().Err(err).Str("pks", p.In.PublicKeySignature()).Msg("could not identify storage key for envelope")
		return p.Reject(api.InvalidKey, "unknown public key signature", true)
	}

	// Decryption and Validation
	var reject *api.Error
	if reject, err = p.In.Open(); err != nil {
		if reject != nil {
			return p.Error(reject)
		}
		return err
	}

	var payload *api.Payload
	if payload, err = p.In.Envelope.Payload(); err != nil {
		p.Log.Error().Err(err).Msg("could not retrieve payload from decrypted envelope")
		return internalError
	}

	if reject = Validate(payload); reject != nil {
		return p.Error(reject)
	}

	// Update transaction with decrypted details if available
	// TODO: move transaction from payload to Postman
	if err = p.DB.Update(transactionFromPayload(payload)); err != nil {
		p.Log.Error().Err(err).Msg("could not update transaction in database with decrypted details")
		return internalError
	}

	// TODO: load auto approve/reject policies for counterparty to determine response
	// TODO: send message to callback webhook and attempt to recieve a response

	// NOTE: for now, the server will always simple return a pending response.
	if payload, err = pendingPayload(payload, p.EnvelopeID()); err != nil {
		p.Log.Error().Err(err).Msg("could not create outgoing pending payload")
		return internalError
	}

	// TODO: determine the transfer state to send a message back
	// NOTE: right now we're always just sending back pending
	if err = p.Send(payload, api.TransferPending); err != nil {
		p.Log.Error().Err(err).Msg("could not update outgoing envelope with payload and transfer state")
		return internalError
	}

	// Seal the outgoing envelope so it's ready to return to the requestor
	if reject, err = p.Out.Seal(); err != nil {
		if reject != nil {
			return p.Error(reject)
		}
		return err
	}

	return nil
}

func (s *Server) HandleError(ctx context.Context, p *postman.Packet) (err error) {
	// If the transaction doesn't exist, why are we receiving an error?
	if p.DB.Created() {
		return status.Error(codes.NotFound, "transaction does not exist")
	}

	// Fetch the error and log it
	trisaError := p.In.Envelope.Error()
	p.Log.Debug().
		Str("code", trisaError.Code.String()).
		Str("message", trisaError.Message).
		Bool("retry", trisaError.Retry).
		Msg("received trisa rejection")

	// Update the transaction status to indicate that an error was received.
	var status string
	switch p.In.Envelope.TransferState() {
	case api.TransferRejected:
		status = models.StatusRejected
	case api.TransferRepair:
		status = models.StatusRepair
	default:
		status = models.StatusUnspecified
	}

	if err = p.DB.Update(&models.Transaction{Status: status}); err != nil {
		p.Log.Error().Err(err).Msg("could not update transaction status on error")
		return internalError
	}

	// Construct a reply that simply echos back the received error.
	transferState := api.TransferPending
	if status == models.StatusRejected {
		transferState = api.TransferRejected
	}

	return p.Error(trisaError, envelope.WithTransferState(transferState))
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
