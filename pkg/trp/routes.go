package trp

import (
	"github.com/gin-gonic/gin"
	"github.com/trisacrypto/envoy/pkg/logger"
	"github.com/trisacrypto/envoy/pkg/metrics"
)

func (s *Server) setupRoutes() error {
	// Application Middleware
	// NOTE: ordering is important to how middleware is handled
	middlewares := []gin.HandlerFunc{
		// Logging should be on the outside so we can record the correct latency of requests
		// NOTE: logging panics will not recover
		logger.GinLogger("trp"),

		// Panic recovery middleware
		gin.Recovery(),

		// Maintenance mode middleware to return unavailable
		s.Maintenance(),

		// TRP API version check is required for all routes
		APICheck,
	}

	// Kubernetes liveness probes added before middleware.
	s.router.GET("/healthz", s.Healthz)
	s.router.GET("/livez", s.Healthz)
	s.router.GET("/readyz", s.Readyz)
	s.router.GET("/status", s.Status)

	// Prometheus metrics handler added before middleware.
	// Note metrics will be served at /metrics
	metrics.Routes(s.router)

	// Add the middleware to the router
	for _, middleware := range middlewares {
		if middleware != nil {
			s.router.Use(middleware)
		}
	}

	// NotFound and NotAllowed routes
	s.router.NoRoute(s.NotFound)
	s.router.NoMethod(s.NotAllowed)

	// TRP Discoverability
	s.router.GET("/version", s.TRPVersion)
	s.router.GET("/uptime", s.Uptime)
	s.router.GET("/extensions", s.TRPExtensions)
	s.router.GET("/identity", s.Identity)

	// TRP Inquiry Routes
	s.router.POST("/transfers", VerifyTRPCore, s.Inquiry)
	s.router.POST("/transfers/a/:accountID", VerifyTRPCore, s.Inquiry)
	s.router.POST("/transfers/w/:walletID", VerifyTRPCore, s.Inquiry)

	// TRP Confirmation Routes
	s.router.POST("/transfers/:envelopeID/confirm", VerifyTRPCore, s.Confirmation)

	return nil
}
