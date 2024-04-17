package web

import (
	"fmt"
	"net/http"
	"net/http/httputil"

	api "github.com/trisacrypto/envoy/pkg/web/api/v1"

	"github.com/gin-gonic/gin"
)

func (s *Server) Debug(c *gin.Context) {
	dump, err := httputil.DumpRequest(c.Request, true)
	if err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	fmt.Printf("\n\n=====================\n\n%s\n\n=====================\n\n", string(dump))
	c.Data(http.StatusOK, "text/plain", dump)
}
