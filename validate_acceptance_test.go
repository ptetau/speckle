package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestValidateValidSpec(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "plan.speckle")
	if err := os.WriteFile(specPath, []byte(minSpec), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(testBinary, "validate", specPath)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("speckle validate: %v\nstdout: %s", err, out)
	}

	var result struct {
		Valid bool `json:"valid"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("output not valid JSON: %v\nraw: %s", err, out)
	}
	if !result.Valid {
		t.Errorf("expected valid:true for valid spec, got: %s", out)
	}
}

func TestValidateInvalidSpec(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "bad.speckle")
	// Missing version field -> version=0 -> unsupported version
	if err := os.WriteFile(specPath, []byte("title: bad\nsections: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(testBinary, "validate", specPath)
	out, err := cmd.Output()
	// Should exit 1
	if err == nil {
		t.Fatalf("expected exit 1 for invalid spec, got success; stdout: %s", out)
	}

	var result struct {
		Valid  bool `json:"valid"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("output not valid JSON: %v\nraw: %s", err, out)
	}
	if result.Valid {
		t.Errorf("expected valid:false for invalid spec, got: %s", out)
	}
	if len(result.Errors) == 0 {
		t.Errorf("expected errors array, got empty; output: %s", out)
	}
	for _, e := range result.Errors {
		if e.Message == "" {
			t.Errorf("error entry has empty message: %s", out)
		}
	}
}

func TestValidateFileNotFound(t *testing.T) {
	cmd := exec.Command(testBinary, "validate", "/nonexistent/path/plan.speckle")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	// Should print to stderr, not stdout (nothing to JSON decode on stdout).
	// Non-zero exit (asserted above via err != nil) is sufficient.
	_ = out
}

func TestValidateRequiresExactlyOneArg(t *testing.T) {
	cmd := exec.Command(testBinary, "validate")
	if err := cmd.Run(); err == nil {
		t.Fatal("expected error with no args")
	}
}
