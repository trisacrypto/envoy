package interceptors

import (
	"context"
	"io"
	"self-hosted-node/pkg"
	"self-hosted-node/pkg/logger"
	"self-hosted-node/pkg/ulids"
	"strings"
	"sync"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	entropy   io.Reader
	mkentropy sync.Once
)

// Monitoring handles both logging and outputing Prometheus metrics (if enabled). These
// are embedded into the same interceptor so that the monitoring uses the same logging,
// tracing, and latency -- allowing this to be the outermost interceptor.
// TODO: add prometheus metrics
func UnaryMonitoring() grpc.UnaryServerInterceptor {
	// Initialize entropy if it hasn't already been initialized.
	mkentropy.Do(func() {
		entropy = ulids.NewPool()
	})

	version := pkg.Version()
	return func(ctx context.Context, in interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (out interface{}, err error) {
		// Parse the method for tags
		service, method := ParseMethod(info.FullMethod)

		// Create a request ID for tracing purposes and add to context
		requestID := ulid.MustNew(ulid.Now(), entropy).String()
		ctx = logger.WithRequestID(ctx, requestID)

		// Handle the request and trace how long the request takes
		start := time.Now()
		out, err = handler(ctx, in)
		duration := time.Since(start)
		code := status.Code(err)

		// Prepare log context for logging
		log := log.With().
			Str("type", "unary").
			Str("service", service).
			Str("method", method).
			Str("version", version).
			Str("request_id", requestID).
			Uint32("code", uint32(code)).
			Dur("latency", duration).
			Logger()

		switch code {
		case codes.OK:
			log.Info().Msg(info.FullMethod)
		case codes.Canceled, codes.InvalidArgument, codes.NotFound, codes.AlreadyExists, codes.Unauthenticated:
			log.Info().Err(err).Msg(info.FullMethod)
		case codes.DeadlineExceeded, codes.PermissionDenied, codes.ResourceExhausted, codes.FailedPrecondition, codes.Aborted, codes.OutOfRange, codes.Unavailable:
			log.Warn().Err(err).Msg(info.FullMethod)
		case codes.Unknown, codes.Unimplemented, codes.Internal, codes.DataLoss:
			log.Error().Err(err).Msg(info.FullMethod)
		default:
			log.Error().Err(err).Msgf("unhandled error code %s: %s", code, info.FullMethod)
		}

		return out, err
	}
}

// Monitoring handles both logging and outputing Prometheus metrics (if enabled). These
// are embedded into the same interceptor so that the monitoring uses the same logging,
// tracing, and latency -- allowing this to be the outermost interceptor.
// TODO: add prometheus metrics
func StreamMonitoring() grpc.StreamServerInterceptor {
	// Initialize entropy if it hasn't already been initialized.
	mkentropy.Do(func() {
		entropy = ulids.NewPool()
	})

	version := pkg.Version()
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		// Parse the method for tags
		service, method := ParseMethod(info.FullMethod)

		// Create a request ID for tracing purposes and add to context
		requestID := ulid.MustNew(ulid.Now(), entropy).String()
		ctx := logger.WithRequestID(stream.Context(), requestID)

		// Create a monitored stream to pass to the stream handler
		monitored := &MonitoredStream{ServerStream: stream, ctx: ctx}

		// Handle the request and trace how long the request takes
		start := time.Now()
		err = handler(srv, monitored)
		duration := time.Since(start)
		code := status.Code(err)

		// Prepare log context for logging
		log := log.With().
			Str("type", "stream").
			Str("service", service).
			Str("method", method).
			Str("version", version).
			Str("request_id", requestID).
			Uint32("code", uint32(code)).
			Uint64("sent", monitored.sends).
			Uint64("recv", monitored.recvs).
			Dur("latency", duration).
			Logger()

		switch code {
		case codes.OK:
			log.Info().Msg(info.FullMethod)
		case codes.Canceled, codes.InvalidArgument, codes.NotFound, codes.AlreadyExists, codes.Unauthenticated:
			log.Info().Err(err).Msg(info.FullMethod)
		case codes.DeadlineExceeded, codes.PermissionDenied, codes.ResourceExhausted, codes.FailedPrecondition, codes.Aborted, codes.OutOfRange, codes.Unavailable:
			log.Warn().Err(err).Msg(info.FullMethod)
		case codes.Unknown, codes.Unimplemented, codes.Internal, codes.DataLoss:
			log.Error().Err(err).Msg(info.FullMethod)
		default:
			log.Error().Err(err).Msgf("unhandled error code %s: %s", code, info.FullMethod)
		}

		return err
	}
}

func ParseMethod(method string) (string, string) {
	method = strings.TrimPrefix(method, "/") // remove leading slash
	if i := strings.Index(method, "/"); i >= 0 {
		return method[:i], method[i+1:]
	}
	return "unknown", "unknown"
}

// MonitoredStream wraps a grpc.ServerStream to count the number of messages sent in
// the stream for logging purposes. It also provides the stream the adapted context.
type MonitoredStream struct {
	grpc.ServerStream
	ctx   context.Context
	sends uint64
	recvs uint64
}

// Increment the number of sent messages if there is no error on Send.
func (s *MonitoredStream) SendMsg(m interface{}) (err error) {
	if err = s.ServerStream.SendMsg(m); err == nil {
		s.sends++
	}
	return err
}

// Increment the number of received messages if there is no error on Recv.
func (s *MonitoredStream) RecvMsg(m interface{}) (err error) {
	if err = s.ServerStream.RecvMsg(m); err == nil {
		s.recvs++
	}
	return err
}

func (s *MonitoredStream) Context() context.Context {
	return s.ctx
}
