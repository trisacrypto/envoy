package web

import (
	"net/http"
	"self-hosted-node/pkg"

	"github.com/gin-gonic/gin"
)

// If the server is in maintenance mode, aborts the current request and renders the
// maintenance mode page instead. Returns nil if not in maintenance mode.
func (s *Server) Maintenance() gin.HandlerFunc {
	if s.conf.Maintenance {
		return func(c *gin.Context) {
			c.HTML(http.StatusServiceUnavailable, "maintenance.html", gin.H{"Version": pkg.Version()})
			c.Abort()
		}
	}
	return nil
}
