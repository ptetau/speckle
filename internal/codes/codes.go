// Package codes auto-assigns DIM-NNN codes to spec entities.
package codes

import (
	"fmt"
	"strings"

	"github.com/ptetau/speckle/internal/spec"
)

// Assign walks every Dimension, Section, Decision, and Option in s and
// assigns a code to any entity that does not already have one.
//
// Code format: dimension ID uppercased + "-" + 3-digit zero-padded sequence.
// Example: dimension id="cli" → codes "CLI-001", "CLI-002", …
//
// The sequence counter is per-dimension and shared across all entity types
// within that dimension (sections, decisions, options all share one counter).
// Entities that already have a code are skipped; the counter is not consumed
// for them.
//
// Dimensions themselves receive a code equal to their uppercased ID (no sequence suffix).
// Sections, decisions, and options within a dimension receive sequenced codes.
// Entities whose section has no dimension are skipped.
func Assign(s *spec.Spec) {
	// counter per dimension prefix (uppercased id)
	seq := make(map[string]int)

	// Assign dimension codes (just the uppercased ID)
	for i := range s.Dimensions {
		d := &s.Dimensions[i]
		if d.Code == "" {
			d.Code = strings.ToUpper(d.ID)
		}
	}

	// Walk sections in order; assign codes to section, its decisions and options.
	for si := range s.Sections {
		sec := &s.Sections[si]
		if sec.Dimension == "" {
			continue
		}
		prefix := strings.ToUpper(sec.Dimension)

		// Section
		if sec.Code == "" {
			seq[prefix]++
			sec.Code = fmt.Sprintf("%s-%03d", prefix, seq[prefix])
		}

		// Decisions within section
		for di := range sec.Decisions {
			dec := &sec.Decisions[di]
			if dec.Code == "" {
				seq[prefix]++
				dec.Code = fmt.Sprintf("%s-%03d", prefix, seq[prefix])
			}

			// Options within decision
			for oi := range dec.Options {
				opt := &dec.Options[oi]
				if opt.Code == "" {
					seq[prefix]++
					opt.Code = fmt.Sprintf("%s-%03d", prefix, seq[prefix])
				}
			}
		}
	}
}
