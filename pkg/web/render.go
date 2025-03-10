package web

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin/render"
	"github.com/rs/zerolog/log"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

//go:embed all:static
//go:embed all:templates
var content embed.FS

const (
	partials           = "partials/**/*.html"
	partialsComponents = "partials/components/**/*.html"
)

var (
	includes = []string{"*.html", "components/*.html"}
	excludes = map[string]struct{}{
		"partials":   {},
		"components": {},
	}
)

// Creates a new template renderer from the default templates.
// Templates should be stored in the "templates" directory and organized as follows:
// Any sub-templates that need to be included with other templates should be added to
// the includes variable above (e.g. components). Partials for HTMX rendering should be
// stored in the partials directory. All other templates should be stored in named
// directories. All templates will include base.html and any html files in the root of
// the subdirectory. Each template file in a sub-sub directory will be treated as
// independent and will not include the templates in the same sub-sub directory or
// sibling directories.
//
// For example, if we have a tempalate in dashboards/transactions/list.html; the parsed
// templates will include *.html, components/*.html, modals/*.html, dashboards/*.html,
// and dashboards/transactions/list.html.
//
// Specify the template required by its path relative to the template directory.
func NewRender(fsys fs.FS) (render *Render, err error) {
	render = &Render{
		templates: make(map[string]*template.Template),
	}

	var entries []fs.DirEntry
	if entries, err = fs.ReadDir(fsys, "."); err != nil {
		return nil, err
	}

	for _, entry := range entries {
		// Skip any excluded directories.
		name := entry.Name()
		if _, ok := excludes[name]; ok || !entry.IsDir() {
			continue
		}

		// Create the includes patterns for the layout
		pattern := fmt.Sprintf("%s/**/*.html", name)
		patternInclude := make([]string, 0, len(includes)+2)
		patternInclude = append(patternInclude, includes...)

		if components := fmt.Sprintf("%s/components/*.html", name); globExists(fsys, components) {
			patternInclude = append(patternInclude, components)
		}

		// Ensure the current layout template is last in the list of templates
		patternInclude = append(patternInclude, fmt.Sprintf("%s/*.html", name))

		// Add the templates to the renderer.
		if err = render.AddPattern(fsys, pattern, patternInclude...); err != nil {
			return nil, err
		}
	}

	// Add the partials to the templates.
	// Partials are independently rendered with other templates included.
	if err = render.AddPattern(fsys, partials, partialsComponents); err != nil {
		return nil, err
	}

	return render, nil
}

// Implements the render.HTMLRender interface for gin.
type Render struct {
	templates map[string]*template.Template
	funcs     template.FuncMap
}

var _ render.HTMLRender = &Render{}

func (r *Render) Instance(name string, data any) render.Render {
	return &render.HTML{
		Template: r.templates[name],
		Name:     filepath.Base(name),
		Data:     data,
	}
}

func (r *Render) AddPattern(fsys fs.FS, pattern string, includes ...string) (err error) {
	var names []string
	if names, err = fs.Glob(fsys, pattern); err != nil {
		return err
	}

	for _, name := range names {
		patterns := make([]string, 0, len(includes)+1)
		patterns = append(patterns, includes...)
		patterns = append(patterns, name)

		tmpl := template.New(name).Funcs(r.FuncMap())
		if r.templates[name], err = tmpl.ParseFS(fsys, patterns...); err != nil {
			return err
		}

		log.Trace().Str("template", name).Strs("patterns", patterns).Msg("parsed template")
	}
	return nil
}

func (r *Render) FuncMap() template.FuncMap {
	title := cases.Title(language.English)
	if r.funcs == nil {
		r.funcs = template.FuncMap{
			"uppercase": func(s string) string {
				return strings.ToUpper(s)
			},
			"lowercase": func(s string) string {
				return strings.ToLower(s)
			},
			"titlecase": func(s string) string {
				return title.String(string(s))
			},
		}
	}
	return r.funcs
}

func globExists(fsys fs.FS, pattern string) (exists bool) {
	names, _ := fs.Glob(fsys, pattern)
	return len(names) > 0
}
