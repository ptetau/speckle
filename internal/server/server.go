// Package server hosts the HTTP surface for `speckle serve`: it renders
// the current spec, accepts a submission from a browser, hands it off
// to a waiting `speckle await` client, and reloads the page whenever
// the file on disk changes.
package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ptetau/speckle/internal/render"
	"github.com/ptetau/speckle/internal/spec"
)

// Server is the public surface of the package. Handler exposes the
// HTTP routes (so tests can mount them in httptest.NewServer); Run
// binds a real listener, writes the lockfile, watches the file, and
// blocks until ctx is cancelled.
type Server interface {
	Handler() http.Handler
	Run(ctx context.Context) error
}

// Config wires the server's dependencies and locates the spec file.
type Config struct {
	Path     string          // path to the .speckle file
	Addr     string          // listen address; "127.0.0.1:0" for a random localhost port
	Parser   spec.Parser     // YAML → *spec.Spec
	Renderer render.Renderer // *spec.Spec → HTML
}

// New constructs a Server, parsing the spec once up-front so an
// invalid file fails the constructor instead of surfacing on the
// first HTTP request.
func New(cfg Config) (Server, error) {
	if cfg.Parser == nil || cfg.Renderer == nil || cfg.Path == "" {
		return nil, errors.New("server.New: Parser, Renderer and Path are required")
	}
	s := &server{
		cfg:   cfg,
		subs:  make(map[chan struct{}]struct{}),
		queue: newQueue(),
	}
	if err := s.reload(); err != nil {
		return nil, err
	}
	return s, nil
}

type server struct {
	cfg Config

	mu      sync.RWMutex
	spec    *spec.Spec
	rawYAML []byte

	subsMu sync.Mutex
	subs   map[chan struct{}]struct{}

	queue *queue
}

func (s *server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/spec", s.handleSpec)
	mux.HandleFunc("/submit", s.handleSubmit)
	mux.HandleFunc("/await", s.handleAwait)
	mux.HandleFunc("/events", s.handleEvents)
	mux.HandleFunc("/history/", s.handleHistory)
	mux.HandleFunc("/history", s.handleHistory)
	mux.HandleFunc("/expand", s.handleExpand)
	return mux
}

func (s *server) Run(ctx context.Context) error {
	ln, err := net.Listen("tcp", s.cfg.Addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	url := fmt.Sprintf("http://%s", ln.Addr())
	lockPath := s.cfg.Path + ".lock"
	lockData, _ := json.Marshal(map[string]any{
		"url":  url,
		"port": ln.Addr().(*net.TCPAddr).Port,
		"pid":  os.Getpid(),
	})
	if err := os.WriteFile(lockPath, lockData, 0o644); err != nil {
		return fmt.Errorf("write lockfile: %w", err)
	}
	defer os.Remove(lockPath)

	absPath, err := filepath.Abs(s.cfg.Path)
	if err != nil {
		absPath = s.cfg.Path
	}
	w, err := newWatcher(absPath, s.reload)
	if err != nil {
		return err
	}
	defer w.Close()

	httpSrv := &http.Server{Handler: s.Handler()}

	fmt.Printf("speckle: serving %s on %s\n", s.cfg.Path, url)
	fmt.Printf("speckle: lockfile %s\n", lockPath)

	watchDone := make(chan struct{})
	go func() {
		w.Run(ctx)
		close(watchDone)
	}()

	serveErr := make(chan error, 1)
	go func() {
		if err := httpSrv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
			return
		}
		serveErr <- nil
	}()

	select {
	case err := <-serveErr:
		<-watchDone
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = httpSrv.Shutdown(shutdownCtx)
		<-watchDone
		<-serveErr
		return nil
	}
}

func (s *server) reload() error {
	b, err := os.ReadFile(s.cfg.Path)
	if err != nil {
		return err
	}
	parsed, err := s.cfg.Parser.Parse(b)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.spec = parsed
	s.rawYAML = b
	s.mu.Unlock()
	s.broadcastReload()
	return nil
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
