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

func TestExampleFileParses(t *testing.T) {
	b, err := os.ReadFile("../../examples/example.speckle")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := spec.NewParser().Parse(b); err != nil {
		t.Fatalf("example.speckle does not parse: %v", err)
	}
}
