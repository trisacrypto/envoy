package metrics

import (
	"fmt"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

const (
	NamespaceHTTPMetrics = "http_stats"
	NamespaceGRPCMetrics = "grpc_stats"
)

var (
	setup    sync.Once
	setupErr error
)

func Setup() error {
	setup.Do(func() {
		// Register the collectors
		setupErr = initCollectors()
	})
	return setupErr
}

func Routes(router *gin.Engine) {
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))
}

func initCollectors() (err error) {
	// Track all collectors to register at the end of the function.
	// When adding new collectors make sure to increase the capacity.
	collectors := make([]prometheus.Collector, 0, 8)

	var httpCollectors []prometheus.Collector
	if httpCollectors, err = initHTTPCollectors(); err != nil {
		return err
	}
	collectors = append(collectors, httpCollectors...)

	var grpcCollectors []prometheus.Collector
	if grpcCollectors, err = initGRPCCollectors(); err != nil {
		return err
	}
	collectors = append(collectors, grpcCollectors...)

	// Register the collectors
	registerCollectors(collectors)
	return nil
}

func registerCollectors(collectors []prometheus.Collector) {
	var err error
	// Register the collectors
	for _, collector := range collectors {
		if err = prometheus.Register(collector); err != nil {
			err = fmt.Errorf("cannot register %s", collector)
			log.Warn().Err(err).Msg("collector already registered")
		}
	}
}
