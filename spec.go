package main

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

type Spec struct {
	Version  int       `yaml:"version" json:"version"`
	Title    string    `yaml:"title" json:"title"`
	Sections []Section `yaml:"sections" json:"sections"`
	Notes    string    `yaml:"notes,omitempty" json:"notes,omitempty"`
}

type Section struct {
	ID        string     `yaml:"id" json:"id"`
	Heading   string     `yaml:"heading" json:"heading"`
	Body      string     `yaml:"body,omitempty" json:"body,omitempty"`
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
}

type Preview struct {
	Kind     string `yaml:"kind" json:"kind"`
	Language string `yaml:"language,omitempty" json:"language,omitempty"`
	Body     string `yaml:"body" json:"body"`
}

func parseSpec(b []byte) (*Spec, error) {
	var s Spec
	if err := yaml.Unmarshal(b, &s); err != nil {
		return nil, fmt.Errorf("yaml: %w", err)
	}
	if s.Version != 1 {
		return nil, fmt.Errorf("unsupported version %d (want 1)", s.Version)
	}
	seenSec := map[string]bool{}
	for i, sec := range s.Sections {
		if sec.ID == "" {
			return nil, fmt.Errorf("section[%d]: missing id", i)
		}
		if seenSec[sec.ID] {
			return nil, fmt.Errorf("duplicate section id %q", sec.ID)
		}
		seenSec[sec.ID] = true
		seenDec := map[string]bool{}
		for j, d := range sec.Decisions {
			if d.ID == "" {
				return nil, fmt.Errorf("section %q decision[%d]: missing id", sec.ID, j)
			}
			if seenDec[d.ID] {
				return nil, fmt.Errorf("section %q: duplicate decision id %q", sec.ID, d.ID)
			}
			seenDec[d.ID] = true
			if len(d.Options) == 0 {
				return nil, fmt.Errorf("section %q decision %q: needs at least one option", sec.ID, d.ID)
			}
			seenOpt := map[string]bool{}
			for k, o := range d.Options {
				if o.ID == "" {
					return nil, fmt.Errorf("section %q decision %q option[%d]: missing id", sec.ID, d.ID, k)
				}
				if seenOpt[o.ID] {
					return nil, fmt.Errorf("section %q decision %q: duplicate option id %q", sec.ID, d.ID, o.ID)
				}
				seenOpt[o.ID] = true
				if o.Preview != nil {
					switch o.Preview.Kind {
					case "code", "html", "text":
					default:
						return nil, fmt.Errorf("section %q decision %q option %q: unknown preview kind %q", sec.ID, d.ID, o.ID, o.Preview.Kind)
					}
				}
			}
		}
	}
	return &s, nil
}
