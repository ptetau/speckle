package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

const minSpec = `version: 1
title: Minimal plan
sections:
  - id: s
    heading: A section
    decisions:
      - id: d
        prompt: Pick one
        options:
          - id: a
            label: A
          - id: b
            label: B
        default: a
        selected: null
notes: ""
`

// startServer launches the built speckle binary serving the given YAML in
// a temp dir, returns its base URL once the lockfile lands, and a cleanup
// that signals the process and reaps it.
func startServer(t *testing.T, specYAML string) (string, func()) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "plan.speckle")
	if err := os.WriteFile(path, []byte(specYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(testBinary, "serve", "--addr=127.0.0.1:0", path)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = io.Discard
	if err := cmd.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}

	lockPath := path + ".lock"
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if data, err := os.ReadFile(lockPath); err == nil {
			var l struct {
				URL string `json:"url"`
			}
			if json.Unmarshal(data, &l) == nil && l.URL != "" {
				return l.URL, func() {
					_ = cmd.Process.Signal(syscall.SIGINT)
					_ = cmd.Wait()
				}
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	_ = cmd.Process.Kill()
	_ = cmd.Wait()
	t.Fatalf("server didn't write lockfile within 2s\nstderr: %s", stderr.String())
	return "", nil
}

// TestServeSubmitAwaitRoundTrip is the load-bearing acceptance test: the
// full agent loop in miniature. User POSTs to /submit; agent GETs /await;
// the same submission comes back.
func TestServeSubmitAwaitRoundTrip(t *testing.T) {
	url, stop := startServer(t, minSpec)
	defer stop()

	r, err := http.Get(url + "/")
	if err != nil {
		t.Fatal(err)
	}
	r.Body.Close()
	if r.StatusCode != 200 {
		t.Fatalf("index: %d", r.StatusCode)
	}

	type result struct {
		body map[string]any
		err  error
	}
	awaited := make(chan result, 1)
	go func() {
		resp, err := http.Get(url + "/await")
		if err != nil {
			awaited <- result{err: err}
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			b, _ := io.ReadAll(resp.Body)
			awaited <- result{err: &httpErr{resp.StatusCode, string(b)}}
			return
		}
		var body map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			awaited <- result{err: err}
			return
		}
		awaited <- result{body: body}
	}()

	// Give the awaiting goroutine a beat to actually start its request.
	time.Sleep(80 * time.Millisecond)

	payload := `{"spec_version":1,"decisions":{"d":{"selected":"b","comment":"because"}},"notes":"ship it"}`
	sr, err := http.Post(url+"/submit", "application/json", strings.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	sr.Body.Close()
	if sr.StatusCode != 204 {
		t.Fatalf("submit: %d", sr.StatusCode)
	}

	select {
	case res := <-awaited:
		if res.err != nil {
			t.Fatal(res.err)
		}
		decisions, ok := res.body["decisions"].(map[string]any)
		if !ok {
			t.Fatalf("no decisions: %+v", res.body)
		}
		d := decisions["d"].(map[string]any)
		if d["selected"] != "b" {
			t.Fatalf("selected: %v", d)
		}
		if d["comment"] != "because" {
			t.Fatalf("comment: %v", d)
		}
		if res.body["notes"] != "ship it" {
			t.Fatalf("notes: %v", res.body["notes"])
		}
	case <-time.After(2 * time.Second):
		t.Fatal("await didn't return within 2s")
	}
}

// TestServeRejectsInvalidSpec — the parser's validation surfaces through
// the CLI: serve must refuse to start on a malformed file.
func TestServeRejectsInvalidSpec(t *testing.T) {
	bad := `version: 1
title: bad
sections:
  - id: dup
    heading: a
    decisions:
      - id: d
        prompt: p
        options:
          - id: o
            label: O
  - id: dup
    heading: b
    decisions:
      - id: d
        prompt: p
        options:
          - id: o
            label: O
`
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.speckle")
	if err := os.WriteFile(path, []byte(bad), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(testBinary, "serve", "--addr=127.0.0.1:0", path)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err == nil {
		t.Fatal("expected serve to fail on duplicate section id")
	}
	if !strings.Contains(stderr.String(), "duplicate section") {
		t.Fatalf("stderr didn't mention duplicate: %q", stderr.String())
	}
}

type httpErr struct {
	code int
	body string
}

func (e *httpErr) Error() string { return e.body }
