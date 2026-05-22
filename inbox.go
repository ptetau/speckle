package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ptetau/speckle/internal/spec"
)

func runInbox(args []string) error {
	if len(args) != 2 {
		return errors.New("usage: speckle inbox <file.speckle> <dim-id>")
	}
	path := args[0]
	dimID := args[1]

	text, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("read stdin: %w", err)
	}
	entry := strings.TrimRight(string(text), "\n")

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read spec: %w", err)
	}

	p := spec.NewParser()
	s, err := p.Parse(data)
	if err != nil {
		return fmt.Errorf("parse spec: %w", err)
	}

	if s.Inbox == nil {
		s.Inbox = make(map[string]string)
	}
	existing := s.Inbox[dimID]
	if existing != "" {
		s.Inbox[dimID] = existing + "\n" + entry
	} else {
		s.Inbox[dimID] = entry
	}

	out, err := marshalSpec(s)
	if err != nil {
		return fmt.Errorf("encode spec: %w", err)
	}
	return os.WriteFile(path, out, 0o644)
}
