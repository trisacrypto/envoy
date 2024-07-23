package trp

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/trisacrypto/envoy/pkg"
	"github.com/trisacrypto/envoy/pkg/trp/api/v1"
)

const (
	serverStatusOK          = "ok"
	serverStatusNotReady    = "not ready"
	serverStatusUnhealthy   = "unhealthy"
	serverStatusMaintenance = "maintenance"
)

// Status reports the version and uptime of the server
func (s *Server) Status(c *gin.Context) {
	var state string
	s.RLock()
	switch {
	case s.healthy && s.ready:
		state = serverStatusOK
	case s.healthy && !s.ready:
		state = serverStatusNotReady
	case !s.healthy:
		state = serverStatusUnhealthy
	}
	s.RUnlock()

	c.JSON(http.StatusOK, &api.StatusReply{
		Status:  state,
		Version: pkg.Version(),
		Uptime:  time.Since(s.started).String(),
	})
}

// If the server is in maintenance mode, aborts the current request and renders the
// maintenance mode page instead. Returns nil if not in maintenance mode.
func (s *Server) Maintenance() gin.HandlerFunc {
	if s.conf.Maintenance {
		return func(c *gin.Context) {
			c.JSON(http.StatusServiceUnavailable, &api.StatusReply{
				Status:  serverStatusMaintenance,
				Version: pkg.Version(),
				Uptime:  time.Since(s.started).String(),
			})
			c.Abort()
		}
	}
	return nil
}

// Healthz is used to alert k8s to the health/liveness status of the server.
func (s *Server) Healthz(c *gin.Context) {
	s.RLock()
	healthy := s.healthy
	s.RUnlock()

	if !healthy {
		c.Data(http.StatusServiceUnavailable, "text/plain", []byte(serverStatusUnhealthy))
		return
	}

	c.Data(http.StatusOK, "text/plain", []byte(serverStatusOK))
}

// Readyz is used to alert k8s to the readiness status of the server.
func (s *Server) Readyz(c *gin.Context) {
	s.RLock()
	ready := s.ready
	s.RUnlock()

	if !ready {
		c.Data(http.StatusServiceUnavailable, "text/plain", []byte(serverStatusNotReady))
		return
	}

	c.Data(http.StatusOK, "text/plain", []byte(serverStatusOK))
}
