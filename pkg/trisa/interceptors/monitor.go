package interceptors

import (
	"context"
	"io"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/trisacrypto/envoy/pkg"
	"github.com/trisacrypto/envoy/pkg/logger"
	"github.com/trisacrypto/envoy/pkg/metrics"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"go.rtnl.ai/ulid"
)

var (
	entropy   io.Reader
	mkentropy sync.Once
)

func initEntropy() {
	mkentropy.Do(func() {
		entropy = ulid.Pool(func() io.Reader {
			return ulid.Monotonic(rand.New(rand.NewSource(time.Now().UnixNano())), 0)
		})
	})
}

// Monitoring handles both logging and outputing Prometheus metrics (if enabled). These
// are embedded into the same interceptor so that the monitoring uses the same logging,
// tracing, and latency -- allowing this to be the outermost interceptor.
// TODO: add prometheus metrics
func UnaryMonitoring() grpc.UnaryServerInterceptor {
	// Initialize entropy if it hasn't already been initialized.
	initEntropy()

	version := pkg.Version(false)
	metrics.Setup()

	return func(ctx context.Context, in interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (out interface{}, err error) {
		// Parse the method for tags
		service, method := ParseMethod(info.FullMethod)

		// Monitor how many RPCs have been started
		metrics.RPCStarted.WithLabelValues("unary", service, method).Inc()

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

		// Monitor how many RPCs have been completed
		metrics.RPCHandled.WithLabelValues("unary", service, method, code.String()).Inc()
		metrics.RPCDuration.WithLabelValues("unary", service, method).Observe(duration.Seconds())

		return out, err
	}
}

// Monitoring handles both logging and outputing Prometheus metrics (if enabled). These
// are embedded into the same interceptor so that the monitoring uses the same logging,
// tracing, and latency -- allowing this to be the outermost interceptor.
// TODO: add prometheus metrics
func StreamMonitoring() grpc.StreamServerInterceptor {
	// Initialize entropy if it hasn't already been initialized.
	initEntropy()

	version := pkg.Version(false)
	metrics.Setup()

	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		// Parse the method for tags
		service, method := ParseMethod(info.FullMethod)

		// Monitor how many streams have been started
		metrics.RPCStarted.WithLabelValues("stream", service, method).Inc()

		// Create a request ID for tracing purposes and add to context
		requestID := ulid.MustNew(ulid.Now(), entropy).String()
		ctx := logger.WithRequestID(stream.Context(), requestID)

		// Create a monitored stream to pass to the stream handler
		monitored := &MonitoredStream{stream, ctx, 0, 0, service, method}

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

		// Monitor how many RPCs have been completed
		metrics.RPCHandled.WithLabelValues("stream", service, method, code.String()).Inc()
		metrics.RPCDuration.WithLabelValues("stream", service, method).Observe(duration.Seconds())

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
	ctx     context.Context
	sends   uint64
	recvs   uint64
	service string
	method  string
}

// Increment the number of sent messages if there is no error on Send.
func (s *MonitoredStream) SendMsg(m interface{}) (err error) {
	if err = s.ServerStream.SendMsg(m); err == nil {
		s.sends++
		metrics.StreamMsgSent.WithLabelValues("stream", s.service, s.method).Inc()
	}
	return err
}

// Increment the number of received messages if there is no error on Recv.
func (s *MonitoredStream) RecvMsg(m interface{}) (err error) {
	if err = s.ServerStream.RecvMsg(m); err == nil {
		s.recvs++
		metrics.StreamMsgRecv.WithLabelValues("stream", s.service, s.method).Inc()
	}
	return err
}

func (s *MonitoredStream) Context() context.Context {
	return s.ctx
}
