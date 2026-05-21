package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/ptetau/speckle/internal/spec"
)

func runValidate(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: speckle validate <file.speckle>")
	}

	data, err := os.ReadFile(args[0])
	if err != nil {
		return fmt.Errorf("read spec: %w", err)
	}

	p := spec.NewParser()
	_, parseErr := p.Parse(data)

	if parseErr == nil {
		out, _ := json.Marshal(map[string]any{"valid": true})
		fmt.Println(string(out))
		return nil
	}

	// Invalid spec: emit JSON to stdout, then exit 1 without printing to stderr.
	type errEntry struct {
		Message string `json:"message"`
	}
	result := struct {
		Valid  bool       `json:"valid"`
		Errors []errEntry `json:"errors"`
	}{
		Valid:  false,
		Errors: []errEntry{{Message: parseErr.Error()}},
	}
	out, _ := json.Marshal(result)
	fmt.Println(string(out))
	os.Exit(1)
	return nil // unreachable
}
