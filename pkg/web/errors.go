package web

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// TODO: return not found page
func (s *Server) NotFound(c *gin.Context) {
	c.String(http.StatusNotFound, http.StatusText(http.StatusNotFound))
}

// TODO: reeturn not allowed page
func (s *Server) NotAllowed(c *gin.Context) {
	c.String(http.StatusMethodNotAllowed, http.StatusText(http.StatusMethodNotAllowed))
}
