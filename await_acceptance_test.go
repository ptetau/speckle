package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestAwaitCLIDeliversSubmission: with `speckle serve` running and a
// pending submission queued, `speckle await <file>` reads the lockfile,
// fetches the submission, and prints it as indented JSON on stdout.
func TestAwaitCLIDeliversSubmission(t *testing.T) {
	url, specPath, stop := startServer(t, minSpec)
	defer stop()

	// Kick off await in the background; it should block until the user
	// submits, then print the JSON and exit 0.
	type result struct {
		stdout string
		err    error
	}
	done := make(chan result, 1)
	go func() {
		cmd := exec.Command(testBinary, "await", specPath)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			done <- result{err: errf("await failed: %v\nstderr: %s", err, stderr.String())}
			return
		}
		done <- result{stdout: stdout.String()}
	}()

	// Give await time to register on /await.
	time.Sleep(150 * time.Millisecond)

	payload := `{"spec_version":1,"decisions":{"d":{"selected":"b","comment":"ok"}},"notes":"ship"}`
	post(t, url+"/submit", payload, 204)

	select {
	case res := <-done:
		if res.err != nil {
			t.Fatal(res.err)
		}
		// Validate stdout is well-formed indented JSON containing what we sent.
		var got map[string]any
		if err := json.Unmarshal([]byte(res.stdout), &got); err != nil {
			t.Fatalf("stdout is not JSON: %v\nout: %s", err, res.stdout)
		}
		d := got["decisions"].(map[string]any)["d"].(map[string]any)
		if d["selected"] != "b" || d["comment"] != "ok" {
			t.Fatalf("decoded: %+v", got)
		}
		if got["notes"] != "ship" {
			t.Fatalf("notes: %v", got["notes"])
		}
		// Indented form should have line breaks; one-line output means SetIndent isn't applied.
		if !strings.Contains(res.stdout, "\n  ") {
			t.Errorf("expected indented JSON, got one line: %s", res.stdout)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("await never returned within 3s")
	}
}

// TestAwaitCLIErrorsWhenNoServerRunning: `speckle await <file>` without
// a server (no lockfile) fails with a message that points at the next
// step the user should take.
func TestAwaitCLIErrorsWhenNoServerRunning(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "plan.speckle")
	// File doesn't need to exist; lockfile lookup happens first.
	cmd := exec.Command(testBinary, "await", path)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = io.Discard
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected await to fail without a running server")
	}
	msg := stderr.String()
	// Error should mention locating the server and how to fix it.
	if !strings.Contains(msg, "locate server") {
		t.Errorf("error didn't mention locating the server: %q", msg)
	}
	if !strings.Contains(msg, "speckle serve") {
		t.Errorf("error didn't suggest `speckle serve`: %q", msg)
	}
}

// TestAwaitCLIRespectsExplicitURL: when --url is passed, await skips the
// lockfile dance and talks to the given URL directly.
func TestAwaitCLIRespectsExplicitURL(t *testing.T) {
	url, _, stop := startServer(t, minSpec)
	defer stop()

	// Path is irrelevant when --url is given — use a clearly nonexistent one.
	bogusPath := filepath.Join(t.TempDir(), "does-not-exist.speckle")

	done := make(chan error, 1)
	go func() {
		cmd := exec.Command(testBinary, "await", "--url="+url, bogusPath)
		cmd.Stdout = io.Discard
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			done <- errf("await failed: %v\nstderr: %s", err, stderr.String())
			return
		}
		done <- nil
	}()

	time.Sleep(150 * time.Millisecond)
	post(t, url+"/submit", `{"spec_version":1,"decisions":{}}`, 204)

	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("await --url didn't return within 3s")
	}
}
