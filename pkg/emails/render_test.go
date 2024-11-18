package emails_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/emails"
)

func TestRender(t *testing.T) {
	names := allEmailTemplates(t)
	for _, name := range names {
		text, html, err := emails.Render(name, nil)
		require.NoError(t, err, "could not render %q", name)
		require.NotEmpty(t, text, "no text data returned for %q", name)
		require.NotEmpty(t, html, "no html data returned for %q", name)
	}
}

func TestRenderUnknown(t *testing.T) {
	_, _, err := emails.Render("foo", nil)
	require.EqualError(t, err, "could not find \"foo.txt\" in templates", "expected unknown template")
}

func allEmailTemplates(t *testing.T) []string {
	paths := make(map[string]struct{})
	ls, err := filepath.Glob("templates/*.*")
	require.NoError(t, err, "could not ls templates directory")

	for _, path := range ls {
		base := filepath.Base(path)
		base = strings.TrimSuffix(base, filepath.Ext(base))
		paths[base] = struct{}{}
	}

	out := make([]string, 0, len(paths))
	for path := range paths {
		out = append(out, path)
	}

	return out
}
