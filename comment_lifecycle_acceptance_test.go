package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const specWithComment = `version: 1
title: Comment lifecycle test
sections:
  - id: s1
    heading: Auth strategy
    decisions:
      - id: auth
        prompt: Which auth method?
        comment: "I think JWT is better because of simplicity"
        options:
          - id: jwt
            label: JWT
          - id: oauth
            label: OAuth
        selected: null
`

func TestPatchClearsCommentOnDecidedDecision(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "plan.speckle")
	if err := os.WriteFile(specPath, []byte(specWithComment), 0o644); err != nil {
		t.Fatal(err)
	}

	overlay := `sections:
  - id: s1
    decisions:
      - id: auth
        selected: jwt
`
	cmd := exec.Command(testBinary, "patch", specPath)
	cmd.Stdin = strings.NewReader(overlay)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("speckle patch: %v\n%s", err, out)
	}

	result, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(result)

	// The comment should be cleared since auth has a selected option
	if strings.Contains(content, "I think JWT is better") {
		t.Errorf("comment should be cleared after selection, but still present in:\n%s", excerpt(content))
	}
	// The selected value should be there
	if !strings.Contains(content, "selected: jwt") {
		t.Errorf("expected selected: jwt in patched spec, got:\n%s", excerpt(content))
	}
}

func TestPatchPreservesCommentOnUndecidedDecision(t *testing.T) {
	const specTwoDecisions = `version: 1
title: Comment lifecycle test
sections:
  - id: s1
    heading: Section
    decisions:
      - id: auth
        prompt: Auth method?
        comment: "Keep this comment - not decided yet"
        options:
          - id: jwt
            label: JWT
          - id: oauth
            label: OAuth
        selected: null
      - id: db
        prompt: Database?
        comment: "Also keep - not decided"
        options:
          - id: pg
            label: PostgreSQL
          - id: sq
            label: SQLite
        selected: null
`
	dir := t.TempDir()
	specPath := filepath.Join(dir, "plan.speckle")
	if err := os.WriteFile(specPath, []byte(specTwoDecisions), 0o644); err != nil {
		t.Fatal(err)
	}

	// Only select one decision
	overlay := `sections:
  - id: s1
    decisions:
      - id: auth
        selected: jwt
`
	cmd := exec.Command(testBinary, "patch", specPath)
	cmd.Stdin = strings.NewReader(overlay)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("speckle patch: %v\n%s", err, out)
	}

	result, _ := os.ReadFile(specPath)
	content := string(result)

	// auth comment should be cleared (decided)
	if strings.Contains(content, "Keep this comment - not decided yet") {
		t.Errorf("decided decision comment should be cleared:\n%s", excerpt(content))
	}
	// db comment should be preserved (not decided)
	if !strings.Contains(content, "Also keep - not decided") {
		t.Errorf("undecided decision comment should be preserved:\n%s", excerpt(content))
	}
}
