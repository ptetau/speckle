package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func runCommitCLI(t *testing.T, args ...string) {
	t.Helper()
	cmd := exec.Command(testBinary, append([]string{"commit"}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("speckle commit %v: %v\n%s", args, err, out)
	}
}

func TestCommitCreatesRepoAndManifest(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "plan.speckle")
	if err := os.WriteFile(specPath, []byte(minSpec), 0o644); err != nil {
		t.Fatal(err)
	}

	runCommitCLI(t, specPath)

	repoPath := filepath.Join(dir, ".speckle-repo")

	if _, err := os.Stat(filepath.Join(repoPath, ".git")); err != nil {
		t.Fatalf(".speckle-repo/.git not created: %v", err)
	}
	if _, err := os.Stat(filepath.Join(repoPath, ".speckle-meta.json")); err != nil {
		t.Fatalf(".speckle-meta.json not created: %v", err)
	}
	if _, err := os.Stat(filepath.Join(repoPath, "plan.speckle")); err != nil {
		t.Fatalf("plan.speckle not committed to repo: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".speckle-manifest.json")); err != nil {
		t.Fatalf(".speckle-manifest.json not created: %v", err)
	}

	assertCommitCountCLI(t, repoPath, 1)
}

func TestCommitWithDecisionsFlagWritesSidecar(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "plan.speckle")
	os.WriteFile(specPath, []byte(minSpec), 0o644)

	decisionsPath := filepath.Join(dir, "decisions.json")
	decisions := `{"spec_version":1,"decisions":{"d":{"selected":"a"}},"notes":"ship it"}`
	os.WriteFile(decisionsPath, []byte(decisions), 0o644)

	runCommitCLI(t, "--decisions", decisionsPath, specPath)

	repoPath := filepath.Join(dir, ".speckle-repo")
	sidecar := filepath.Join(repoPath, "plan.decisions.json")
	if _, err := os.Stat(sidecar); err != nil {
		t.Fatalf("plan.decisions.json not in repo: %v", err)
	}
	data, _ := os.ReadFile(sidecar)
	if string(data) != decisions {
		t.Fatalf("decisions mismatch: %s", data)
	}
}

func TestCommitReusesRepoOnSubsequentRuns(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "plan.speckle")
	os.WriteFile(specPath, []byte(minSpec), 0o644)

	runCommitCLI(t, specPath)
	runCommitCLI(t, specPath)

	repoPath := filepath.Join(dir, ".speckle-repo")
	assertCommitCountCLI(t, repoPath, 2)

	// Exactly one .speckle-repo directory created
	entries, _ := os.ReadDir(dir)
	repoCount := 0
	for _, e := range entries {
		if e.Name() == ".speckle-repo" {
			repoCount++
		}
	}
	if repoCount != 1 {
		t.Fatalf("expected 1 .speckle-repo, found %d", repoCount)
	}
}

func TestCommitManifestUsedOnReopen(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "plan.speckle")
	os.WriteFile(specPath, []byte(minSpec), 0o644)

	runCommitCLI(t, specPath)

	// Manifest must exist and reference the spec
	manifestData, err := os.ReadFile(filepath.Join(dir, ".speckle-manifest.json"))
	if err != nil {
		t.Fatal("no manifest")
	}
	if !strings.Contains(string(manifestData), "plan.speckle") {
		t.Fatalf("manifest doesn't reference spec: %s", manifestData)
	}
}

func assertCommitCountCLI(t *testing.T, repoPath string, want int) {
	t.Helper()
	cmd := exec.Command("git", "-C", repoPath, "log", "--oneline")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git log: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		lines = nil
	}
	if len(lines) != want {
		t.Fatalf("want %d commit(s), got %d:\n%s", want, len(lines), out)
	}
}
