package server_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ptetau/speckle/internal/render"
	"github.com/ptetau/speckle/internal/server"
	"github.com/ptetau/speckle/internal/spec"
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
          - id: b
            label: B
        default: a
        selected: null
`

func newTestServer(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "plan.speckle")
	if err := os.WriteFile(path, []byte(minSpec), 0o644); err != nil {
		t.Fatal(err)
	}
	srv, err := server.New(server.Config{
		Path:     path,
		Addr:     "127.0.0.1:0",
		Parser:   spec.NewParser(),
		Renderer: render.NewRenderer(),
	})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts, path
}

func TestIndexRenders(t *testing.T) {
	ts, _ := newTestServer(t)
	r, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	if r.StatusCode != 200 {
		t.Fatalf("index: %d", r.StatusCode)
	}
}

func TestSubmitDeliversToAwait(t *testing.T) {
	ts, _ := newTestServer(t)

	type result struct {
		body map[string]any
		err  error
	}
	got := make(chan result, 1)
	go func() {
		resp, err := http.Get(ts.URL + "/await")
		if err != nil {
			got <- result{err: err}
			return
		}
		defer resp.Body.Close()
		var b map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&b); err != nil {
			got <- result{err: err}
			return
		}
		got <- result{body: b}
	}()

	time.Sleep(50 * time.Millisecond)
	payload := `{"spec_version":1,"decisions":{"d":{"selected":"b","comment":"k"}}}`
	res, err := http.Post(ts.URL+"/submit", "application/json", strings.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != 204 {
		t.Fatalf("submit: %d", res.StatusCode)
	}

	select {
	case r := <-got:
		if r.err != nil {
			t.Fatal(r.err)
		}
		d := r.body["decisions"].(map[string]any)["d"].(map[string]any)
		if d["selected"] != "b" || d["comment"] != "k" {
			t.Fatalf("decoded: %+v", r.body)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("await never returned")
	}
}

// A later submission replaces an earlier unconsumed one — the queue
// only holds the most recent.
func TestSubmitReplacesPreviousUnconsumed(t *testing.T) {
	ts, _ := newTestServer(t)

	for _, sel := range []string{"a", "b"} {
		payload := `{"spec_version":1,"decisions":{"d":{"selected":"` + sel + `"}}}`
		res, err := http.Post(ts.URL+"/submit", "application/json", strings.NewReader(payload))
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
	}

	resp, err := http.Get(ts.URL + "/await")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var b map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&b); err != nil {
		t.Fatal(err)
	}
	d := b["decisions"].(map[string]any)["d"].(map[string]any)
	if d["selected"] != "b" {
		t.Fatalf("expected latest (b), got: %+v", d)
	}
}

func TestSpecEndpointReturnsJSON(t *testing.T) {
	ts, _ := newTestServer(t)
	r, err := http.Get(ts.URL + "/spec")
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	if r.StatusCode != 200 {
		t.Fatalf("status: %d", r.StatusCode)
	}
	if ct := r.Header.Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Fatalf("Content-Type: %q", ct)
	}
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["title"] != "t" || body["version"].(float64) != 1 {
		t.Fatalf("decoded: %+v", body)
	}
}

func TestSpecEndpointReturnsRawYAMLWhenAsked(t *testing.T) {
	ts, _ := newTestServer(t)
	r, err := http.Get(ts.URL + "/spec?raw=1")
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	if r.StatusCode != 200 {
		t.Fatalf("status: %d", r.StatusCode)
	}
	if ct := r.Header.Get("Content-Type"); !strings.Contains(ct, "yaml") {
		t.Fatalf("Content-Type: %q", ct)
	}
	body, _ := io.ReadAll(r.Body)
	if !strings.HasPrefix(string(body), "version: 1") {
		t.Fatalf("not raw YAML: %s", string(body))
	}
}

func TestNewRejectsInvalidSpec(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.speckle")
	bad := `version: 1
title: t
sections:
  - id: dup
    heading: a
    decisions:
      - { id: d, prompt: p, options: [{id: o, label: O}] }
  - id: dup
    heading: b
    decisions:
      - { id: d, prompt: p, options: [{id: o, label: O}] }
`
	if err := os.WriteFile(path, []byte(bad), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := server.New(server.Config{
		Path: path, Addr: "127.0.0.1:0",
		Parser: spec.NewParser(), Renderer: render.NewRenderer(),
	})
	if err == nil {
		t.Fatal("expected New to reject duplicate section id")
	}
	if !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("error: %v", err)
	}
}
