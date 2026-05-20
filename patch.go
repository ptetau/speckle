package main

import (
	"errors"
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
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

	var base, overlay any
	if err := yaml.Unmarshal(baseBytes, &base); err != nil {
		return fmt.Errorf("parse base: %w", err)
	}
	if err := yaml.Unmarshal(overlayBytes, &overlay); err != nil {
		return fmt.Errorf("parse overlay: %w", err)
	}

	merged := mergeOverlay(base, overlay)
	out, err := yaml.Marshal(merged)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	if _, err := parseSpec(out); err != nil {
		return fmt.Errorf("merged result is not a valid spec: %w", err)
	}
	return os.WriteFile(path, out, 0o644)
}
