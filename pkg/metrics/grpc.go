package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	// Generic gRPC collectors for observability defined here.
	RPCStarted    *prometheus.CounterVec
	RPCHandled    *prometheus.CounterVec
	RPCDuration   *prometheus.HistogramVec
	StreamMsgSent *prometheus.CounterVec
	StreamMsgRecv *prometheus.CounterVec
)

func initGRPCCollectors() (collectors []prometheus.Collector, err error) {
	collectors = make([]prometheus.Collector, 0, 5)

	RPCStarted = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: NamespaceGRPCMetrics,
		Name:      "server_started_total",
		Help:      "count the total number of RPCs started on the server",
	}, []string{"type", "service", "method"})
	collectors = append(collectors, RPCStarted)

	RPCHandled = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: NamespaceGRPCMetrics,
		Name:      "server_handled_total",
		Help:      "count the total number of RPCs completed on the server regardless of success or failure",
	}, []string{"type", "service", "method", "code"})
	collectors = append(collectors, RPCHandled)

	RPCDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: NamespaceGRPCMetrics,
		Name:      "server_handler_duration",
		Help:      "response latency (in seconds) of the application handler for the rpc method",
	}, []string{"type", "service", "method"})
	collectors = append(collectors, RPCDuration)

	StreamMsgSent = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: NamespaceGRPCMetrics,
		Name:      "server_stream_messages_sent",
		Help:      "total number of streaming messages sent by the server",
	}, []string{"type", "service", "method"})
	collectors = append(collectors, StreamMsgSent)

	StreamMsgRecv = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: NamespaceGRPCMetrics,
		Name:      "server_stream_messages_recv",
		Help:      "total number of streaming messages received by the server",
	}, []string{"type", "service", "method"})
	collectors = append(collectors, StreamMsgRecv)

	return collectors, nil
}
