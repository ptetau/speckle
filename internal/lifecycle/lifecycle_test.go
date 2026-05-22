package lifecycle_test

import (
	"testing"

	"github.com/ptetau/speckle/internal/lifecycle"
	"github.com/ptetau/speckle/internal/spec"
)

func ptr(s string) *string { return &s }

func TestClearDecidedComments(t *testing.T) {
	s := &spec.Spec{
		Version: 1,
		Sections: []spec.Section{
			{
				ID:      "s1",
				Heading: "Section",
				Decisions: []spec.Decision{
					{
						ID:       "decided",
						Prompt:   "?",
						Comment:  "should be cleared",
						Selected: ptr("opt1"),
						Options: []spec.Option{
							{ID: "opt1", Label: "Option 1"},
						},
					},
					{
						ID:       "undecided",
						Prompt:   "?",
						Comment:  "should be preserved",
						Selected: nil,
						Options: []spec.Option{
							{ID: "opt2", Label: "Option 2"},
						},
					},
				},
			},
		},
	}

	lifecycle.ClearDecidedComments(s)

	decided := s.Sections[0].Decisions[0]
	undecided := s.Sections[0].Decisions[1]

	if decided.Comment != "" {
		t.Errorf("decided.Comment should be empty, got %q", decided.Comment)
	}
	if undecided.Comment != "should be preserved" {
		t.Errorf("undecided.Comment should be preserved, got %q", undecided.Comment)
	}
}

func TestClearDecidedCommentsEmptySelected(t *testing.T) {
	emptyStr := ""
	s := &spec.Spec{
		Version: 1,
		Sections: []spec.Section{
			{
				ID:      "s1",
				Heading: "Section",
				Decisions: []spec.Decision{
					{
						ID:       "d1",
						Prompt:   "?",
						Comment:  "should be preserved when selected is empty string",
						Selected: &emptyStr,
						Options: []spec.Option{
							{ID: "o1", Label: "O1"},
						},
					},
				},
			},
		},
	}

	lifecycle.ClearDecidedComments(s)

	if s.Sections[0].Decisions[0].Comment == "" {
		t.Error("comment should not be cleared when selected is empty string")
	}
}
