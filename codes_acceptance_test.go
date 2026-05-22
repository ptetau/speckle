package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const specWithDims = `version: 1
title: Code test
dimensions:
  - id: cli
    label: CLI
    color: "#2c6fbb"
  - id: api
    label: API
    color: "#e08aa0"
sections:
  - id: auth-section
    heading: Authentication
    dimension: cli
    decisions:
      - id: auth-method
        prompt: How to authenticate?
        options:
          - id: jwt
            label: JWT
          - id: oauth
            label: OAuth
        selected: null
  - id: data-section
    heading: Data Layer
    dimension: api
    decisions:
      - id: db-choice
        prompt: Which database?
        options:
          - id: postgres
            label: PostgreSQL
          - id: sqlite
            label: SQLite
        selected: null
`

const overlayForCodes = `sections:
  - id: auth-section
    decisions:
      - id: auth-method
        selected: jwt
`

func TestPatchAssignsCodes(t *testing.T) {
	requireGit(t)
	dir := t.TempDir()
	specPath := filepath.Join(dir, "plan.speckle")
	if err := os.WriteFile(specPath, []byte(specWithDims), 0o644); err != nil {
		t.Fatal(err)
	}
	overlayPath := filepath.Join(dir, "overlay.yaml")
	if err := os.WriteFile(overlayPath, []byte(overlayForCodes), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(testBinary, "patch", specPath)
	overlayFile, err := os.Open(overlayPath)
	if err != nil {
		t.Fatal(err)
	}
	defer overlayFile.Close()
	cmd.Stdin = overlayFile
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("speckle patch: %v\n%s", err, out)
	}

	result, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(result)

	// Should contain CLI-NNN codes
	if !strings.Contains(content, "CLI-") {
		t.Errorf("expected CLI- codes in patched spec, got:\n%s", excerpt(content))
	}
	// Should contain API-NNN codes
	if !strings.Contains(content, "API-") {
		t.Errorf("expected API- codes in patched spec, got:\n%s", excerpt(content))
	}
}

func TestPatchPreservesExistingCodes(t *testing.T) {
	dir := t.TempDir()
	const specWithExistingCode = `version: 1
title: Code preserve test
dimensions:
  - id: cli
    label: CLI
    color: "#2c6fbb"
sections:
  - id: s1
    heading: Section One
    dimension: cli
    decisions:
      - id: d1
        prompt: Choose?
        options:
          - id: opt1
            label: Option 1
            code: CLI-001
        selected: null
`
	specPath := filepath.Join(dir, "plan.speckle")
	if err := os.WriteFile(specPath, []byte(specWithExistingCode), 0o644); err != nil {
		t.Fatal(err)
	}

	overlay := `sections:
  - id: s1
    decisions:
      - id: d1
        selected: opt1
`
	cmd := exec.Command(testBinary, "patch", specPath)
	cmd.Stdin = strings.NewReader(overlay)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("speckle patch: %v\n%s", err, out)
	}

	result, _ := os.ReadFile(specPath)
	content := string(result)
	// The existing code CLI-001 should still be there (for the option)
	if !strings.Contains(content, "CLI-001") {
		t.Errorf("existing code CLI-001 should be preserved, got:\n%s", excerpt(content))
	}
	// Verify the option that already had CLI-001 still has CLI-001 (not reassigned)
	if strings.Contains(content, "label: Option 1\n") {
		// Find if the option has code: CLI-001 nearby by checking the spec has it
		if !strings.Contains(content, "code: CLI-001") {
			t.Errorf("option with existing CLI-001 code should keep it:\n%s", excerpt(content))
		}
	}
}
