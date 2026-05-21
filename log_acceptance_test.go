package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestLogNoHistory(t *testing.T) {
	requireGit(t)
	dir := t.TempDir()
	specPath := filepath.Join(dir, "plan.speckle")
	if err := os.WriteFile(specPath, []byte(minSpec), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(testBinary, "log", specPath)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("speckle log: %v\nstdout: %s", err, out)
	}
	if strings.TrimSpace(string(out)) != "no history" {
		t.Errorf("expected 'no history', got: %s", out)
	}
}

func TestLogShowsCommitsNewestFirst(t *testing.T) {
	requireGit(t)
	dir := t.TempDir()
	specPath := filepath.Join(dir, "plan.speckle")
	if err := os.WriteFile(specPath, []byte(minSpec), 0o644); err != nil {
		t.Fatal(err)
	}

	// Make two commits: first a patch, then a submit with decisions
	runCommitCLI(t, "--message", "patch", specPath)

	decisionsPath := filepath.Join(dir, "decisions.json")
	decisions := `{"spec_version":1,"decisions":{"d":{"selected":"a"}}}`
	if err := os.WriteFile(decisionsPath, []byte(decisions), 0o644); err != nil {
		t.Fatal(err)
	}
	runCommitCLI(t, "--decisions", decisionsPath, "--message", "submit", specPath)

	cmd := exec.Command(testBinary, "log", specPath)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("speckle log: %v\nstdout: %s", err, out)
	}

	lines := nonEmptyLines(string(out))
	if len(lines) != 2 {
		t.Fatalf("expected 2 log lines, got %d:\n%s", len(lines), out)
	}

	// Newest commit (submit) must appear first
	if !strings.Contains(lines[0], "submit") {
		t.Errorf("expected newest commit (submit) first, got: %s", lines[0])
	}
	if !strings.Contains(lines[1], "patch") {
		t.Errorf("expected older commit (patch) second, got: %s", lines[1])
	}
}

func TestLogLineFormat(t *testing.T) {
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
	runCommitCLI(t, "--decisions", decisionsPath, "--message", "submit", specPath)

	cmd := exec.Command(testBinary, "log", specPath)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("speckle log: %v\nstdout: %s", err, out)
	}

	lines := nonEmptyLines(string(out))
	if len(lines) != 1 {
		t.Fatalf("expected 1 log line, got %d:\n%s", len(lines), out)
	}
	line := lines[0]

	// Must contain a short hash (7 hex chars), ISO timestamp, and subject
	// Short hash: something like "abc1234"
	if len(line) < 7 {
		t.Fatalf("log line too short: %q", line)
	}

	// Must contain timestamp with T separator (ISO 8601)
	if !strings.Contains(line, "T") {
		t.Errorf("log line missing ISO timestamp: %q", line)
	}

	// For submit commits: must contain decisions summary (key→value)
	if !strings.Contains(line, "d") || !strings.Contains(line, "a") {
		t.Errorf("log line missing decisions summary: %q", line)
	}
}

func TestLogRequiresExactlyOneArg(t *testing.T) {
	cmd := exec.Command(testBinary, "log")
	if err := cmd.Run(); err == nil {
		t.Fatal("expected error with no args")
	}
}

func nonEmptyLines(s string) []string {
	var out []string
	for _, l := range strings.Split(strings.TrimSpace(s), "\n") {
		if strings.TrimSpace(l) != "" {
			out = append(out, l)
		}
	}
	return out
}
