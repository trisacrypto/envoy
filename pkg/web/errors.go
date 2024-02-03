package web

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Renders the "not found page"
func (s *Server) NotFound(c *gin.Context) {
	c.HTML(http.StatusNotFound, "404.html", nil)
}

// Renders the "invalid action page"
func (s *Server) NotAllowed(c *gin.Context) {
	c.HTML(http.StatusMethodNotAllowed, "405.html", nil)
}
