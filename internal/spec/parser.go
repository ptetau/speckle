package spec

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

type parser struct{}

func (p *parser) Parse(b []byte) (*Spec, error) {
	var s Spec
	if err := yaml.Unmarshal(b, &s); err != nil {
		return nil, fmt.Errorf("yaml: %w", err)
	}
	if s.Version != 1 {
		return nil, fmt.Errorf("unsupported version %d (want 1)", s.Version)
	}
	seenDim := map[string]bool{}
	for i, d := range s.Dimensions {
		if d.ID == "" {
			return nil, fmt.Errorf("dimension[%d]: missing id", i)
		}
		if seenDim[d.ID] {
			return nil, fmt.Errorf("duplicate dimension id %q", d.ID)
		}
		seenDim[d.ID] = true
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
		if sec.Dimension != "" && !seenDim[sec.Dimension] {
			return nil, fmt.Errorf("section %q: unknown dimension id %q", sec.ID, sec.Dimension)
		}
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
