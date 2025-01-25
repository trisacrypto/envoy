package web

import (
	"net/http"
	"time"

	"github.com/trisacrypto/envoy/pkg"
	"github.com/trisacrypto/envoy/pkg/store"
	"github.com/trisacrypto/envoy/pkg/web/api/v1"

	"github.com/gin-gonic/gin"
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

// DBInfo reports the database connection status and information if available,
// otherwise returns a 501 Not Implemented http error.
func (s *Server) DBInfo(c *gin.Context) {
	db, ok := s.store.(store.Stats)
	if !ok {
		c.JSON(http.StatusNotImplemented, api.Error("store does not implement stats"))
		return
	}

	// Render the database statistics
	stats := db.Stats()
	c.JSON(http.StatusOK, &api.DBInfo{
		MaxOpenConnections: stats.MaxOpenConnections,
		OpenConnections:    stats.OpenConnections,
		InUse:              stats.InUse,
		Idle:               stats.Idle,
		WaitCount:          stats.WaitCount,
		WaitDuration:       stats.WaitDuration.String(),
		MaxIdleClosed:      stats.MaxIdleClosed,
		MaxIdleTimeClosed:  stats.MaxIdleTimeClosed,
		MaxLifetimeClosed:  stats.MaxLifetimeClosed,
	})
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
