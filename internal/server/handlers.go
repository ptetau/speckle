package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
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
