package render_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ptetau/speckle/internal/render"
	"github.com/ptetau/speckle/internal/spec"
)

func TestRenderEmitsOptionsAndSandboxedPreviews(t *testing.T) {
	selected := "a"
	s := &spec.Spec{
		Version: 1,
		Title:   "T",
		Sections: []spec.Section{{
			ID: "s", Heading: "Heading", Body: "Some **prose**.",
			Decisions: []spec.Decision{{
				ID: "d", Prompt: "Pick",
				Default:  "a",
				Selected: &selected,
				Options: []spec.Option{
					{ID: "a", Label: "Alpha"},
					{ID: "b", Label: "Beta", Preview: &spec.Preview{
						Kind: "html", Body: `<button onclick="alert(1)">x</button>`,
					}},
					{ID: "c", Label: "Gamma", Preview: &spec.Preview{
						Kind: "code", Language: "go", Body: "fmt.Println(1)",
					}},
				},
			}},
		}},
	}

	var buf bytes.Buffer
	if err := render.NewRenderer().Render(&buf, s); err != nil {
		t.Fatal(err)
	}
	out := buf.String()

	if !strings.Contains(out, `<title>T</title>`) {
		t.Errorf("title not rendered: %s", excerpt(out))
	}
	if !strings.Contains(out, `Heading`) || !strings.Contains(out, `<strong>prose</strong>`) {
		t.Errorf("body not markdown-rendered: %s", excerpt(out))
	}
	// the selected option's radio is checked
	if !strings.Contains(out, `value="a" checked`) {
		t.Errorf("default-selected radio not checked: %s", excerpt(out))
	}
	// html preview gets sandbox="allow-scripts" and srcdoc-escaped content
	if !strings.Contains(out, `iframe sandbox="allow-scripts" srcdoc=`) {
		t.Errorf("html preview missing sandbox attribute: %s", excerpt(out))
	}
	// SECURITY: previews must not be granted allow-same-origin (would let
	// preview JS reach the parent page) or allow-top-navigation (would
	// let them redirect the browser away). The README promises this.
	for _, forbidden := range []string{"allow-same-origin", "allow-top-navigation", "allow-popups", "allow-forms"} {
		if strings.Contains(out, forbidden) {
			t.Errorf("iframe sandbox should not grant %q: %s", forbidden, excerpt(out))
		}
	}
	// raw script content must be escaped, not interpolated literally
	if strings.Contains(out, `onclick="alert(1)`) {
		t.Errorf("preview HTML leaked unescaped into attribute: %s", excerpt(out))
	}
	// code preview keeps the language hint on a code element
	if !strings.Contains(out, `<code class="lang-go">`) {
		t.Errorf("code preview missing language class: %s", excerpt(out))
	}
}

func excerpt(s string) string {
	if len(s) > 600 {
		return s[:600] + "…"
	}
	return s
}
