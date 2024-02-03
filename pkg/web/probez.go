package web

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	serverStatusOK          = "ok"
	serverStatusNotReady    = "not ready"
	serverStatusUnhealthy   = "unhealthy"
	serverStatusMaintenance = "maintenance"
)

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
