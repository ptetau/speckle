package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

const specWithInboxContent = `version: 1
title: Process inbox test
dimensions:
  - id: cli
    label: CLI
    color: "#4A90D9"
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
  cli: "add --dry-run flag to patch"
`

func TestProcessInboxOutputsJSON(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "plan.speckle")
	if err := os.WriteFile(specPath, []byte(specWithInboxContent), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(testBinary, "process-inbox", specPath)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("speckle process-inbox: %v\n%s", err, out)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("output not valid JSON: %v\n%s", err, out)
	}
	inbox, ok := payload["inbox"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected inbox field in output, got: %v", payload)
	}
	if inbox["cli"] != "add --dry-run flag to patch" {
		t.Errorf("inbox.cli mismatch: %v", inbox["cli"])
	}
	if payload["title"] != "Process inbox test" {
		t.Errorf("title mismatch: %v", payload["title"])
	}
}

func TestProcessInboxEmptyExitsClean(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "plan.speckle")
	if err := os.WriteFile(specPath, []byte(minSpec), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(testBinary, "process-inbox", specPath)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("expected clean exit for empty inbox: %v\n%s", err, out)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("output not valid JSON: %v\n%s", err, out)
	}
	inbox := payload["inbox"]
	if inbox != nil {
		m, _ := inbox.(map[string]interface{})
		if len(m) != 0 {
			t.Errorf("expected empty inbox, got: %v", inbox)
		}
	}
}

func TestProcessInboxDoesNotModifySpec(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "plan.speckle")
	if err := os.WriteFile(specPath, []byte(specWithInboxContent), 0o644); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(specPath)

	cmd := exec.Command(testBinary, "process-inbox", specPath)
	if out, err := cmd.Output(); err != nil {
		t.Fatalf("process-inbox: %v\n%s", err, out)
	}

	after, _ := os.ReadFile(specPath)
	if string(before) != string(after) {
		t.Error("process-inbox must not modify the spec file")
	}
}

func TestProcessInboxUsageError(t *testing.T) {
	cmd := exec.Command(testBinary, "process-inbox")
	if err := cmd.Run(); err == nil {
		t.Fatal("expected error for missing args")
	}
}
