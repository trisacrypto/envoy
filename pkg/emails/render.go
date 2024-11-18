package emails

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"path/filepath"
)

const (
	// Email templates must be provided in this directory and are loaded at compile time
	templatesDir = "templates"

	// Partials are included when rendering templates for composability and reuse
	partialsDir = "partials"
)

var (
	//go:embed templates/*.html templates/*.txt templates/partials/*html
	files     embed.FS
	templates map[string]*template.Template
)

// Load templates when the package is imported
func init() {
	var (
		err           error
		templateFiles []fs.DirEntry
	)

	templates = make(map[string]*template.Template)
	if templateFiles, err = fs.ReadDir(files, templatesDir); err != nil {
		panic(err)
	}

	// Each template needs to be parsed independently to ensure that define directives
	// are not overriden if they have the same name; e.g. to use the base template.
	for _, file := range templateFiles {
		if file.IsDir() {
			continue
		}

		// Each template will be accessible by its base name in the global map
		patterns := make([]string, 0, 2)
		patterns = append(patterns, filepath.Join(templatesDir, file.Name()))
		switch filepath.Ext(file.Name()) {
		case ".html":
			patterns = append(patterns, filepath.Join(templatesDir, partialsDir, "*.html"))
		}

		templates[file.Name()] = template.Must(template.ParseFS(files, patterns...))
	}
}

// Render returns the text and html executed templates for the specified name and data.
// Ensure that the extension is not supplied to the render method.
func Render(name string, data interface{}) (text, html []byte, err error) {
	if text, err = render(name+".txt", data); err != nil {
		return nil, nil, err
	}

	if html, err = render(name+".html", data); err != nil {
		return nil, nil, err
	}

	return text, html, nil
}

func RenderString(name string, data interface{}) (text, html string, err error) {
	var (
		tb []byte
		hb []byte
	)

	if tb, hb, err = Render(name, data); err != nil {
		return "", "", nil
	}

	return string(tb), string(hb), nil
}

func render(name string, data interface{}) (_ []byte, err error) {
	t, ok := templates[name]
	if !ok {
		return nil, fmt.Errorf("could not find %q in templates", name)
	}

	buf := &bytes.Buffer{}
	if err = t.Execute(buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
