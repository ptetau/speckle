package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestFullAgentLoop is the marquee acceptance test: serve → submit →
// await → patch → SSE reload → re-await — one full agent iteration
// driven entirely through the CLI surface.
//
//  1. Agent starts `speckle serve`.
//  2. A browser subscribes to /events.
//  3. Browser submits a decision via POST /submit.
//  4. Agent's `speckle await` returns the submission JSON.
//  5. Agent computes an overlay (add a new option, flip default).
//  6. Agent applies the overlay via `speckle patch`.
//  7. Browser receives an SSE "reload" event.
//  8. The patched file contains the agent's changes, key order intact.
//  9. A second submission flows through the next await cleanly.
func TestFullAgentLoop(t *testing.T) {
	url, specPath, stop := startServer(t, minSpec)
	defer stop()

	// 2. Subscribe to /events before any change happens.
	events, eventsBody := openSSE(t, url)
	defer eventsBody.Close()
	drainSSE(events, 250*time.Millisecond)

	// First iteration: send a submission and let await consume it.
	awaitDone := runAwaitAsync(t, specPath)

	time.Sleep(150 * time.Millisecond)
	post(t, url+"/submit",
		`{"spec_version":1,"decisions":{"d":{"selected":"b","comment":"first round"}}}`, 204)

	first := waitForAwait(t, awaitDone, 3*time.Second)
	if got := jsonField(t, first, "decisions", "d", "selected"); got != "b" {
		t.Fatalf("first round selected: %v", got)
	}
	if got := jsonField(t, first, "decisions", "d", "comment"); got != "first round" {
		t.Fatalf("first round comment: %v", got)
	}

	// 5+6. Apply a patch as the agent would: append a new option, flip the default.
	patchOverlay := `sections:
  - id: s
    decisions:
      - id: d
        default: b
        options:
          - id: c
            label: C
`
	runPatchCLI(t, specPath, patchOverlay)

	// 7. Browser sees a reload event.
	if !waitForSSEEvent(events, "event: reload", 2*time.Second) {
		t.Fatal("no SSE reload event after speckle patch")
	}

	// 8. File now contains the change and the original key order survived.
	patched, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatal(err)
	}
	body := string(patched)
	if !strings.HasPrefix(body, "version: 1\ntitle:") {
		t.Errorf("top-level key order not preserved:\n%s", body)
	}
	if !strings.Contains(body, "default: b") || !strings.Contains(body, "id: c") {
		t.Errorf("patch didn't apply:\n%s", body)
	}

	// 9. Next iteration: another submission, another await, success.
	awaitDone2 := runAwaitAsync(t, specPath)
	time.Sleep(150 * time.Millisecond)
	post(t, url+"/submit",
		`{"spec_version":1,"decisions":{"d":{"selected":"c"}},"notes":"second round done"}`, 204)
	second := waitForAwait(t, awaitDone2, 3*time.Second)
	if got := jsonField(t, second, "decisions", "d", "selected"); got != "c" {
		t.Fatalf("second round selected: %v", got)
	}
	if got, ok := second["notes"].(string); !ok || got != "second round done" {
		t.Fatalf("second round notes: %v", second["notes"])
	}
}

// --- helpers ---

type awaitResult struct {
	body map[string]any
	err  error
}

func runAwaitAsync(t *testing.T, specPath string) <-chan awaitResult {
	t.Helper()
	out := make(chan awaitResult, 1)
	go func() {
		cmd := exec.Command(testBinary, "await", specPath)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			out <- awaitResult{err: errf("await: %v\nstderr: %s", err, stderr.String())}
			return
		}
		var body map[string]any
		if err := json.Unmarshal(stdout.Bytes(), &body); err != nil {
			out <- awaitResult{err: errf("decode await stdout: %v\nstdout: %s", err, stdout.String())}
			return
		}
		out <- awaitResult{body: body}
	}()
	return out
}

func waitForAwait(t *testing.T, ch <-chan awaitResult, within time.Duration) map[string]any {
	t.Helper()
	select {
	case r := <-ch:
		if r.err != nil {
			t.Fatal(r.err)
		}
		return r.body
	case <-time.After(within):
		t.Fatalf("await didn't return within %s", within)
		return nil
	}
}

func runPatchCLI(t *testing.T, specPath, overlay string) {
	t.Helper()
	cmd := exec.Command(testBinary, "patch", specPath)
	cmd.Stdin = strings.NewReader(overlay)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("patch: %v\nstderr: %s", err, stderr.String())
	}
}

func openSSE(t *testing.T, url string) (chan string, io.ReadCloser) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, url+"/events", nil)
	req.Header.Set("Accept", "text/event-stream")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		resp.Body.Close()
		t.Fatalf("/events: %d", resp.StatusCode)
	}
	ch := make(chan string, 32)
	go func() {
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			ch <- scanner.Text()
		}
		close(ch)
	}()
	return ch, resp.Body
}

func drainSSE(ch <-chan string, for_ time.Duration) {
	deadline := time.After(for_)
	for {
		select {
		case <-ch:
		case <-deadline:
			return
		}
	}
}

func waitForSSEEvent(ch <-chan string, want string, within time.Duration) bool {
	deadline := time.After(within)
	for {
		select {
		case line, ok := <-ch:
			if !ok {
				return false
			}
			if strings.HasPrefix(line, want) {
				return true
			}
		case <-deadline:
			return false
		}
	}
}

// jsonField walks a JSON-decoded map by string keys; non-existent
// keys cause a t.Fatal.
func jsonField(t *testing.T, body map[string]any, path ...string) any {
	t.Helper()
	var cur any = body
	for _, p := range path {
		m, ok := cur.(map[string]any)
		if !ok {
			t.Fatalf("jsonField %v: not a map at %q (have %T: %v)", path, p, cur, cur)
		}
		cur = m[p]
	}
	return cur
}
