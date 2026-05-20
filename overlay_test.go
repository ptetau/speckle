package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// merge runs the same pipeline as `speckle patch`: parse both as nodes,
// merge, re-marshal. Returns the resulting YAML as a string.
func merge(t *testing.T, baseStr, overlayStr string) string {
	t.Helper()
	var baseDoc, overlayDoc yaml.Node
	if err := yaml.Unmarshal([]byte(baseStr), &baseDoc); err != nil {
		t.Fatalf("base yaml: %v", err)
	}
	if err := yaml.Unmarshal([]byte(overlayStr), &overlayDoc); err != nil {
		t.Fatalf("overlay yaml: %v", err)
	}
	merged := mergeOverlayNodes(baseDoc.Content[0], overlayDoc.Content[0])
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(merged); err != nil {
		t.Fatalf("encode: %v", err)
	}
	enc.Close()
	return strings.TrimRight(buf.String(), "\n")
}

func eq(t *testing.T, got, want string) {
	t.Helper()
	if strings.TrimRight(got, "\n") != strings.TrimRight(want, "\n") {
		t.Fatalf("merge result mismatch:\n--- got ---\n%s\n--- want ---\n%s\n", got, want)
	}
}

func TestMergeMapDeep(t *testing.T) {
	got := merge(t,
		"a: 1\nb:\n  c: 2\n  d: 3\n",
		"b:\n  c: 99\n  e: 4\n",
	)
	eq(t, got, "a: 1\nb:\n  c: 99\n  d: 3\n  e: 4")
}

func TestMergeNullDeletes(t *testing.T) {
	got := merge(t,
		"a: 1\nb: 2\nc: 3\n",
		"b: null\n",
	)
	eq(t, got, "a: 1\nc: 3")
}

func TestMergeListByID(t *testing.T) {
	got := merge(t, `
items:
  - id: a
    label: first
  - id: b
    label: second
`, `
items:
  - id: b
    label: updated
  - id: c
    label: new
`)
	eq(t, got, `items:
  - id: a
    label: first
  - id: b
    label: updated
  - id: c
    label: new`)
}

func TestMergeListDelete(t *testing.T) {
	got := merge(t, `
items:
  - id: a
  - id: b
  - id: c
`, `
items:
  - id: b
    _delete: true
`)
	eq(t, got, `items:
  - id: a
  - id: c`)
}

func TestMergePlainListReplaces(t *testing.T) {
	got := merge(t, "tags:\n  - one\n  - two\n  - three\n", "tags:\n  - four\n")
	eq(t, got, "tags:\n  - four")
}

// The chief reason for the node-based merge: preserve the agent's key
// order across patch rounds.
func TestMergePreservesKeyOrder(t *testing.T) {
	got := merge(t, `
version: 1
title: hello
sections:
  - id: one
    heading: First
    body: prose
`, `
sections:
  - id: one
    body: revised
`)
	eq(t, got, `version: 1
title: hello
sections:
  - id: one
    heading: First
    body: revised`)
}

func TestExampleSpecParses(t *testing.T) {
	b, err := os.ReadFile("examples/example.speckle")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parseSpec(b); err != nil {
		t.Fatalf("example.speckle does not parse: %v", err)
	}
}
