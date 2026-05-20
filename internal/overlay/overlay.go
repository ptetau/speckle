// Package overlay deep-merges a YAML overlay document into a base
// document at the yaml.Node level, preserving the base's key order
// and (where unchanged) its comments.
package overlay

import "gopkg.in/yaml.v3"

// Merger applies an overlay yaml.Node to a base yaml.Node.
//
// Rules:
//
//   - Maps merge by key. A null value in the overlay deletes that key.
//   - Lists of mappings where every item has an "id" field merge by id;
//     matching items deep-merge, unmatched overlay items are appended,
//     and an overlay item with "_delete: true" removes the matching
//     base item.
//   - All other values (scalars, mismatched kinds, lists without ids)
//     are replaced wholesale by the overlay.
type Merger interface {
	Merge(base, overlay *yaml.Node) *yaml.Node
}

// NewMerger returns the standard Merger implementation.
func NewMerger() Merger { return &merger{} }
