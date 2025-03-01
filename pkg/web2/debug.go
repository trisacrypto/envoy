package web

import (
	"fmt"
	"net/http"
	"net/http/httputil"

	"github.com/rs/zerolog/log"
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

func (s *Server) Page(c *gin.Context) {
	r := s.router.HTMLRender.(*Render)
	for key, value := range r.templates {
		log.Info().Str("name", key).Str("defined", value.DefinedTemplates()).Msg("template")
	}

	c.HTML(http.StatusOK, "dashboard/debug/debug.html", nil)
}
