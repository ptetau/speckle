// Package spec parses and validates .speckle YAML files.
package spec

// Parser turns the bytes of a .speckle YAML document into a validated
// Spec, or returns a descriptive error.
type Parser interface {
	Parse(b []byte) (*Spec, error)
}

// NewParser returns the standard Parser implementation.
func NewParser() Parser { return &parser{} }

// Spec is the in-memory representation of a .speckle file.
type Spec struct {
	Version    int         `yaml:"version" json:"version"`
	Title      string      `yaml:"title" json:"title"`
	Dimensions []Dimension `yaml:"dimensions,omitempty" json:"dimensions,omitempty"`
	Sections   []Section   `yaml:"sections" json:"sections"`
	Notes      string      `yaml:"notes,omitempty" json:"notes,omitempty"`
}

// Dimension groups sections by concern and gives them a colour in the UI.
type Dimension struct {
	ID    string `yaml:"id" json:"id"`
	Label string `yaml:"label" json:"label"`
	Color string `yaml:"color" json:"color"`
}

type Section struct {
	ID        string     `yaml:"id" json:"id"`
	Heading   string     `yaml:"heading" json:"heading"`
	Body      string     `yaml:"body,omitempty" json:"body,omitempty"`
	Dimension string     `yaml:"dimension,omitempty" json:"dimension,omitempty"`
	Decisions []Decision `yaml:"decisions,omitempty" json:"decisions,omitempty"`
	Comment   string     `yaml:"comment,omitempty" json:"comment,omitempty"`
}

type Decision struct {
	ID       string   `yaml:"id" json:"id"`
	Prompt   string   `yaml:"prompt" json:"prompt"`
	Options  []Option `yaml:"options" json:"options"`
	Default  string   `yaml:"default,omitempty" json:"default,omitempty"`
	Selected *string  `yaml:"selected" json:"selected"`
	Comment  string   `yaml:"comment,omitempty" json:"comment,omitempty"`
}

type Option struct {
	ID          string   `yaml:"id" json:"id"`
	Label       string   `yaml:"label" json:"label"`
	Description string   `yaml:"description,omitempty" json:"description,omitempty"`
	Preview     *Preview `yaml:"preview,omitempty" json:"preview,omitempty"`
	Pros        []string `yaml:"pros,omitempty" json:"pros,omitempty"`
	Cons        []string `yaml:"cons,omitempty" json:"cons,omitempty"`
	Recommended bool     `yaml:"recommended,omitempty" json:"recommended,omitempty"`
}

type Preview struct {
	Kind     string `yaml:"kind" json:"kind"`
	Language string `yaml:"language,omitempty" json:"language,omitempty"`
	Body     string `yaml:"body" json:"body"`
}
