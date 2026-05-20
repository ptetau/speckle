package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestPatchAppliesOverlayPreservingKeyOrder: the YAML overlay round-trip
// preserves the agent's authored key order, adds new id-keyed list items
// at the tail, and the merged file still validates.
func TestPatchAppliesOverlayPreservingKeyOrder(t *testing.T) {
	base := `version: 1
title: plan
sections:
  - id: s
    heading: First
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
notes: ""
`
	dir := t.TempDir()
	path := filepath.Join(dir, "plan.speckle")
	if err := os.WriteFile(path, []byte(base), 0o644); err != nil {
		t.Fatal(err)
	}

	overlay := `sections:
  - id: s
    decisions:
      - id: d
        default: b
        options:
          - id: c
            label: C
`
	cmd := exec.Command(testBinary, "patch", path)
	cmd.Stdin = strings.NewReader(overlay)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("patch: %v\nstderr: %s", err, stderr.String())
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	s := string(got)

	if !strings.HasPrefix(s, "version: 1\ntitle: plan\nsections:") {
		t.Errorf("top-level key order not preserved:\n%s", s)
	}
	if !strings.Contains(s, "default: b") {
		t.Errorf("default not flipped:\n%s", s)
	}
	if !strings.Contains(s, "id: c") || !strings.Contains(s, "label: C") {
		t.Errorf("new option not appended:\n%s", s)
	}
	// option order: existing first, new appended at the end.
	aIdx := strings.Index(s, "id: a")
	bIdx := strings.Index(s, "id: b")
	cIdx := strings.Index(s, "id: c")
	if !(aIdx < bIdx && bIdx < cIdx) {
		t.Errorf("expected a < b < c in file, got positions %d %d %d:\n%s", aIdx, bIdx, cIdx, s)
	}
}

// TestPatchRejectsOverlayThatBreaksValidation: if the resulting file
// would fail spec validation, patch refuses to overwrite.
func TestPatchRejectsOverlayThatBreaksValidation(t *testing.T) {
	base := `version: 1
title: plan
sections:
  - id: s
    heading: First
    decisions:
      - id: d
        prompt: p
        options:
          - id: a
            label: A
        default: a
        selected: null
`
	dir := t.TempDir()
	path := filepath.Join(dir, "plan.speckle")
	if err := os.WriteFile(path, []byte(base), 0o644); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(path)

	// removing the only option leaves a decision with zero options, which
	// parseSpec rejects.
	overlay := `sections:
  - id: s
    decisions:
      - id: d
        options:
          - id: a
            _delete: true
`
	cmd := exec.Command(testBinary, "patch", path)
	cmd.Stdin = strings.NewReader(overlay)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err == nil {
		t.Fatal("expected patch to fail on validation")
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Errorf("file was modified despite validation failure")
	}
}
