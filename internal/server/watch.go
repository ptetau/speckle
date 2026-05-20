package server

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

// watcher invokes onChange (debounced) whenever the target file is
// written, created, or renamed. It watches the parent directory
// because some editors atomically replace files via rename.
type watcher struct {
	w        *fsnotify.Watcher
	target   string
	onChange func() error
}

func newWatcher(absTarget string, onChange func() error) (*watcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("fsnotify: %w", err)
	}
	if err := w.Add(filepath.Dir(absTarget)); err != nil {
		w.Close()
		return nil, fmt.Errorf("watch dir: %w", err)
	}
	return &watcher{w: w, target: absTarget, onChange: onChange}, nil
}

func (w *watcher) Close() error { return w.w.Close() }

func (w *watcher) Run(ctx context.Context) {
	var debounce *time.Timer
	for {
		select {
		case ev, ok := <-w.w.Events:
			if !ok {
				return
			}
			evPath, _ := filepath.Abs(ev.Name)
			if evPath != w.target {
				continue
			}
			if ev.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) == 0 {
				continue
			}
			if debounce != nil {
				debounce.Stop()
			}
			debounce = time.AfterFunc(80*time.Millisecond, func() {
				if err := w.onChange(); err != nil {
					fmt.Fprintln(os.Stderr, "speckle: reload error:", err)
				}
			})
		case err, ok := <-w.w.Errors:
			if !ok {
				return
			}
			fmt.Fprintln(os.Stderr, "speckle: watch error:", err)
		case <-ctx.Done():
			return
		}
	}
}
