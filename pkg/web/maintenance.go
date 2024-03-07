package web

import (
	"net/http"
	"self-hosted-node/pkg"
	"self-hosted-node/pkg/web/api/v1"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

// If the server is in maintenance mode, aborts the current request and renders the
// maintenance mode page instead. Returns nil if not in maintenance mode.
func (s *Server) Maintenance() gin.HandlerFunc {
	if s.conf.Maintenance {
		return func(c *gin.Context) {
			c.Negotiate(http.StatusServiceUnavailable, gin.Negotiate{
				Offered: []string{binding.MIMEJSON, binding.MIMEHTML},
				Data: &api.StatusReply{
					Status:  "maintenance",
					Version: pkg.Version(),
					Uptime:  time.Since(s.started).String(),
				},
				HTMLName: "maintenance.html",
			})
			c.Abort()
		}
	}
	return nil
}
