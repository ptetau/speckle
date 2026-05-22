package main

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestHistoryEndpointReturnsCommits(t *testing.T) {
	requireGit(t)
	dir := t.TempDir()
	specPath := filepath.Join(dir, "plan.speckle")
	if err := os.WriteFile(specPath, []byte(minSpec), 0o644); err != nil {
		t.Fatal(err)
	}

	// Make a commit
	runCommitCLI(t, specPath)

	// Start server
	url, _, stop := startServerAtPath(t, specPath)
	defer stop()

	resp, err := http.Get(url + "/history")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /history: status %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var entries []map[string]interface{}
	if err := json.Unmarshal(body, &entries); err != nil {
		t.Fatalf("parse /history JSON: %v\nbody: %s", err, body)
	}
	if len(entries) == 0 {
		t.Fatalf("expected at least 1 history entry, got 0")
	}

	// Check structure of first entry
	first := entries[0]
	if _, ok := first["hash"]; !ok {
		t.Errorf("history entry missing 'hash' field: %v", first)
	}
	if _, ok := first["timestamp"]; !ok {
		t.Errorf("history entry missing 'timestamp' field: %v", first)
	}
	if _, ok := first["subject"]; !ok {
		t.Errorf("history entry missing 'subject' field: %v", first)
	}
}

func TestHistoryRefEndpointReturnsReadOnly(t *testing.T) {
	requireGit(t)
	dir := t.TempDir()
	specPath := filepath.Join(dir, "plan.speckle")
	if err := os.WriteFile(specPath, []byte(minSpec), 0o644); err != nil {
		t.Fatal(err)
	}

	runCommitCLI(t, specPath)

	url, _, stop := startServerAtPath(t, specPath)
	defer stop()

	// Get history to find a hash
	resp, err := http.Get(url + "/history")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	var entries []map[string]interface{}
	if err := json.Unmarshal(body, &entries); err != nil || len(entries) == 0 {
		t.Fatalf("need at least 1 history entry, got: %s", body)
	}

	hash := entries[0]["hash"].(string)
	resp2, err := http.Get(url + "/history/" + hash)
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		body2, _ := io.ReadAll(resp2.Body)
		t.Fatalf("GET /history/%s: status %d\n%s", hash, resp2.StatusCode, body2)
	}

	body2, _ := io.ReadAll(resp2.Body)
	html := string(body2)
	// Should contain "Viewing version" banner
	if !strings.Contains(html, "Viewing version") {
		t.Errorf("history view should contain 'Viewing version', got:\n%s", excerpt(html))
	}
}

func TestHistoryEndpointEmptyWithoutManifest(t *testing.T) {
	url, _, stop := startServer(t, minSpec)
	defer stop()

	resp, err := http.Get(url + "/history")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /history: status %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	// Should return empty JSON array
	var entries []interface{}
	if err := json.Unmarshal(body, &entries); err != nil {
		t.Fatalf("parse /history JSON: %v\nbody: %s", err, body)
	}
	if len(entries) != 0 {
		t.Fatalf("expected empty history when no manifest, got %d entries", len(entries))
	}
}

func TestCommitDigestFlag(t *testing.T) {
	requireGit(t)
	dir := t.TempDir()
	specPath := filepath.Join(dir, "plan.speckle")
	if err := os.WriteFile(specPath, []byte(minSpec), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(testBinary, "commit", "--digest", "sha256:abc123", specPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("speckle commit --digest: %v\n%s", err, out)
	}

	// Start server and check /history returns the digest
	url, _, stop := startServerAtPath(t, specPath)
	defer stop()

	resp, err := http.Get(url + "/history")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var entries []map[string]interface{}
	if err := json.Unmarshal(body, &entries); err != nil || len(entries) == 0 {
		t.Fatalf("need history entries: %s", body)
	}
	first := entries[0]
	if dig, ok := first["digest"]; !ok || dig != "sha256:abc123" {
		t.Errorf("expected digest 'sha256:abc123' in history entry, got: %v", first)
	}
}

// startServerAtPath starts the serve subcommand using an existing spec file
// (not the in-memory spec pattern used by startServer).
func startServerAtPath(t *testing.T, specPath string) (url, path string, stop func()) {
	t.Helper()
	return startServerWithContent(t, specPath, "")
}
