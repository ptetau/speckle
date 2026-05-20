package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Submission struct {
	SpecVersion     int                       `json:"spec_version"`
	Decisions       map[string]DecisionAnswer `json:"decisions"`
	SectionComments map[string]string         `json:"section_comments,omitempty"`
	Notes           string                    `json:"notes,omitempty"`
}

type DecisionAnswer struct {
	Selected string `json:"selected"`
	Comment  string `json:"comment,omitempty"`
}

type server struct {
	path string

	mu      sync.RWMutex
	spec    *Spec
	rawYAML []byte

	subsMu sync.Mutex
	subs   map[chan struct{}]struct{}

	submitMu sync.Mutex
	pending  chan Submission
}

func runServe(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	addr := fs.String("addr", "127.0.0.1:0", "address to listen on (default: random localhost port)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("usage: speckle serve [--addr=ADDR] <file.speckle>")
	}
	path := fs.Arg(0)

	s := &server{
		path:    path,
		subs:    make(map[chan struct{}]struct{}),
		pending: make(chan Submission, 1),
	}
	if err := s.reload(); err != nil {
		return err
	}

	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/spec", s.handleSpec)
	mux.HandleFunc("/submit", s.handleSubmit)
	mux.HandleFunc("/await", s.handleAwait)
	mux.HandleFunc("/events", s.handleEvents)

	httpSrv := &http.Server{Handler: mux}

	url := fmt.Sprintf("http://%s", ln.Addr())
	lockPath := path + ".lock"
	lockData, _ := json.Marshal(map[string]any{
		"url":  url,
		"port": ln.Addr().(*net.TCPAddr).Port,
		"pid":  os.Getpid(),
	})
	if err := os.WriteFile(lockPath, lockData, 0o644); err != nil {
		return fmt.Errorf("write lockfile: %w", err)
	}
	defer os.Remove(lockPath)

	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("fsnotify: %w", err)
	}
	defer w.Close()
	if err := w.Add(filepath.Dir(absPath)); err != nil {
		return fmt.Errorf("watch dir: %w", err)
	}
	go s.watch(w, absPath)

	fmt.Printf("speckle: serving %s on %s\n", path, url)
	fmt.Printf("speckle: lockfile %s\n", lockPath)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = httpSrv.Shutdown(shutdownCtx)
	}()

	if err := httpSrv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func (s *server) reload() error {
	b, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}
	spec, err := parseSpec(b)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.spec = spec
	s.rawYAML = b
	s.mu.Unlock()
	s.broadcastReload()
	return nil
}

func (s *server) watch(w *fsnotify.Watcher, target string) {
	var debounce *time.Timer
	for {
		select {
		case ev, ok := <-w.Events:
			if !ok {
				return
			}
			evPath, _ := filepath.Abs(ev.Name)
			if evPath != target {
				continue
			}
			if ev.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) == 0 {
				continue
			}
			if debounce != nil {
				debounce.Stop()
			}
			debounce = time.AfterFunc(80*time.Millisecond, func() {
				if err := s.reload(); err != nil {
					fmt.Fprintln(os.Stderr, "speckle: reload error:", err)
				}
			})
		case err, ok := <-w.Errors:
			if !ok {
				return
			}
			fmt.Fprintln(os.Stderr, "speckle: watch error:", err)
		}
	}
}

func (s *server) broadcastReload() {
	s.subsMu.Lock()
	defer s.subsMu.Unlock()
	for ch := range s.subs {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

func (s *server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	s.mu.RLock()
	spec := s.spec
	s.mu.RUnlock()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := renderHTML(w, spec); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *server) handleSpec(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	raw := s.rawYAML
	spec := s.spec
	s.mu.RUnlock()
	if r.URL.Query().Get("raw") == "1" {
		w.Header().Set("Content-Type", "application/x-yaml")
		w.Write(raw)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(spec)
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
	s.submitMu.Lock()
	select {
	case <-s.pending:
	default:
	}
	s.pending <- sub
	s.submitMu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) handleAwait(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	select {
	case sub := <-s.pending:
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(sub)
	case <-ctx.Done():
		w.WriteHeader(http.StatusGatewayTimeout)
	}
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
