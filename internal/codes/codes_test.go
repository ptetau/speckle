package codes_test

import (
	"testing"

	"github.com/ptetau/speckle/internal/codes"
	"github.com/ptetau/speckle/internal/spec"
)

func ptr(s string) *string { return &s }

func TestAssignCodesBasic(t *testing.T) {
	s := &spec.Spec{
		Version: 1,
		Dimensions: []spec.Dimension{
			{ID: "cli", Label: "CLI", Color: "#000"},
		},
		Sections: []spec.Section{
			{
				ID:        "s1",
				Heading:   "Section One",
				Dimension: "cli",
				Decisions: []spec.Decision{
					{
						ID:     "d1",
						Prompt: "Choose?",
						Options: []spec.Option{
							{ID: "opt1", Label: "Option 1"},
							{ID: "opt2", Label: "Option 2"},
						},
						Selected: ptr("opt1"),
					},
				},
			},
		},
	}

	codes.Assign(s)

	// Dimension gets a code
	if s.Dimensions[0].Code == "" {
		t.Error("dimension should get a code")
	}
	if s.Dimensions[0].Code != "CLI" {
		t.Errorf("dimension code: got %q, want %q", s.Dimensions[0].Code, "CLI")
	}

	// Section gets a code
	if s.Sections[0].Code == "" {
		t.Error("section should get a code")
	}
	// Decision gets a code
	if s.Sections[0].Decisions[0].Code == "" {
		t.Error("decision should get a code")
	}
	// Options get codes
	for _, opt := range s.Sections[0].Decisions[0].Options {
		if opt.Code == "" {
			t.Errorf("option %q should get a code", opt.ID)
		}
	}
}

func TestAssignCodesSkipsExisting(t *testing.T) {
	s := &spec.Spec{
		Version: 1,
		Dimensions: []spec.Dimension{
			{ID: "cli", Label: "CLI", Color: "#000"},
		},
		Sections: []spec.Section{
			{
				ID:        "s1",
				Heading:   "Section",
				Dimension: "cli",
				Decisions: []spec.Decision{
					{
						ID:     "d1",
						Prompt: "Choose?",
						Options: []spec.Option{
							{ID: "opt1", Label: "Option 1", Code: "CLI-001"},
						},
						Selected: nil,
					},
				},
			},
		},
	}

	codes.Assign(s)

	// Should not overwrite existing code
	if s.Sections[0].Decisions[0].Options[0].Code != "CLI-001" {
		t.Errorf("existing code should not be overwritten, got %q", s.Sections[0].Decisions[0].Options[0].Code)
	}
}

func TestAssignCodesMultipleDimensions(t *testing.T) {
	s := &spec.Spec{
		Version: 1,
		Dimensions: []spec.Dimension{
			{ID: "cli", Label: "CLI", Color: "#000"},
			{ID: "api", Label: "API", Color: "#fff"},
		},
		Sections: []spec.Section{
			{
				ID:        "s1",
				Heading:   "CLI section",
				Dimension: "cli",
				Decisions: []spec.Decision{
					{
						ID:     "d1",
						Prompt: "?",
						Options: []spec.Option{
							{ID: "o1", Label: "O1"},
						},
						Selected: nil,
					},
				},
			},
			{
				ID:        "s2",
				Heading:   "API section",
				Dimension: "api",
				Decisions: []spec.Decision{
					{
						ID:     "d2",
						Prompt: "?",
						Options: []spec.Option{
							{ID: "o2", Label: "O2"},
						},
						Selected: nil,
					},
				},
			},
		},
	}

	codes.Assign(s)

	// CLI and API should have separate sequences
	cliSection := s.Sections[0].Code
	apiSection := s.Sections[1].Code
	if cliSection == "" {
		t.Error("cli section should have code")
	}
	if apiSection == "" {
		t.Error("api section should have code")
	}
	if cliSection[:3] != "CLI" {
		t.Errorf("cli section code should start with CLI, got %q", cliSection)
	}
	if apiSection[:3] != "API" {
		t.Errorf("api section code should start with API, got %q", apiSection)
	}
	// Sequences are independent — both sections should get -001
	if cliSection != "CLI-001" {
		t.Errorf("first cli entity should be CLI-001, got %q", cliSection)
	}
	if apiSection != "API-001" {
		t.Errorf("first api entity should be API-001, got %q", apiSection)
	}
}

func TestAssignCodesSequentialWithinDimension(t *testing.T) {
	s := &spec.Spec{
		Version: 1,
		Dimensions: []spec.Dimension{
			{ID: "cli", Label: "CLI", Color: "#000"},
		},
		Sections: []spec.Section{
			{
				ID:        "s1",
				Heading:   "Section",
				Dimension: "cli",
				Decisions: []spec.Decision{
					{
						ID:     "d1",
						Prompt: "?",
						Options: []spec.Option{
							{ID: "o1", Label: "O1"},
							{ID: "o2", Label: "O2"},
						},
						Selected: nil,
					},
				},
			},
		},
	}

	codes.Assign(s)

	sec := s.Sections[0]
	dec := sec.Decisions[0]

	codes4 := []string{sec.Code, dec.Code, dec.Options[0].Code, dec.Options[1].Code}
	expected := []string{"CLI-001", "CLI-002", "CLI-003", "CLI-004"}
	for i, got := range codes4 {
		if got != expected[i] {
			t.Errorf("entity[%d]: got %q, want %q", i, got, expected[i])
		}
	}
}
