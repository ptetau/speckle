package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestInboxAppendsToDimension(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "plan.speckle")
	if err := os.WriteFile(specPath, []byte(minSpec), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(testBinary, "inbox", specPath, "cli")
	cmd.Stdin = strings.NewReader("This is an idea for the CLI dimension")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("speckle inbox: %v\n%s", err, out)
	}

	result, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(result)
	if !strings.Contains(content, "inbox:") {
		t.Errorf("expected inbox: in spec, got:\n%s", excerpt(content))
	}
	if !strings.Contains(content, "This is an idea for the CLI dimension") {
		t.Errorf("expected inbox text in spec, got:\n%s", excerpt(content))
	}
}

func TestInboxAppendsToExisting(t *testing.T) {
	dir := t.TempDir()
	const specWithInbox = `version: 1
title: Inbox test
sections:
  - id: s1
    heading: Section
    decisions:
      - id: d1
        prompt: Choose?
        options:
          - id: a
            label: A
        selected: null
inbox:
  cli: "first entry"
`
	specPath := filepath.Join(dir, "plan.speckle")
	if err := os.WriteFile(specPath, []byte(specWithInbox), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(testBinary, "inbox", specPath, "cli")
	cmd.Stdin = strings.NewReader("second entry")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("speckle inbox: %v\n%s", err, out)
	}

	result, _ := os.ReadFile(specPath)
	content := string(result)
	if !strings.Contains(content, "first entry") {
		t.Errorf("should preserve existing inbox text, got:\n%s", excerpt(content))
	}
	if !strings.Contains(content, "second entry") {
		t.Errorf("should append new inbox text, got:\n%s", excerpt(content))
	}
}

func TestInboxUsageError(t *testing.T) {
	cmd := exec.Command(testBinary, "inbox")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected error for missing args, got none\n%s", out)
	}
}
