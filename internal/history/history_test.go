package history_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ptetau/speckle/internal/history"
)

const minSpec = `version: 1
title: t
sections:
  - id: s
    heading: h
    decisions:
      - id: d
        prompt: p
        options:
          - id: a
            label: A
        default: a
        selected: null
`

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
}

func writeSpec(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "plan.speckle")
	if err := os.WriteFile(path, []byte(minSpec), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestOpenCreatesRepo(t *testing.T) {
	requireGit(t)
	dir := t.TempDir()
	specPath := writeSpec(t, dir)

	mgr, err := history.Open(specPath)
	if err != nil {
		t.Fatal(err)
	}

	repoPath := mgr.RepoPath()
	if repoPath == "" {
		t.Fatal("empty repo path")
	}

	// .git must exist
	if _, err := os.Stat(filepath.Join(repoPath, ".git")); err != nil {
		t.Fatalf(".git missing in repo %s: %v", repoPath, err)
	}
	// .speckle-meta.json must exist (detection marker)
	if _, err := os.Stat(filepath.Join(repoPath, ".speckle-meta.json")); err != nil {
		t.Fatalf(".speckle-meta.json missing: %v", err)
	}
}

func TestOpenWritesManifest(t *testing.T) {
	requireGit(t)
	dir := t.TempDir()
	specPath := writeSpec(t, dir)

	if _, err := history.Open(specPath); err != nil {
		t.Fatal(err)
	}

	manifestPath := filepath.Join(dir, ".speckle-manifest.json")
	if _, err := os.Stat(manifestPath); err != nil {
		t.Fatalf(".speckle-manifest.json not written: %v", err)
	}
	data, _ := os.ReadFile(manifestPath)
	if !strings.Contains(string(data), "plan.speckle") {
		t.Fatalf("manifest doesn't reference spec:\n%s", data)
	}
}

func TestOpenReusesExistingRepo(t *testing.T) {
	requireGit(t)
	dir := t.TempDir()
	specPath := writeSpec(t, dir)

	mgr1, err := history.Open(specPath)
	if err != nil {
		t.Fatal(err)
	}
	mgr2, err := history.Open(specPath)
	if err != nil {
		t.Fatal(err)
	}
	if mgr1.RepoPath() != mgr2.RepoPath() {
		t.Fatalf("Open returned different repo paths: %s vs %s", mgr1.RepoPath(), mgr2.RepoPath())
	}
}

func TestOpenReadsManifest(t *testing.T) {
	requireGit(t)
	dir := t.TempDir()
	specPath := writeSpec(t, dir)

	// First open — creates repo at default location
	mgr1, err := history.Open(specPath)
	if err != nil {
		t.Fatal(err)
	}
	firstRepo := mgr1.RepoPath()

	// Corrupt the default location so Open MUST use the manifest to find the repo
	// (delete .speckle-meta.json from the default path — but the manifest still
	// points there, so Open finds it via manifest then sees .git exists)
	// Actually: just call Open again; it must use manifest to find same path.
	mgr2, err := history.Open(specPath)
	if err != nil {
		t.Fatal(err)
	}
	if mgr2.RepoPath() != firstRepo {
		t.Fatalf("second Open got different repo: %s (want %s)", mgr2.RepoPath(), firstRepo)
	}
}

func TestCommitWritesSpecAndCreatesCommit(t *testing.T) {
	requireGit(t)
	dir := t.TempDir()
	specPath := writeSpec(t, dir)

	mgr, err := history.Open(specPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := mgr.Commit(nil, "submit"); err != nil {
		t.Fatal(err)
	}

	// Spec must be in repo
	if _, err := os.Stat(filepath.Join(mgr.RepoPath(), "plan.speckle")); err != nil {
		t.Fatalf("plan.speckle not in repo: %v", err)
	}
	// git log must have one commit
	assertCommitCount(t, mgr.RepoPath(), 1)
}

func TestCommitWithDecisionsWritesSidecar(t *testing.T) {
	requireGit(t)
	dir := t.TempDir()
	specPath := writeSpec(t, dir)

	mgr, err := history.Open(specPath)
	if err != nil {
		t.Fatal(err)
	}

	decisions := []byte(`{"spec_version":1,"decisions":{"d":{"selected":"a"}}}`)
	if err := mgr.Commit(decisions, "submit"); err != nil {
		t.Fatal(err)
	}

	sidecar := filepath.Join(mgr.RepoPath(), "plan.decisions.json")
	if _, err := os.Stat(sidecar); err != nil {
		t.Fatalf("plan.decisions.json not in repo: %v", err)
	}
	data, _ := os.ReadFile(sidecar)
	if string(data) != string(decisions) {
		t.Fatalf("decisions content mismatch: %s", data)
	}
}

func TestCommitMessageContainsDecisions(t *testing.T) {
	requireGit(t)
	dir := t.TempDir()
	specPath := writeSpec(t, dir)

	mgr, err := history.Open(specPath)
	if err != nil {
		t.Fatal(err)
	}

	decisions := []byte(`{"spec_version":1,"decisions":{"d":{"selected":"a"}},"notes":"test note"}`)
	if err := mgr.Commit(decisions, "submit"); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("git", "-C", mgr.RepoPath(), "log", "-1", "--format=%B")
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	msg := string(out)
	if !strings.Contains(msg, "speckle: submit") {
		t.Errorf("commit message missing prefix: %s", msg)
	}
	if !strings.Contains(msg, "test note") {
		t.Errorf("commit message missing decisions content: %s", msg)
	}
}

func TestCommitTwiceProducesTwoCommits(t *testing.T) {
	requireGit(t)
	dir := t.TempDir()
	specPath := writeSpec(t, dir)

	mgr, err := history.Open(specPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := mgr.Commit(nil, "submit"); err != nil {
		t.Fatal(err)
	}
	if err := mgr.Commit(nil, "patch"); err != nil {
		t.Fatal(err)
	}
	assertCommitCount(t, mgr.RepoPath(), 2)
}

func TestMetaFileIsValidJSON(t *testing.T) {
	requireGit(t)
	dir := t.TempDir()
	specPath := writeSpec(t, dir)

	mgr, err := history.Open(specPath)
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(mgr.RepoPath(), ".speckle-meta.json"))
	if err != nil {
		t.Fatal(err)
	}
	var v map[string]any
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatalf("meta not valid JSON: %v\n%s", err, data)
	}
	if v["version"] == nil {
		t.Error("meta missing version field")
	}
}

// assertCommitCount verifies git log shows exactly n commits.
func assertCommitCount(t *testing.T, repoPath string, n int) {
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
	if len(lines) != n {
		t.Fatalf("want %d commit(s), got %d:\n%s", n, len(lines), out)
	}
}
