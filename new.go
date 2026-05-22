package main

import (
	"errors"
	"fmt"
	"os"
)

// starterSpec is a commented, valid starter .speckle file.
const starterSpec = `# .speckle spec — edit this file to describe your plan.
# Run:  speckle serve <file.speckle>   to view it in a browser.
#       speckle await <file.speckle>   to wait for a human decision.
#       speckle commit <file.speckle>  to snapshot the current state.

# version: schema version. Always 1.
version: 1

# title: a short name for the plan shown at the top of the page.
title: My plan

# sections: one or more sections, each with an id and a heading.
sections:
  - id: example-section
    heading: Example decision
    # body: optional Markdown description shown above the choices.
    body: |
      Pick a strategy. Each option can include a code preview.

    # decisions: one or more decisions nested inside this section.
    decisions:
      - id: strategy
        # prompt: the question shown to the human decision-maker.
        prompt: Which approach should we take?
        # options: at least two options; give each a unique id and label.
        options:
          - id: option-a
            label: Option A
            # description: optional detail shown below the label.
            description: The first approach — fast but less flexible.
          - id: option-b
            label: Option B
            description: The second approach — slower but more robust.
        # default: the pre-selected option id (must match one option id).
        default: option-a
        # selected: set by speckle after the human submits; leave null.
        selected: null
        # code: auto-assigned by speckle patch (e.g. CLI-003). Do not set manually.
`

func runNew(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: speckle new <file.speckle>")
	}
	path := args[0]

	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("file already exists: %s", path)
	}

	if err := os.WriteFile(path, []byte(starterSpec), 0o644); err != nil {
		return fmt.Errorf("write starter spec: %w", err)
	}
	fmt.Fprintf(os.Stderr, "speckle: created %s\n", path)
	return nil
}
