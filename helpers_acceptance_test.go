package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"
)

// post sends a JSON body to url and fails the test if the response
// status doesn't match wantStatus.
func post(t *testing.T, url, jsonBody string, wantStatus int) {
	t.Helper()
	r, err := http.Post(url, "application/json", strings.NewReader(jsonBody))
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	defer r.Body.Close()
	if r.StatusCode != wantStatus {
		b, _ := io.ReadAll(r.Body)
		t.Fatalf("POST %s: status %d (want %d): %s", url, r.StatusCode, wantStatus, b)
	}
}

// errf builds an error using fmt.Errorf — convenience for goroutine
// channels where t.Fatal isn't usable.
func errf(format string, args ...any) error { return fmt.Errorf(format, args...) }

// requireGit skips the test if git is not installed.
func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
}

// excerpt returns the first 600 bytes of s, for use in test failure messages.
func excerpt(s string) string {
	if len(s) > 600 {
		return s[:600] + "…"
	}
	return s
}

// startServerWithContent launches the speckle binary with an existing spec file
// when specPath is non-empty and specYAML is empty, or writes specYAML to a
// temp file when specPath is empty. Returns the base URL, spec path, and a
// cleanup function.
func startServerWithContent(t *testing.T, specPath, specYAML string) (baseURL, path string, cleanup func()) {
	t.Helper()
	if specPath == "" {
		dir := t.TempDir()
		specPath = filepath.Join(dir, "plan.speckle")
		if err := os.WriteFile(specPath, []byte(specYAML), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cmd := exec.Command(testBinary, "serve", "--addr=127.0.0.1:0", specPath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = io.Discard
	if err := cmd.Start(); err != nil {
		t.Fatalf("start server: %v", err)
	}

	lockPath := specPath + ".lock"
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if data, err := os.ReadFile(lockPath); err == nil {
			var l struct {
				URL string `json:"url"`
			}
			if json.Unmarshal(data, &l) == nil && l.URL != "" {
				return l.URL, specPath, func() {
					if runtime.GOOS == "windows" {
						_ = cmd.Process.Kill()
					} else {
						_ = cmd.Process.Signal(syscall.SIGINT)
					}
					_ = cmd.Wait()
				}
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	_ = cmd.Process.Kill()
	_ = cmd.Wait()
	t.Fatalf("server didn't write lockfile within 2s\nstderr: %s", stderr.String())
	return "", "", nil
}
