package main

import (
	"fmt"
	"io"
	"net/http"
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
