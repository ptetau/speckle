// Package render turns a *spec.Spec into the HTML page that
// `speckle serve` returns from GET /.
package render

import (
	"bytes"
	"embed"
	"html/template"
	"io"

	"github.com/yuin/goldmark"

	"github.com/ptetau/speckle/internal/spec"
)

// Renderer writes the HTML representation of a Spec to w.
type Renderer interface {
	Render(w io.Writer, s *spec.Spec) error
}

// NewRenderer parses the embedded template once and returns a Renderer
// that reuses it. Panics on a programming error in the template — that
// only ever fires during development.
func NewRenderer() Renderer {
	tpl := template.Must(
		template.New("page").Funcs(template.FuncMap{
			"renderMarkdown":   renderMarkdown,
			"selectedOption":   selectedOption,
			"findDimension":    findDimension,
			"isDecided":        isDecided,
			"isOpen":           isOpen,
			"hasOpenDecisions": hasOpenDecisions,
		}).ParseFS(embedded, "template.html"),
	)
	return &renderer{tpl: tpl}
}

//go:embed template.html
var embedded embed.FS

type renderer struct {
	tpl *template.Template
}

func (r *renderer) Render(w io.Writer, s *spec.Spec) error {
	return r.tpl.ExecuteTemplate(w, "template.html", s)
}

func renderMarkdown(s string) template.HTML {
	var buf bytes.Buffer
	if err := goldmark.Convert([]byte(s), &buf); err != nil {
		return template.HTML(template.HTMLEscapeString(s))
	}
	return template.HTML(buf.String())
}

func findDimension(id string, dims []spec.Dimension) *spec.Dimension {
	for i := range dims {
		if dims[i].ID == id {
			return &dims[i]
		}
	}
	return nil
}

func selectedOption(d spec.Decision) string {
	if d.Selected != nil && *d.Selected != "" {
		return *d.Selected
	}
	return d.Default
}

// isOpen returns true if the section has at least one decision without a Selected value.
func isOpen(sec spec.Section) bool {
	return !isDecided(sec)
}

// isDecided returns true if all decisions in the section have a non-empty Selected value.
func isDecided(sec spec.Section) bool {
	if len(sec.Decisions) == 0 {
		return false
	}
	for _, d := range sec.Decisions {
		if d.Selected == nil || *d.Selected == "" {
			return false
		}
	}
	return true
}

// hasOpenDecisions returns true if any section in the spec has at least one
// decision without a Selected value.
func hasOpenDecisions(sections []spec.Section) bool {
	for _, sec := range sections {
		for _, d := range sec.Decisions {
			if d.Selected == nil || *d.Selected == "" {
				return true
			}
		}
	}
	return false
}
