package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/ptetau/speckle/internal/history"
)

func runShow(args []string) error {
	if len(args) != 2 {
		return errors.New("usage: speckle show <file.speckle> <ref>")
	}

	mgr, err := history.Open(args[0])
	if err != nil {
		return fmt.Errorf("open history: %w", err)
	}

	round, err := mgr.Show(args[1])
	if err != nil {
		return fmt.Errorf("show: %w", err)
	}

	fmt.Fprintln(os.Stdout, "--- spec ---")
	os.Stdout.Write(round.Spec)
	if !endsWithNewline(round.Spec) {
		fmt.Fprintln(os.Stdout)
	}

	if len(round.Decisions) > 0 {
		fmt.Fprintln(os.Stdout, "--- decisions ---")
		os.Stdout.Write(round.Decisions)
		if !endsWithNewline(round.Decisions) {
			fmt.Fprintln(os.Stdout)
		}
	}
	return nil
}

func endsWithNewline(b []byte) bool {
	return len(b) > 0 && b[len(b)-1] == '\n'
}
