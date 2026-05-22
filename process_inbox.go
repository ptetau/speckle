package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/ptetau/speckle/internal/spec"
)

type processInboxOutput struct {
	SpecVersion int               `json:"spec_version"`
	Title       string            `json:"title"`
	Inbox       map[string]string `json:"inbox"`
}

func runProcessInbox(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: speckle process-inbox <file.speckle>")
	}
	path := args[0]

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read spec: %w", err)
	}
	p := spec.NewParser()
	s, err := p.Parse(data)
	if err != nil {
		return fmt.Errorf("parse spec: %w", err)
	}

	out := processInboxOutput{
		SpecVersion: s.Version,
		Title:       s.Title,
		Inbox:       s.Inbox,
	}
	if out.Inbox == nil {
		out.Inbox = map[string]string{}
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
