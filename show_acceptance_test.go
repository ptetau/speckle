package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestShowSpecAtRef(t *testing.T) {
	requireGit(t)
	dir := t.TempDir()
	specPath := filepath.Join(dir, "plan.speckle")
	if err := os.WriteFile(specPath, []byte(minSpec), 0o644); err != nil {
		t.Fatal(err)
	}

	runCommitCLI(t, specPath)

	// Get the short hash of the only commit
	ref := gitShortHash(t, filepath.Join(dir, ".speckle-repo"), "HEAD")

	cmd := exec.Command(testBinary, "show", specPath, ref)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("speckle show: %v\nstdout: %s", err, out)
	}

	content := string(out)

	// Must have --- spec --- header
	if !strings.Contains(content, "--- spec ---") {
		t.Errorf("output missing '--- spec ---' header:\n%s", content)
	}

	// Spec content must appear
	if !strings.Contains(content, "version:") {
		t.Errorf("output missing spec content:\n%s", content)
	}
}

func TestShowSpecWithDecisionsAtRef(t *testing.T) {
	requireGit(t)
	dir := t.TempDir()
	specPath := filepath.Join(dir, "plan.speckle")
	if err := os.WriteFile(specPath, []byte(minSpec), 0o644); err != nil {
		t.Fatal(err)
	}

	decisionsPath := filepath.Join(dir, "decisions.json")
	decisions := `{"spec_version":1,"decisions":{"d":{"selected":"a"}}}`
	if err := os.WriteFile(decisionsPath, []byte(decisions), 0o644); err != nil {
		t.Fatal(err)
	}
	runCommitCLI(t, "--decisions", decisionsPath, specPath)

	ref := gitShortHash(t, filepath.Join(dir, ".speckle-repo"), "HEAD")

	cmd := exec.Command(testBinary, "show", specPath, ref)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("speckle show: %v\nstdout: %s", err, out)
	}

	content := string(out)

	// Must have both headers
	if !strings.Contains(content, "--- spec ---") {
		t.Errorf("output missing '--- spec ---' header:\n%s", content)
	}
	if !strings.Contains(content, "--- decisions ---") {
		t.Errorf("output missing '--- decisions ---' header:\n%s", content)
	}

	// Decisions JSON must appear
	if !strings.Contains(content, `"selected"`) {
		t.Errorf("output missing decisions JSON:\n%s", content)
	}
}

func TestShowNoDecisionsSectionWhenNoSidecar(t *testing.T) {
	requireGit(t)
	dir := t.TempDir()
	specPath := filepath.Join(dir, "plan.speckle")
	if err := os.WriteFile(specPath, []byte(minSpec), 0o644); err != nil {
		t.Fatal(err)
	}

	// Commit without decisions
	runCommitCLI(t, specPath)

	ref := gitShortHash(t, filepath.Join(dir, ".speckle-repo"), "HEAD")

	cmd := exec.Command(testBinary, "show", specPath, ref)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("speckle show: %v\nstdout: %s", err, out)
	}

	content := string(out)
	if strings.Contains(content, "--- decisions ---") {
		t.Errorf("output should not contain decisions section when no sidecar:\n%s", content)
	}
}

func TestShowInvalidRefFails(t *testing.T) {
	requireGit(t)
	dir := t.TempDir()
	specPath := filepath.Join(dir, "plan.speckle")
	if err := os.WriteFile(specPath, []byte(minSpec), 0o644); err != nil {
		t.Fatal(err)
	}

	runCommitCLI(t, specPath)

	cmd := exec.Command(testBinary, "show", specPath, "deadbeef")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected error for nonexistent ref, got success; output: %s", out)
	}
}

func TestShowRequiresTwoArgs(t *testing.T) {
	cmd := exec.Command(testBinary, "show")
	if err := cmd.Run(); err == nil {
		t.Fatal("expected error with no args")
	}

	dir := t.TempDir()
	specPath := filepath.Join(dir, "plan.speckle")
	cmd2 := exec.Command(testBinary, "show", specPath)
	if err := cmd2.Run(); err == nil {
		t.Fatal("expected error with only one arg")
	}
}

// gitShortHash returns the abbreviated commit hash for ref in repoPath.
func gitShortHash(t *testing.T, repoPath, ref string) string {
	t.Helper()
	cmd := exec.Command("git", "-C", repoPath, "rev-parse", "--short", ref)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git rev-parse --short %s: %v", ref, err)
	}
	return strings.TrimSpace(string(out))
}
