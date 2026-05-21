package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/ptetau/speckle/internal/history"
)

func runLog(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: speckle log <file.speckle>")
	}

	mgr, err := history.Open(args[0])
	if err != nil {
		return fmt.Errorf("open history: %w", err)
	}

	entries, err := mgr.Log()
	if err != nil {
		return fmt.Errorf("log: %w", err)
	}

	if len(entries) == 0 {
		fmt.Fprintln(os.Stdout, "no history")
		return nil
	}

	for _, e := range entries {
		ts := e.Timestamp.UTC().Format("2006-01-02T15:04:05Z")
		line := fmt.Sprintf("%s %s %s", e.Hash, ts, e.Subject)
		if e.Decisions != "" {
			line += "  [" + e.Decisions + "]"
		}
		fmt.Fprintln(os.Stdout, line)
	}
	return nil
}
