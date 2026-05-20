package main

import (
	"bytes"
	"embed"
	"html/template"
	"io"

	"github.com/yuin/goldmark"
)

//go:embed template.html
var embedded embed.FS

var tpl = template.Must(
	template.New("page").Funcs(template.FuncMap{
		"renderMarkdown": renderMarkdown,
		"selectedOption": selectedOption,
	}).ParseFS(embedded, "template.html"),
)

func renderHTML(w io.Writer, spec *Spec) error {
	return tpl.ExecuteTemplate(w, "template.html", spec)
}

func renderMarkdown(s string) template.HTML {
	var buf bytes.Buffer
	if err := goldmark.Convert([]byte(s), &buf); err != nil {
		return template.HTML(template.HTMLEscapeString(s))
	}
	return template.HTML(buf.String())
}

func selectedOption(d Decision) string {
	if d.Selected != nil && *d.Selected != "" {
		return *d.Selected
	}
	return d.Default
}
