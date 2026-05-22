package spec_test

import (
	"os"
	"strings"
	"testing"

	"github.com/ptetau/speckle/internal/spec"
)

const validSpec = `version: 1
title: t
sections:
  - id: s
    heading: h
    decisions:
      - id: d
        prompt: p
        options:
          - id: a
            label: A
          - id: b
            label: B
        default: a
        selected: null
`

func parse(t *testing.T, src string) (*spec.Spec, error) {
	t.Helper()
	return spec.NewParser().Parse([]byte(src))
}

func TestValidSpec(t *testing.T) {
	s, err := parse(t, validSpec)
	if err != nil {
		t.Fatal(err)
	}
	if s.Title != "t" || len(s.Sections) != 1 {
		t.Fatalf("unexpected: %+v", s)
	}
}

func TestVersionMustBeOne(t *testing.T) {
	_, err := parse(t, "version: 2\ntitle: t\nsections: []\n")
	if err == nil || !strings.Contains(err.Error(), "version") {
		t.Fatalf("want version error, got: %v", err)
	}
}

func TestRejectsMissingSectionID(t *testing.T) {
	src := `version: 1
title: t
sections:
  - heading: h
    decisions:
      - id: d
        prompt: p
        options:
          - id: a
            label: A
`
	_, err := parse(t, src)
	if err == nil || !strings.Contains(err.Error(), "missing id") {
		t.Fatalf("want missing-id error, got: %v", err)
	}
}

func TestRejectsDuplicateSectionID(t *testing.T) {
	src := `version: 1
title: t
sections:
  - id: dup
    heading: a
    decisions:
      - { id: d, prompt: p, options: [{id: o, label: O}] }
  - id: dup
    heading: b
    decisions:
      - { id: d, prompt: p, options: [{id: o, label: O}] }
`
	_, err := parse(t, src)
	if err == nil || !strings.Contains(err.Error(), "duplicate section") {
		t.Fatalf("want duplicate-section error, got: %v", err)
	}
}

func TestRejectsDuplicateDecisionID(t *testing.T) {
	src := `version: 1
title: t
sections:
  - id: s
    heading: h
    decisions:
      - { id: dup, prompt: p, options: [{id: o, label: O}] }
      - { id: dup, prompt: p, options: [{id: o, label: O}] }
`
	_, err := parse(t, src)
	if err == nil || !strings.Contains(err.Error(), "duplicate decision") {
		t.Fatalf("want duplicate-decision error, got: %v", err)
	}
}

func TestRejectsDuplicateOptionID(t *testing.T) {
	src := `version: 1
title: t
sections:
  - id: s
    heading: h
    decisions:
      - id: d
        prompt: p
        options:
          - { id: dup, label: A }
          - { id: dup, label: B }
`
	_, err := parse(t, src)
	if err == nil || !strings.Contains(err.Error(), "duplicate option") {
		t.Fatalf("want duplicate-option error, got: %v", err)
	}
}

func TestRejectsDecisionWithNoOptions(t *testing.T) {
	src := `version: 1
title: t
sections:
  - id: s
    heading: h
    decisions:
      - id: d
        prompt: p
        options: []
`
	_, err := parse(t, src)
	if err == nil || !strings.Contains(err.Error(), "at least one option") {
		t.Fatalf("want no-options error, got: %v", err)
	}
}

func TestRejectsUnknownPreviewKind(t *testing.T) {
	src := `version: 1
title: t
sections:
  - id: s
    heading: h
    decisions:
      - id: d
        prompt: p
        options:
          - id: o
            label: O
            preview:
              kind: weird
              body: x
`
	_, err := parse(t, src)
	if err == nil || !strings.Contains(err.Error(), "unknown preview kind") {
		t.Fatalf("want preview-kind error, got: %v", err)
	}
}

func TestDimensionsParse(t *testing.T) {
	src := `version: 1
title: t
dimensions:
  - id: eng
    label: Engineering
    color: "#2c6fbb"
  - id: design
    label: Design
    color: "#c25a78"
sections:
  - id: s
    heading: h
    dimension: eng
    decisions:
      - id: d
        prompt: p
        options:
          - id: a
            label: A
`
	s, err := parse(t, src)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Dimensions) != 2 {
		t.Fatalf("want 2 dimensions, got %d", len(s.Dimensions))
	}
	if s.Dimensions[0].ID != "eng" || s.Dimensions[0].Color != "#2c6fbb" {
		t.Fatalf("unexpected dimension: %+v", s.Dimensions[0])
	}
	if s.Sections[0].Dimension != "eng" {
		t.Fatalf("section dimension not set: %+v", s.Sections[0])
	}
}

func TestDimensionsRejectsDuplicateID(t *testing.T) {
	src := `version: 1
title: t
dimensions:
  - id: dup
    label: A
    color: "#111"
  - id: dup
    label: B
    color: "#222"
sections:
  - id: s
    heading: h
    decisions:
      - id: d
        prompt: p
        options:
          - id: a
            label: A
`
	_, err := parse(t, src)
	if err == nil || !strings.Contains(err.Error(), "duplicate dimension") {
		t.Fatalf("want duplicate-dimension error, got: %v", err)
	}
}

