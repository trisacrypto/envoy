package web

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/trisacrypto/envoy/pkg/web/scene"
)

//===========================================================================
// Transfer Policies REST Resource
//===========================================================================

func (s *Server) ListTransferPolicies(c *gin.Context) {
	c.HTML(http.StatusOK, "partials/transferpolicies/list.html", scene.New(c))
}

func (s *Server) CreateTransferPolicy(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
}

func (s *Server) TransferPolicyDetail(c *gin.Context) {
	c.HTML(http.StatusNotImplemented, "partials/transferpolicies/detail.html", scene.New(c))
}

func (s *Server) UpdateTransferPolicy(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
}

func (s *Server) DeleteTransferPolicy(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
}
