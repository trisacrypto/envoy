package interceptors

import (
	"context"

	"github.com/trisacrypto/envoy/pkg/config"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// The health check endpoint for TRISA nodes.
const statusEndpoint = "/trisa.api.v1beta1.TRISAHealth/Status"

// Maintenance message returned with unavailable status
const maintenanceMessage = "conducting temporary maintenance"

func UnaryAvailable(conf config.TRISAConfig) grpc.UnaryServerInterceptor {
	// Keep the maintenance boolean with the closure so we're not passing the entire
	// server object into the interceptor.
	maintenance := conf.Maintenance
	return func(ctx context.Context, in interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (out interface{}, err error) {
		// If we're in maintenance mode, do not respond to requests unless it is the
		// health check endpoint that we're responding to.
		if maintenance && info.FullMethod != statusEndpoint {
			return nil, status.Error(codes.Unavailable, maintenanceMessage)
		}

		// Handle the request as normal
		return handler(ctx, in)
	}
}

func StreamAvailable(conf config.TRISAConfig) grpc.StreamServerInterceptor {
	// Keep the maintenance boolean with the closure so we're not passing the entire
	// server object into the interceptor.
	maintenance := conf.Maintenance
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// If we're in maintenance mode, do not respond to stream requests.
		if maintenance {
			return status.Error(codes.Unavailable, maintenanceMessage)
		}

		// Handle the request as normal
		return handler(srv, ss)
	}
}
