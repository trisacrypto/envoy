package trp

import (
	"github.com/gin-gonic/gin"
	"github.com/trisacrypto/envoy/pkg/logger"
	"github.com/trisacrypto/envoy/pkg/metrics"
	"github.com/trisacrypto/trisa/pkg/openvasp"
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
	}

	// Kubernetes liveness probes added before middleware.
	s.router.GET("/healthz", s.Healthz)
	s.router.GET("/livez", s.Healthz)
	s.router.GET("/readyz", s.Readyz)

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
	s.router.GET("/version", s.VerifyTRPHeaders, s.TRPVersion)
	s.router.GET("/uptime", s.VerifyTRPHeaders, s.Uptime)
	s.router.GET("/extensions", s.VerifyTRPHeaders, s.TRPExtensions)
	s.router.GET("/identity", s.VerifyTRPHeaders, s.Identity)

	// TRP Inquiry Routes
	inquiry := gin.WrapH(openvasp.TransferInquiry(s))
	s.router.POST("/transfers", inquiry)
	s.router.POST("/transfers/a/:accountID", inquiry)
	s.router.POST("/transfers/w/:walletID", inquiry)

	// TRP Confirmation Routes
	confirm := gin.WrapH(openvasp.TransferConfirmation(s))
	s.router.POST("/transfers/:envelopeID/confirm", confirm)

	// API Routes
	v1 := s.router.Group("/v1")
	{
		// Status/Heartbeat endpoint
		v1.GET("/status", s.Status)
	}

	return nil
}
