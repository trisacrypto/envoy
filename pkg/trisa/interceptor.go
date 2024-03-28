package trisa

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// The health check endpoint for TRISA nodes.
const statusEndpoint = "/trisa.api.v1beta1.TRISAHealth/Status"

// Returns the server option chaining all unary interceptors in the specified order.
func (s *Server) UnaryInterceptors() grpc.ServerOption {
	return grpc.ChainUnaryInterceptor(
		s.UnaryAvailable(),
	)
}

// Returns the server option chaining all stream interceptors in the specified order.
func (s *Server) StreamInterceptors() grpc.ServerOption {
	return grpc.ChainStreamInterceptor(
		s.StreamAvailable(),
	)
}

// Maintenance message returned with unavailable status
const maintenanceMessage = "conducting temporary maintenance"

func (s *Server) UnaryAvailable() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, in interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (out interface{}, err error) {
		// If we're in maintenance mode, do not respond to requests unless it is the
		// health check endpoint that we're responding to.
		if s.conf.Maintenance && info.FullMethod != statusEndpoint {
			return nil, status.Error(codes.Unavailable, maintenanceMessage)
		}

		// Handle the request as normal
		return handler(ctx, in)
	}
}

func (s *Server) StreamAvailable() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// If we're in maintenance mode, do not respond to stream requests.
		if s.conf.Maintenance {
			return status.Error(codes.Unavailable, maintenanceMessage)
		}

		// Handle the request as normal
		return handler(srv, ss)
	}
}
