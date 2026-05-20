package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/ptetau/speckle/internal/overlay"
	"github.com/ptetau/speckle/internal/spec"
)

func runPatch(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: speckle patch <file.speckle> < overlay.yaml")
	}
	path := args[0]

	overlayBytes, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("read stdin: %w", err)
	}
	baseBytes, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var baseDoc, overlayDoc yaml.Node
	if err := yaml.Unmarshal(baseBytes, &baseDoc); err != nil {
		return fmt.Errorf("parse base: %w", err)
	}
	if err := yaml.Unmarshal(overlayBytes, &overlayDoc); err != nil {
		return fmt.Errorf("parse overlay: %w", err)
	}
	if baseDoc.Kind != yaml.DocumentNode || len(baseDoc.Content) == 0 {
		return errors.New("base file is empty")
	}
	if overlayDoc.Kind != yaml.DocumentNode || len(overlayDoc.Content) == 0 {
		return errors.New("overlay is empty")
	}

	merged := overlay.NewMerger().Merge(baseDoc.Content[0], overlayDoc.Content[0])

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(merged); err != nil {
		return fmt.Errorf("encode: %w", err)
	}
	if err := enc.Close(); err != nil {
		return err
	}
	out := buf.Bytes()
	if _, err := spec.NewParser().Parse(out); err != nil {
		return fmt.Errorf("merged result is not a valid spec: %w", err)
	}
	return os.WriteFile(path, out, 0o644)
}
