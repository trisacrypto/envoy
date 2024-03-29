package interceptors

import (
	"self-hosted-node/pkg/config"

	"google.golang.org/grpc"
)

// Returns the server option chaining all unary interceptors in the specified order.
func UnaryInterceptors(conf config.TRISAConfig) grpc.ServerOption {
	opts := []grpc.UnaryServerInterceptor{
		UnaryMonitoring(),
		UnaryRecovery(),
	}

	// The very last interceptor should be the availability checker but if we're not
	// in maintenance mode, do not add the final interceptor.
	if available := UnaryAvailable(conf); available != nil {
		opts = append(opts, available)
	}

	return grpc.ChainUnaryInterceptor(opts...)
}

// Returns the server option chaining all stream interceptors in the specified order.
func StreamInterceptors(conf config.TRISAConfig) grpc.ServerOption {
	opts := []grpc.StreamServerInterceptor{
		StreamMonitoring(),
		StreamRecovery(),
	}

	// The very last interceptor should be the availability checker but if we're not
	// in maintenance mode, do not add the final interceptor.
	if available := StreamAvailable(conf); available != nil {
		opts = append(opts, available)
	}

	return grpc.ChainStreamInterceptor(opts...)
}
