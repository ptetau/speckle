package main

import (
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"testing"
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
