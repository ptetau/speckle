package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ptetau/speckle/internal/history"
)

func (s *server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	s.mu.RLock()
	sp := s.spec
	s.mu.RUnlock()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.cfg.Renderer.Render(w, sp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *server) handleSpec(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	raw := s.rawYAML
	sp := s.spec
	s.mu.RUnlock()
	if r.URL.Query().Get("raw") == "1" {
		w.Header().Set("Content-Type", "application/x-yaml")
		w.Write(raw)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(sp)
}

func (s *server) handleSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var sub Submission
	if err := json.NewDecoder(r.Body).Decode(&sub); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.queue.Push(sub)
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) handleAwait(w http.ResponseWriter, r *http.Request) {
	sub, ok := s.queue.Await(r.Context())
	if !ok {
		w.WriteHeader(http.StatusGatewayTimeout)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(sub)
}

// historyEntry is the JSON shape returned by GET /history.
type historyEntry struct {
	Hash      string `json:"hash"`
	Timestamp string `json:"timestamp"`
	Digest    string `json:"digest,omitempty"`
	Subject   string `json:"subject"`
}

func (s *server) handleHistory(w http.ResponseWriter, r *http.Request) {
	// Support /history/{ref} via URL path suffix
	ref := strings.TrimPrefix(r.URL.Path, "/history")
	ref = strings.TrimPrefix(ref, "/")

	mgr, err := history.Open(s.cfg.Path)
	if err != nil {
		// No history repo: return empty array for list, 404 for specific ref
		if ref != "" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]\n"))
		return
	}

	if ref != "" {
		// Return read-only HTML for the specific ref
		round, err := mgr.Show(ref)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		// Parse the spec at this ref and render it as read-only HTML
		sp, err := s.cfg.Parser.Parse(round.Spec)
		if err != nil {
			http.Error(w, fmt.Sprintf("parse spec at %s: %v", ref, err), http.StatusInternalServerError)
			return
		}
		// Render with read-only banner
		var buf bytes.Buffer
		if err := s.cfg.Renderer.Render(&buf, sp); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		html := buf.String()
		// Inject read-only banner and disable all inputs
		banner := fmt.Sprintf(
			`<script>document.addEventListener('DOMContentLoaded',function(){`+
				`var b=document.getElementById('history-banner');`+
				`if(b){b.textContent='Viewing version %s';b.style.display='block';}` +
				`document.querySelectorAll('input,textarea,button#submit,select').forEach(function(el){el.disabled=true;});`+
				`});</script>`,
			ref,
		)
		html = strings.Replace(html, "</body>", banner+"</body>", 1)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(html))
		return
	}

	// Return JSON array of log entries
	entries, err := mgr.Log()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	out := make([]historyEntry, 0, len(entries))
	for _, e := range entries {
		out = append(out, historyEntry{
			Hash:      e.Hash,
			Timestamp: e.Timestamp.Format(time.RFC3339),
			Digest:    e.Digest,
			Subject:   e.Subject,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

// expandRequest is the body of POST /expand.
type expandRequest struct {
	DecisionID string `json:"decision_id"`
	Mode       string `json:"mode"`
}

func (s *server) handleExpand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req expandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.Mode == "" {
		req.Mode = "hybrid"
	}

	s.mu.RLock()
	sp := s.spec
	s.mu.RUnlock()

	// Find the section containing this decision
	var sectionID string
	for _, sec := range sp.Sections {
		for _, dec := range sec.Decisions {
			if dec.ID == req.DecisionID {
				sectionID = sec.ID
				break
			}
		}
		if sectionID != "" {
			break
		}
	}
	if sectionID == "" {
		http.Error(w, fmt.Sprintf("decision %q not found", req.DecisionID), http.StatusNotFound)
		return
	}

	overlay := fmt.Sprintf(`# speckle expand: add 3 new options to decision %q using mode: %s
# Fill in the options below and run: speckle patch <file> < this-file.yaml
sections:
  - id: %s
    decisions:
      - id: %s
        options:
          - id: new_option_1
            label: "TODO: option label (mode: %s)"
            pros: []
            cons: []
          - id: new_option_2
            label: "TODO"
          - id: new_option_3
            label: "TODO"
`, req.DecisionID, req.Mode, sectionID, req.DecisionID, req.Mode)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"overlay": overlay})
}

func (s *server) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := make(chan struct{}, 4)
	s.subsMu.Lock()
	s.subs[ch] = struct{}{}
	s.subsMu.Unlock()
	defer func() {
		s.subsMu.Lock()
		delete(s.subs, ch)
		s.subsMu.Unlock()
	}()

	fmt.Fprint(w, ": ok\n\n")
	flusher.Flush()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ch:
			fmt.Fprint(w, "event: reload\ndata: {}\n\n")
			flusher.Flush()
		case <-ticker.C:
			fmt.Fprint(w, ": keepalive\n\n")
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}
