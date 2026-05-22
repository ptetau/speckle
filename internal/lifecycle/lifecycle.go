// Package lifecycle applies post-patch lifecycle operations to a spec.
package lifecycle

import "github.com/ptetau/speckle/internal/spec"

// ClearDecidedComments clears the Comment field on any Decision where
// Selected is set to a non-empty value. Comments on undecided decisions
// are preserved.
func ClearDecidedComments(s *spec.Spec) {
	for si := range s.Sections {
		for di := range s.Sections[si].Decisions {
			dec := &s.Sections[si].Decisions[di]
			if dec.Selected != nil && *dec.Selected != "" {
				dec.Comment = ""
			}
		}
	}
}
