package trisa

import (
	"context"
	"io"
	"sync"

	"self-hosted-node/pkg/logger"
	"self-hosted-node/pkg/trisa/peers"

	api "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	log = log.With().Str("peer", peer.String()).Logger()
	log.Debug().Str("envelope_id", in.Id).Msg("trisa transfer received")

	// Handle the incoming transfer
	if out, err = s.HandleIncomingTransfer(ctx, peer, in); err != nil {
		log.Warn().Err(err).Str("envelope_id", in.Id).Msg("trisa transfer handler failed")
		return nil, err
	}

	log.Debug().Str("envelope_id", out.Id).Msg("trisa transfer completed")
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
			var out *api.SecureEnvelope
			if out, err = s.HandleIncomingTransfer(ctx, peer, in); err != nil {
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

func (s *Server) HandleIncomingTransfer(ctx context.Context, peer peers.Peer, in *api.SecureEnvelope) (out *api.SecureEnvelope, err error) {
	return &api.SecureEnvelope{}, nil
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
