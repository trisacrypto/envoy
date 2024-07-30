package web

import (
	"io/fs"
	"net/http"
	"strings"
	"sync"
	"text/template"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/trisacrypto/envoy/pkg"
	api "github.com/trisacrypto/envoy/pkg/web/api/v1"
)

func (s *Server) OpenAPI(c *gin.Context) {
	var (
		err        error
		data       gin.H
		templates  *template.Template
		initialize sync.Once
	)

	initialize.Do(func() {
		data = gin.H{
			"Version":     pkg.Version(),
			"Origin":      s.conf.Web.Origin,
			"Description": "Envoy",
		}

		var files fs.FS
		if files, err = fs.Sub(content, "templates/openapi"); err != nil {
			log.Error().Err(err).Msg("could not load openapi templates from content embed")
			return
		}

		if templates, err = template.ParseFS(files, "*.json", "*.yaml"); err != nil {
			log.Error().Err(err).Msg("could not parse openapi templates from fs")
			return
		}
	})

	if err != nil {
		c.AbortWithError(http.StatusServiceUnavailable, err)
		return
	}

	switch strings.ToLower(c.Param("ext")) {
	case "json":
		templates.ExecuteTemplate(c.Writer, "openapi.json", data)
	case "yaml":
		templates.ExecuteTemplate(c.Writer, "openapi.yaml", data)
	default:
		c.JSON(http.StatusNotFound, api.Error("no openapi resource with the specified extension exists"))
	}
}
