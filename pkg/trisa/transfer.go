package trisa

import (
	"context"
	"io"
	"sync"
	"time"

	"self-hosted-node/pkg/logger"
	"self-hosted-node/pkg/trisa/peers"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	api "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
	generic "github.com/trisacrypto/trisa/pkg/trisa/data/generic/v1beta1"
	"github.com/trisacrypto/trisa/pkg/trisa/envelope"
	"github.com/trisacrypto/trisa/pkg/trisa/keys"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/anypb"
)

func (s *Server) Transfer(ctx context.Context, in *api.SecureEnvelope) (out *api.SecureEnvelope, err error) {
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
	incoming := &Incoming{ctx, peer, in, log}
	if out, err = s.HandleIncoming(incoming); err != nil {
		log.Warn().Err(err).Str("envelope_id", in.Id).Msg("trisa transfer handler failed")
		return nil, err
	}

	log.Debug().Msg("trisa transfer completed")
	return out, nil
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
			incoming := &Incoming{ctx, peer, in, log.With().Str("envelope_id", in.Id).Logger()}

			var out *api.SecureEnvelope
			if out, err = s.HandleIncoming(incoming); err != nil {
				log.Warn().Err(err).Str("envelope_id", in.Id).Msg("unable to handle transfer request, stream closing")
				return
			}

			// Queue the message to be sent on the outgoing channel
			outgoing <- out
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

// Incoming stores the full context of an incoming transfer message for handling.
type Incoming struct {
	ctx  context.Context
	peer peers.Peer
	env  *api.SecureEnvelope
	log  zerolog.Logger
}

func (i *Incoming) ID() string {
	return i.env.Id
}

func (s *Server) HandleIncoming(in *Incoming) (out *api.SecureEnvelope, err error) {
	// TODO: store incoming envelope before processing

	// If the envelope only contains an error, handle it without decryption
	switch envelope.Status(in.env) {
	case envelope.Error:
		return s.HandleIncomingError(in)
	case envelope.Corrupted:
		return s.Reject(api.BadRequest, "received envelope in corrupted state", false, in)
	}

	// Identify the sealing key of the counterparty to return an encrypted response, if
	// it's not available, perform a side-channel RPC to fetch the keys.
	var sealingKey keys.PublicKey
	if sealingKey, err = s.network.SealingKey(in.peer.Name()); err != nil {
		in.log.Info().Msg("conducting mid-transfer key exchange")
		if sealingKey, err = s.network.KeyExchange(in.ctx, in.peer); err != nil {
			// If we cannot exchange keys, return a TRISA rejection error for retry
			in.log.Warn().Err(err).Msg("cannot complete transfer without counterparty sealing keys")
			return s.Reject(api.NoSigningKey, "unable to identify sender's sealing keys to complete transfer", true, in)
		}
	}

	// Identify local unsealing keys to decrypt the incoming envelope
	var unsealingKey keys.PrivateKey
	if unsealingKey, err = s.network.UnsealingKey(in.env.PublicKeySignature, in.peer.Name()); err != nil {
		// Return TRISA rejection message if we cannot unseal the envelope
		in.log.Warn().Err(err).Str("pks", in.env.PublicKeySignature).Msg("could not identify unsealing key for envelope")
		return s.Reject(api.InvalidKey, "unknown public key signature", true, in)
	}

	// Decryption and validation
	var (
		reject  *api.Error
		payload *api.Payload
		unseal  interface{}
	)

	if unseal, err = unsealingKey.UnsealingKey(); err != nil {
		in.log.Error().Err(err).Str("pks", in.env.PublicKeySignature).Msg("unsealing private key not available")
		return nil, status.Error(codes.Internal, "unable to process secure envelope")
	}

	if payload, reject, err = envelope.Open(in.env, envelope.WithUnsealingKey(unseal)); err != nil {
		if reject != nil {
			return s.Reject(reject.Code, reject.Message, reject.Retry, in)
		}

		in.log.Error().Err(err).Str("pks", in.env.PublicKeySignature).Msg("could not open incoming secure envelope")
		return nil, status.Error(codes.Internal, "unable to process secure envelope")
	}

	if reject = Validate(payload); reject != nil {
		return s.Reject(reject.Code, reject.Message, reject.Retry, in)
	}

	// TODO: load auto approve/reject policies for counterparty to determine response
	// TODO: send message to callback server on backend

	// NOTE: for now, the server will always simply return a pending response
	if payload, err = pendingPayload(payload, in.ID()); err != nil {
		in.log.Error().Err(err).Msg("could not create outgoing payload")
		return nil, status.Error(codes.Internal, "unable to process secure envelope")
	}

	var seal interface{}
	if seal, err = sealingKey.SealingKey(); err != nil {
		in.log.Error().Err(err).Msg("sealing public key not available")
		return nil, status.Error(codes.Internal, "unable to process secure envelope")
	}

	if out, reject, err = envelope.Seal(payload, envelope.WithSealingKey(seal), envelope.WithEnvelopeID(in.ID())); err != nil {
		if reject != nil {
			return s.Reject(reject.Code, reject.Message, reject.Retry, in)
		}

		pks, _ := sealingKey.PublicKeySignature()
		in.log.Error().Err(err).Str("pks", pks).Msg("could not seal outgoing envelope")
		return nil, status.Error(codes.Internal, "unable to process secure envelope")
	}

	// TODO: store outgoing envelope for auding and retrieval purposes

	return out, nil
}

// Handles envelopes that only contain errors and require no decryption. The error is
// stored locally and to complete the transfer, the error is echoed back to the sender.
func (s *Server) HandleIncomingError(in *Incoming) (out *api.SecureEnvelope, err error) {
	// Construct a reply
	if out, err = envelope.Reject(in.env.Error, envelope.WithEnvelopeID(in.ID())); err != nil {
		log.Error().Err(err).Msg("could not create error response")
		return nil, status.Error(codes.Internal, "could not respond to error envelope")
	}

	// TODO: store outgoing envelope for auding and retrieval purposes

	in.log.Debug().
		Str("code", in.env.Error.Code.String()).
		Str("message", in.env.Error.Message).
		Bool("retry", in.env.Error.Retry).
		Msg("received trisa rejection")
	return out, nil
}

// Helper method for preparing a TRISA error envelope to return the caller.
func (s *Server) Reject(code api.Error_Code, message string, retry bool, in *Incoming) (out *api.SecureEnvelope, err error) {
	reject := &api.Error{
		Code:    code,
		Message: message,
		Retry:   retry,
	}

	if out, err = envelope.Reject(reject, envelope.WithEnvelopeID(in.ID())); err != nil {
		log.Error().Err(err).Msg("could not prepare rejection envelope")
		return nil, status.Error(codes.Internal, "could not complete TRISA transfer")
	}

	// TODO: store outgoing envelope for auditing and retrieval purposes

	in.log.Info().
		Str("code", code.String()).
		Str("message", message).
		Bool("retry", retry).
		Msg("trisa transfer rejected")
	return out, nil
}

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
