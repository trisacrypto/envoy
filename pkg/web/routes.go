package web

import (
	"embed"
	"net/http"
	"self-hosted-node/pkg/logger"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

//go:embed all:static
var staticFiles embed.FS

//go:embed all:templates
var templateFiles embed.FS

// Sets up the server's middleware and routes.
func (s *Server) setupRoutes() error {
	// Template Handling

	// Create CORS configuration
	corsConf := cors.Config{
		AllowMethods:     []string{"GET", "HEAD"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization", "X-CSRF-TOKEN"},
		AllowOrigins:     []string{s.conf.Origin},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}

	// Application Middleware
	// NOTE: ordering is important to how middleware is handled
	middlewares := []gin.HandlerFunc{
		// Logging should be on the outside so we can record the correct latency of requests
		// NOTE: logging panics will not recover
		logger.GinLogger("web"),

		// Panic recovery middleware
		gin.Recovery(),

		// CORS configuration allows the front-end to make cross-origin requests
		cors.New(corsConf),

		// TODO: add maintenance mode handler
	}

	// Add the middleware to the router
	for _, middleware := range middlewares {
		if middleware != nil {
			s.router.Use(middleware)
		}
	}

	// Kubernetes liveness probes
	s.router.GET("/healthz", s.Healthz)
	s.router.GET("/livez", s.Healthz)
	s.router.GET("/readyz", s.Readyz)

	// NotFound and NotAllowed routes
	s.router.NoRoute(s.NotFound)
	s.router.NoMethod(s.NotAllowed)

	// Static Files
	s.router.StaticFS("/static", http.FS(staticFiles))

	return nil
}
