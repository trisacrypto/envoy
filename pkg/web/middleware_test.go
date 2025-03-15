package web_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/web"
)

func TestIsAPIRequest(t *testing.T) {
	tests := []struct {
		method  string
		path    string
		headers map[string]string
		assert  require.BoolAssertionFunc
	}{
		{http.MethodGet, "/v1/users", map[string]string{"Accept": "application/json"}, require.True},
		{http.MethodGet, "/v1/users", map[string]string{"Accept": "text/html"}, require.False},
		{http.MethodGet, "/v1/users", map[string]string{"Accept": "application/yaml"}, require.False},
		{http.MethodGet, "/v1/users", map[string]string{}, require.True},
		{http.MethodGet, "/v1/users", map[string]string{"Accept": "*/*"}, require.True},
		{http.MethodGet, "/v1/users", map[string]string{"Accept": "*/*", "HX-Request": "true"}, require.False},
		{http.MethodGet, "/v1/users", map[string]string{"Accept": "text/html", "HX-Request": "true"}, require.False},
		{http.MethodGet, "/v1/users", map[string]string{"Accept": "application/json", "HX-Request": "true"}, require.False},
		{http.MethodGet, "/v1/users", map[string]string{"Accept": "application/json", "HX-Request": "false"}, require.True},
		{http.MethodGet, "/users", map[string]string{"Accept": "application/json"}, require.False},
		{http.MethodGet, "/users", map[string]string{"Accept": "text/html"}, require.False},
		{http.MethodGet, "/users", map[string]string{"Accept": "*/*"}, require.False},
		{http.MethodGet, "/users", map[string]string{"Accept": "application/yaml"}, require.False},
		{http.MethodGet, "/users", map[string]string{"Accept": "application/json", "HX-Request": "true"}, require.False},
		{http.MethodGet, "/users", map[string]string{"Accept": "application/json", "HX-Request": "false"}, require.False},
	}

	for i, tc := range tests {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(tc.method, tc.path, nil)
		for k, v := range tc.headers {
			c.Request.Header.Set(k, v)
		}

		tc.assert(t, web.IsAPIRequest(c), "test case %d failed", i)
	}
}