func TestDimensionsRejectsUnknownReference(t *testing.T) {
	src := `version: 1
title: t
dimensions:
  - id: eng
    label: Engineering
    color: "#2c6fbb"
sections:
  - id: s
    heading: h
    dimension: unknown
    decisions:
      - id: d
        prompt: p
        options:
          - id: a
            label: A
`
	_, err := parse(t, src)
	if err == nil || !strings.Contains(err.Error(), "unknown dimension") {
		t.Fatalf("want unknown-dimension error, got: %v", err)
	}
}

func TestExampleFileParses(t *testing.T) {
	b, err := os.ReadFile("../../examples/example.speckle")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := spec.NewParser().Parse(b); err != nil {
		t.Fatalf("example.speckle does not parse: %v", err)
	}
}

func TestParseProsConsRecommended(t *testing.T) {
	src := `version: 1
title: t
sections:
  - id: s
    heading: h
    decisions:
      - id: d
        prompt: p
        options:
          - id: a
            label: A
            pros:
              - Fast startup
              - Small footprint
            cons:
              - Less mature
            recommended: true
          - id: b
            label: B
            pros:
              - Battle-tested
            cons:
              - Heavyweight
              - Slow cold start
`
	s, err := parse(t, src)
	if err != nil {
		t.Fatal(err)
	}
	opts := s.Sections[0].Decisions[0].Options
	if len(opts) != 2 {
		t.Fatalf("want 2 options, got %d", len(opts))
	}

	a := opts[0]
	if !a.Recommended {
		t.Errorf("option a: want Recommended=true")
	}
	if len(a.Pros) != 2 || a.Pros[0] != "Fast startup" || a.Pros[1] != "Small footprint" {
		t.Errorf("option a: unexpected pros: %v", a.Pros)
	}
	if len(a.Cons) != 1 || a.Cons[0] != "Less mature" {
		t.Errorf("option a: unexpected cons: %v", a.Cons)
	}

	b := opts[1]
	if b.Recommended {
		t.Errorf("option b: want Recommended=false")
	}
	if len(b.Pros) != 1 || b.Pros[0] != "Battle-tested" {
		t.Errorf("option b: unexpected pros: %v", b.Pros)
	}
	if len(b.Cons) != 2 || b.Cons[0] != "Heavyweight" || b.Cons[1] != "Slow cold start" {
		t.Errorf("option b: unexpected cons: %v", b.Cons)
	}
}

func TestParseInboxField(t *testing.T) {
	src := `version: 1
title: Inbox test
sections:
  - id: s
    heading: h
    decisions:
      - id: d
        prompt: p
        options:
          - id: a
            label: A
        selected: null
inbox:
  cli: "some idea"
  api: "another idea"
`
	s, err := parse(t, src)
	if err != nil {
		t.Fatal(err)
	}
	if s.Inbox == nil {
		t.Fatal("inbox should not be nil")
	}
	if s.Inbox["cli"] != "some idea" {
		t.Errorf("inbox[cli]: got %q, want %q", s.Inbox["cli"], "some idea")
	}
	if s.Inbox["api"] != "another idea" {
		t.Errorf("inbox[api]: got %q, want %q", s.Inbox["api"], "another idea")
	}
}

func TestParseDerivesDecisionSelectedFromOptionSelected(t *testing.T) {
	src := `version: 1
title: t
sections:
  - id: s
    heading: h
    decisions:
      - id: d
        prompt: p
        options:
          - id: a
            label: A
          - id: b
            label: B
            selected: true
`
	s, err := parse(t, src)
	if err != nil {
		t.Fatal(err)
	}
	d := s.Sections[0].Decisions[0]
	if d.Selected == nil || *d.Selected != "b" {
		t.Errorf("want Selected=b, got %v", d.Selected)
	}
}

func TestParseDecisionSelectedNotOverridden(t *testing.T) {
	src := `version: 1
title: t
sections:
  - id: s
    heading: h
    decisions:
      - id: d
        prompt: p
        selected: a
        options:
          - id: a
            label: A
          - id: b
            label: B
            selected: true
`
	s, err := parse(t, src)
	if err != nil {
		t.Fatal(err)
	}
	d := s.Sections[0].Decisions[0]
	if d.Selected == nil || *d.Selected != "a" {
		t.Errorf("want Selected=a (explicit wins), got %v", d.Selected)
	}
}

func TestParseProsConsOmitted(t *testing.T) {
	// A spec without pros/cons/recommended must parse cleanly.
	s, err := parse(t, validSpec)
	if err != nil {
		t.Fatal(err)
	}
	opt := s.Sections[0].Decisions[0].Options[0]
	if opt.Recommended {
		t.Errorf("want Recommended=false when omitted")
	}
	if len(opt.Pros) != 0 {
		t.Errorf("want empty Pros when omitted, got %v", opt.Pros)
	}
	if len(opt.Cons) != 0 {
		t.Errorf("want empty Cons when omitted, got %v", opt.Cons)
	}
}
