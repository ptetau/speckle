package main

import (
	"bufio"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

// TestSSEReloadFiresOnFileChange: subscribing to /events and then
// modifying the .speckle file on disk delivers a "reload" SSE event to
// the browser, so a running page knows to refresh after `speckle patch`.
func TestSSEReloadFiresOnFileChange(t *testing.T) {
	url, specPath, stop := startServer(t, minSpec)
	defer stop()

	req, _ := http.NewRequest(http.MethodGet, url+"/events", nil)
	req.Header.Set("Accept", "text/event-stream")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("/events: %d", resp.StatusCode)
	}

	// Read SSE events on a background goroutine so we can time out.
	type sseLine struct {
		line string
		err  error
	}
	lines := make(chan sseLine, 16)
	go func() {
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			lines <- sseLine{line: scanner.Text()}
		}
		lines <- sseLine{err: scanner.Err()}
	}()

	// Drain the initial ": ok" comment and any whitespace.
	drainUntil := time.After(300 * time.Millisecond)
drain:
	for {
		select {
		case <-lines:
			// discard
		case <-drainUntil:
			break drain
		}
	}

	// Now modify the file. Use a small valid spec rewrite so reload
	// succeeds (a bad file would log a warn and not broadcast).
	updated := strings.Replace(minSpec, "Minimal plan", "Updated plan", 1)
	if err := os.WriteFile(specPath, []byte(updated), 0o644); err != nil {
		t.Fatal(err)
	}

	// Expect "event: reload" within fsnotify-debounce + slack.
	deadline := time.After(2 * time.Second)
	for {
		select {
		case ev := <-lines:
			if ev.err != nil {
				t.Fatalf("SSE stream error: %v", ev.err)
			}
			if strings.HasPrefix(ev.line, "event: reload") {
				return //
			}
		case <-deadline:
			t.Fatal("no reload event within 2s of file write")
		}
	}
}
