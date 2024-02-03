package web

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Home currently renders the primary landing page for the web ui.
// TODO: replace with dashboard or redirect as necessary.
func (s *Server) Home(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", nil)
}
