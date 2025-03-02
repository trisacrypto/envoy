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

const (
	keyVersion     = "Version"
	keyOrigin      = "Origin"
	keyDescription = "Description"
)

// If this is an HTML request it renders the OpenAPI documentation page; otherwise if
// this is an API request, then it returns a list of the available endpoints in JSON.
func (s *Server) APIDocs(c *gin.Context) {
	c.HTML(http.StatusOK, "docs/openapi/openapi.html", gin.H{})
}

// Prepares and returns the OpenAPI spec in the requested format (JSON or YAML).
func (s *Server) OpenAPI() gin.HandlerFunc {
	var (
		err        error
		data       gin.H
		templates  *template.Template
		initialize sync.Once
	)

	// While we're not worried about concurrency issues here, we still use a sync.Once
	// to ensure that if there are any errors in the data and template initialization
	// processing stops and the error causes an abort handler to be returned.
	initialize.Do(func() {
		data = gin.H{
			keyVersion:     pkg.Version(false),
			keyOrigin:      s.conf.Web.Origin,
			keyDescription: s.conf.Web.DocsName,
		}

		if s.conf.Web.DocsName == "" {
			data[keyDescription] = s.conf.Organization
		}

		var files fs.FS
		if files, err = fs.Sub(content, "templates/docs/openapi"); err != nil {
			log.Error().Err(err).Msg("could not load openapi templates from content embed")
			return
		}

		if templates, err = template.ParseFS(files, "*.json", "*.yaml"); err != nil {
			log.Error().Err(err).Msg("could not parse openapi templates from fs")
			return
		}
	})

	if err != nil {
		// If we could not process the template files, then we return an error handler.
		return func(c *gin.Context) {
			c.AbortWithError(http.StatusServiceUnavailable, err)
		}
	}

	return func(c *gin.Context) {
		switch strings.ToLower(c.Param("ext")) {
		case "json":
			templates.ExecuteTemplate(c.Writer, "openapi.json", data)
		case "yaml":
			templates.ExecuteTemplate(c.Writer, "openapi.yaml", data)
		default:
			c.JSON(http.StatusNotFound, api.Error("no openapi resource with the specified extension exists"))
		}
	}
}
