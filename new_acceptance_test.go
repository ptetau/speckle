package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ptetau/speckle/internal/spec"
)

func TestNewCreatesStarterSpec(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "plan.speckle")

	cmd := exec.Command(testBinary, "new", specPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("speckle new: %v\n%s", err, out)
	}

	data, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("file not created: %v", err)
	}

	// Must be parseable by spec.Parse
	p := spec.NewParser()
	if _, err := p.Parse(data); err != nil {
		t.Fatalf("starter spec is not valid: %v\ncontent:\n%s", err, data)
	}
}

func TestNewStarterSpecHasRequiredFields(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "plan.speckle")

	cmd := exec.Command(testBinary, "new", specPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("speckle new: %v\n%s", err, out)
	}

	data, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	// Must contain version, title, a section, a decision, two options, comments
	if !strings.Contains(content, "version:") {
		t.Error("starter spec missing 'version:' field")
	}
	if !strings.Contains(content, "title:") {
		t.Error("starter spec missing 'title:' field")
	}
	if !strings.Contains(content, "sections:") {
		t.Error("starter spec missing 'sections:' field")
	}
	if !strings.Contains(content, "decisions:") {
		t.Error("starter spec missing 'decisions:' field")
	}
	if !strings.Contains(content, "options:") {
		t.Error("starter spec missing 'options:' field")
	}
	// Must have at least two options
	optCount := strings.Count(content, "- id:")
	if optCount < 2 {
		t.Errorf("expected at least 2 options (- id:), got %d", optCount)
	}
	// Must have comment lines
	if !strings.Contains(content, "#") {
		t.Error("starter spec has no comment lines")
	}

	p := spec.NewParser()
	parsed, err := p.Parse(data)
	if err != nil {
		t.Fatalf("parsed spec error: %v", err)
	}
	if len(parsed.Sections) == 0 {
		t.Error("starter spec has no sections")
	}
	if len(parsed.Sections[0].Decisions) == 0 {
		t.Error("starter spec has no decisions in first section")
	}
	if len(parsed.Sections[0].Decisions[0].Options) < 2 {
		t.Errorf("starter spec first decision has fewer than 2 options: %d",
			len(parsed.Sections[0].Decisions[0].Options))
	}
}

func TestNewFailsIfFileAlreadyExists(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "plan.speckle")

	// Create the file first
	if err := os.WriteFile(specPath, []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(testBinary, "new", specPath)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected 'new' to fail when file already exists, but it succeeded")
	}
	// stderr should mention the file or "exists"
	combined := string(out)
	if !strings.Contains(strings.ToLower(combined), "exist") &&
		!strings.Contains(combined, specPath) {
		t.Errorf("expected error about existing file, got: %s", combined)
	}

	// Original file must be unchanged
	data, _ := os.ReadFile(specPath)
	if string(data) != "existing" {
		t.Errorf("existing file was overwritten: %s", data)
	}
}

func TestNewRequiresExactlyOneArg(t *testing.T) {
	cmd := exec.Command(testBinary, "new")
	if err := cmd.Run(); err == nil {
		t.Fatal("expected error with no args")
	}
}
