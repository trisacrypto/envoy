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
	c.HTML(http.StatusNotImplemented, "partials/transferpolicies/list.html", scene.New(c))
}
