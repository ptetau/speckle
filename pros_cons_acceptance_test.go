package main

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

// TestProsCons_RenderedInHTML verifies that when an option has pros, cons,
// and/or a recommended flag, those are rendered in the served HTML.
func TestProsCons_RenderedInHTML(t *testing.T) {
	specYAML := `version: 1
title: Pros/Cons Test
sections:
  - id: s
    heading: A section
    decisions:
      - id: d
        prompt: Pick one
        options:
          - id: a
            label: Option A
            pros:
              - Fast startup
              - Small footprint
            cons:
              - Less mature
            recommended: true
          - id: b
            label: Option B
            pros:
              - Battle-tested
            cons:
              - Heavyweight
              - Slow cold start
        default: a
        selected: null
`
	url, _, stop := startServer(t, specYAML)
	defer stop()

	r, err := http.Get(url + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	if r.StatusCode != 200 {
		t.Fatalf("index: %d", r.StatusCode)
	}
	body, _ := io.ReadAll(r.Body)
	html := string(body)

	// Recommended badge must appear for option a.
	if !strings.Contains(html, "Recommended") {
		t.Errorf("rendered page missing 'Recommended' badge: %s", excerpt(html))
	}

	// Pros items for option a must appear.
	if !strings.Contains(html, "Fast startup") {
		t.Errorf("rendered page missing pro 'Fast startup': %s", excerpt(html))
	}
	if !strings.Contains(html, "Small footprint") {
		t.Errorf("rendered page missing pro 'Small footprint': %s", excerpt(html))
	}

	// Cons item for option a must appear.
	if !strings.Contains(html, "Less mature") {
		t.Errorf("rendered page missing con 'Less mature': %s", excerpt(html))
	}

	// Pros for option b must appear.
	if !strings.Contains(html, "Battle-tested") {
		t.Errorf("rendered page missing pro 'Battle-tested': %s", excerpt(html))
	}

	// Cons for option b must appear.
	if !strings.Contains(html, "Heavyweight") {
		t.Errorf("rendered page missing con 'Heavyweight': %s", excerpt(html))
	}
	if !strings.Contains(html, "Slow cold start") {
		t.Errorf("rendered page missing con 'Slow cold start': %s", excerpt(html))
	}
}

// TestProsCons_ExistingSpecUnchanged verifies that a spec without pros/cons/recommended
// fields continues to work exactly as before — no new elements rendered.
func TestProsCons_ExistingSpecUnchanged(t *testing.T) {
	url, _, stop := startServer(t, minSpec)
	defer stop()

	r, err := http.Get(url + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	if r.StatusCode != 200 {
		t.Fatalf("index: %d", r.StatusCode)
	}
	body, _ := io.ReadAll(r.Body)
	html := string(body)

	// Existing spec must still render its title and option labels.
	if !strings.Contains(html, "Minimal plan") {
		t.Errorf("title not rendered: %s", excerpt(html))
	}
	// No spurious recommended badge on a spec without it.
	if strings.Contains(html, "Recommended") {
		t.Errorf("unexpected 'Recommended' badge in spec without recommended field: %s", excerpt(html))
	}
}
