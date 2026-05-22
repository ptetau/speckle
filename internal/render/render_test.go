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

func TestRenderDimensionsLegendAndBadge(t *testing.T) {
	s := &spec.Spec{
		Version: 1,
		Title:   "T",
		Dimensions: []spec.Dimension{
			{ID: "eng", Label: "Engineering", Color: "#2c6fbb"},
			{ID: "design", Label: "Graphic Design", Color: "#c25a78"},
		},
		Sections: []spec.Section{
			{
				ID: "s1", Heading: "Auth", Dimension: "eng",
				Decisions: []spec.Decision{{
					ID: "d", Prompt: "P",
					Options: []spec.Option{{ID: "a", Label: "A"}},
				}},
			},
			{
				ID: "s2", Heading: "UI", Dimension: "design",
				Decisions: []spec.Decision{{
					ID: "d2", Prompt: "P",
					Options: []spec.Option{{ID: "a", Label: "A"}},
				}},
			},
		},
	}

	var buf bytes.Buffer
	if err := render.NewRenderer().Render(&buf, s); err != nil {
		t.Fatal(err)
	}
	out := buf.String()

	// Legend must contain both dimension labels.
	if !strings.Contains(out, "Engineering") {
		t.Errorf("dimension legend missing 'Engineering': %s", excerpt(out))
	}
	if !strings.Contains(out, "Graphic Design") {
		t.Errorf("dimension legend missing 'Graphic Design': %s", excerpt(out))
	}
	// Dimension colors must appear in rendered output.
	if !strings.Contains(out, "#2c6fbb") {
		t.Errorf("dimension color #2c6fbb not in output: %s", excerpt(out))
	}
	if !strings.Contains(out, "#c25a78") {
		t.Errorf("dimension color #c25a78 not in output: %s", excerpt(out))
	}
	// Sections with a dimension must carry the color so CSS can use it.
	if !strings.Contains(out, "s1") || !strings.Contains(out, "s2") {
		t.Errorf("sections not rendered: %s", excerpt(out))
	}
}

func TestRenderProsConsRecommended(t *testing.T) {
	s := &spec.Spec{
		Version: 1,
		Title:   "T",
		Sections: []spec.Section{{
			ID: "s", Heading: "H",
			Decisions: []spec.Decision{{
				ID: "d", Prompt: "P",
				Options: []spec.Option{
					{
						ID:          "a",
						Label:       "Option A",
						Pros:        []string{"Fast startup", "Small footprint"},
						Cons:        []string{"Less mature"},
						Recommended: true,
					},
					{
						ID:    "b",
						Label: "Option B",
						Pros:  []string{"Battle-tested"},
						Cons:  []string{"Heavyweight", "Slow cold start"},
					},
				},
			}},
		}},
	}

	var buf bytes.Buffer
	if err := render.NewRenderer().Render(&buf, s); err != nil {
		t.Fatal(err)
	}
	out := buf.String()

	// Recommended badge for option a.
	if !strings.Contains(out, "Recommended") {
		t.Errorf("missing Recommended badge: %s", excerpt(out))
	}

	// Pros for option a.
	if !strings.Contains(out, "Fast startup") {
		t.Errorf("missing pro 'Fast startup': %s", excerpt(out))
	}
	if !strings.Contains(out, "Small footprint") {
		t.Errorf("missing pro 'Small footprint': %s", excerpt(out))
	}

	// Cons for option a.
	if !strings.Contains(out, "Less mature") {
		t.Errorf("missing con 'Less mature': %s", excerpt(out))
	}

	// Pros for option b.
	if !strings.Contains(out, "Battle-tested") {
		t.Errorf("missing pro 'Battle-tested': %s", excerpt(out))
	}

	// Cons for option b.
	if !strings.Contains(out, "Heavyweight") {
		t.Errorf("missing con 'Heavyweight': %s", excerpt(out))
	}
	if !strings.Contains(out, "Slow cold start") {
		t.Errorf("missing con 'Slow cold start': %s", excerpt(out))
	}
}

func TestRenderNoProsConsNoExtraMarkup(t *testing.T) {
	// An option with no pros/cons/recommended must not produce the
	// badge or list elements at all — existing layout must be intact.
	s := &spec.Spec{
		Version: 1,
		Title:   "T",
		Sections: []spec.Section{{
			ID: "s", Heading: "H",
			Decisions: []spec.Decision{{
				ID: "d", Prompt: "P",
				Options: []spec.Option{
					{ID: "a", Label: "Plain"},
				},
			}},
		}},
	}

	var buf bytes.Buffer
	if err := render.NewRenderer().Render(&buf, s); err != nil {
		t.Fatal(err)
	}
	out := buf.String()

	if strings.Contains(out, "Recommended") {
		t.Errorf("unexpected Recommended badge when not set: %s", excerpt(out))
	}
	if strings.Contains(out, `class="option-pros"`) {
		t.Errorf("unexpected pros list when not set: %s", excerpt(out))
	}
	if strings.Contains(out, `class="option-cons"`) {
		t.Errorf("unexpected cons list when not set: %s", excerpt(out))
	}
}

func excerpt(s string) string {
	if len(s) > 600 {
		return s[:600] + "…"
	}
	return s
}
