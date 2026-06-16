package web

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"text/template"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg"
	"github.com/trisacrypto/envoy/pkg/config"
)

const docsProductBlurb = "The TRISA Envoy API allows users to interact with their TRISA open source Envoy node in a programmatic fashion."

func TestOpenAPITemplatesRenderVersionAndDescription(t *testing.T) {
	files, err := fs.Sub(content, "templates/docs/openapi")
	require.NoError(t, err)

	templates, err := template.ParseFS(files, "*.json", "*.yaml")
	require.NoError(t, err)

	data := map[string]string{
		keyVersion:     "9.9.9-test",
		keyOrigin:      "http://example.test",
		keyDescription: "Envoy API Docs",
		keyTitle:       "Envoy API Docs",
	}

	for _, name := range []string{"openapi.json", "openapi.yaml"} {
		t.Run(name, func(t *testing.T) {
			var buf strings.Builder
			require.NoError(t, templates.ExecuteTemplate(&buf, name, data))

			out := buf.String()
			require.Contains(t, out, "9.9.9-test")
			require.Contains(t, out, "Envoy API Docs")
			require.Contains(t, out, docsProductBlurb)
			require.NotContains(t, out, "{{ .Version }}")
			require.NotContains(t, out, "{{ .Description }}")

			if name == "openapi.json" {
				var spec struct {
					Info struct {
						Title       string `json:"title"`
						Summary     string `json:"summary"`
						Version     string `json:"version"`
						Description string `json:"description"`
					} `json:"info"`
				}
				require.NoError(t, json.Unmarshal([]byte(out), &spec))
				require.Equal(t, "Envoy API Docs", spec.Info.Title)
				require.Equal(t, "v9.9.9-test", spec.Info.Version)
				require.Contains(t, spec.Info.Summary, docsProductBlurb)
				require.Contains(t, spec.Info.Description, docsProductBlurb)
			}
		})
	}
}

func TestOpenAPIHandlerServesRenderedJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	conf := config.Config{
		Organization: "Envoy",
		Web: config.WebConfig{
			Origin:   "http://localhost:8000",
			DocsName: "Envoy API Docs",
		},
	}
	srv := &Server{conf: conf}

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/docs/openapi.json", nil)
	ctx.Params = gin.Params{{Key: "ext", Value: "json"}}

	srv.OpenAPI()(ctx)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), pkg.Version(false))
	require.Contains(t, rec.Body.String(), conf.Web.DocsName)
	require.Contains(t, rec.Body.String(), docsProductBlurb)
	require.NotContains(t, rec.Body.String(), "{{ .Version }}")
	require.NotContains(t, rec.Body.String(), "{{ .Description }}")
	require.NotContains(t, rec.Body.String(), "{{ .Title }}")
}

func TestAPIDocsHTMLRendersTitleLabel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	templateFiles, err := fs.Sub(content, "templates")
	require.NoError(t, err)
	render, err := NewRender(templateFiles)
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	require.NoError(t, render.Instance("docs/openapi/openapi.html", gin.H{
		keyTitle: "Envoy API Docs",
	}).Render(rec))

	out := rec.Body.String()
	require.Contains(t, out, `data-docs-label="Envoy API Docs"`)
	require.Contains(t, out, "dataset.docsLabel")
	require.Contains(t, out, "envoy-home-link")
	require.Contains(t, out, "onLoaded:")
	require.Contains(t, out, "/static/js/scalar/standalone.js")
	require.NotContains(t, out, "rapidoc")
	require.NotContains(t, out, "{{ .Title }}")
}
