package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/ptetau/speckle/internal/spec"
)

func runExpand(args []string) error {
	fs := flag.NewFlagSet("expand", flag.ContinueOnError)
	mode := fs.String("mode", "hybrid", "expansion mode: hybrid|adjacent|experimental|mix")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 2 {
		return errors.New("usage: speckle expand <file.speckle> <decision-id> [--mode hybrid|adjacent|experimental|mix]")
	}
	path := fs.Arg(0)
	decisionID := fs.Arg(1)

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read spec: %w", err)
	}
	s, err := spec.NewParser().Parse(data)
	if err != nil {
		return fmt.Errorf("parse spec: %w", err)
	}

	// Find the decision and its section
	var sectionID string
	found := false
	for _, sec := range s.Sections {
		for _, dec := range sec.Decisions {
			if dec.ID == decisionID {
				sectionID = sec.ID
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	if !found {
		return fmt.Errorf("decision %q not found in spec", decisionID)
	}

	stub := expandOverlayStub(sectionID, decisionID, *mode)
	fmt.Print(stub)
	return nil
}

func expandOverlayStub(sectionID, decisionID, mode string) string {
	return fmt.Sprintf(`# speckle expand: add 3 new options to decision %q using mode: %s
# Fill in the options below and run: speckle patch <file> < this-file.yaml
sections:
  - id: %s
    decisions:
      - id: %s
        options:
          - id: new_option_1
            label: "TODO: option label (mode: %s)"
            pros: []
            cons: []
          - id: new_option_2
            label: "TODO"
          - id: new_option_3
            label: "TODO"
`, decisionID, mode, sectionID, decisionID, mode)
}
