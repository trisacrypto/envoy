package trisa

import (
	"context"

	"github.com/trisacrypto/envoy/pkg/logger"
	"github.com/trisacrypto/envoy/pkg/trisa/peers"

	api "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
	"github.com/trisacrypto/trisa/pkg/trisa/keys"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// KeyExchange is a preperatory RPC that is required before Transfer RPCs to ensure each
// counterparty has the public keys needed to encrypt secure envelopes for the recipient.
func (s *Server) KeyExchange(ctx context.Context, in *api.SigningKey) (out *api.SigningKey, err error) {
	// Add tracing from context to the log context.
	log := logger.Tracing(ctx)

	// Identify the counterparty peer from the context.
	var peer peers.Peer
	if peer, err = s.network.FromContext(ctx); err != nil {
		log.Error().Err(err).Msg("could not identify peer from context")
		return nil, status.Error(codes.Unauthenticated, "could not identify remote peer from mTLS certificates")
	}

	// Add peer to the log context.
	log = log.With().Str("peer", peer.String()).Logger()

	// Store incoming sealing key in the key cache
	var remote keys.Key
	if remote, err = keys.FromSigningKey(in); err != nil {
		log.Error().Err(err).Msg("could not parse incoming signing key")
		return nil, status.Error(codes.InvalidArgument, "could not parse public key in request")
	}

	if err = s.network.Cache(peer.Name(), remote); err != nil {
		log.Error().Err(err).Msg("could not cache exchange key from remote peer")
		return nil, status.Error(codes.Internal, "could not complete key exchange")
	}

	// Prepare exchange key to return back to the remote peer
	var local keys.PublicKey
	if local, err = s.network.ExchangeKey(peer.Name()); err != nil {
		log.Error().Err(err).Msg("could not retrieve sealing key to exchange with remote peer")
		return nil, status.Error(codes.Internal, "could not complete key exchange")
	}

	if out, err = local.Proto(); err != nil {
		log.Error().Err(err).Msg("could not marshal sealing key protocol buffers")
		return nil, status.Error(codes.Internal, "could not complete key exchange")
	}

	return out, nil
}
