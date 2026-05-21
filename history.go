package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/ptetau/speckle/internal/history"
)

func runCommit(args []string) error {
	fs := flag.NewFlagSet("commit", flag.ContinueOnError)
	decisionsPath := fs.String("decisions", "", "path to decisions JSON file to include as sidecar")
	msgPrefix := fs.String("message", "submit", `commit message prefix ("submit" or "patch")`)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("usage: speckle commit [--decisions FILE] [--message MSG] <file.speckle>")
	}

	mgr, err := history.Open(fs.Arg(0))
	if err != nil {
		return fmt.Errorf("open history: %w", err)
	}

	var decisions []byte
	if *decisionsPath != "" {
		decisions, err = os.ReadFile(*decisionsPath)
		if err != nil {
			return fmt.Errorf("read decisions file: %w", err)
		}
	}

	if err := mgr.Commit(decisions, *msgPrefix); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "speckle: committed to %s\n", mgr.RepoPath())
	return nil
}
