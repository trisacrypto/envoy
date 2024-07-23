package trp

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/trisacrypto/envoy/pkg/trp/api/v1"
)

// Returns a not found JSON response
func (s *Server) NotFound(c *gin.Context) {
	c.JSON(http.StatusNotFound, api.NotFound)
}

// Returns a method not allowed JSON response
func (s *Server) NotAllowed(c *gin.Context) {
	c.JSON(http.StatusMethodNotAllowed, api.NotAllowed)
}
