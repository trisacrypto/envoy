package htmx

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/trisacrypto/envoy/pkg/web/api/v1"
)

// Only limits a route to HTMX requests.
func Only(c *gin.Context) {
	if !IsHTMXRequest(c) {
		c.AbortWithStatusJSON(http.StatusNotAcceptable, api.Error("endpoint requires an HTMX request"))
		return
	}
	c.Next()
}
