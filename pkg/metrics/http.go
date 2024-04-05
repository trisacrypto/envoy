package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	// Total HTTP requests handled by the server
	RequestsHandled *prometheus.CounterVec

	// HTTP Request duration (latency)
	RequestDuration *prometheus.HistogramVec
)

func initHTTPCollectors() (collectors []prometheus.Collector, err error) {
	collectors = make([]prometheus.Collector, 0, 2)

	RequestsHandled = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: NamespaceHTTPMetrics,
		Name:      "requests_handled",
		Help:      "total requests handled, disaggregated by service, http status code, and path",
	}, []string{"service", "code", "path"})
	collectors = append(collectors, RequestsHandled)

	RequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: NamespaceHTTPMetrics,
		Name:      "request_duration",
		Help:      "duration of requests, disaggregated by service, http status code, and path",
	}, []string{"service", "code", "path"})
	collectors = append(collectors, RequestDuration)

	return collectors, nil
}
