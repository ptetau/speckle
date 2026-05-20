package main

import "gopkg.in/yaml.v3"

// mergeOverlayNodes deep-merges overlay into base at the yaml.Node level,
// preserving the base document's key order and (where unchanged) its
// comments. Rules:
//
//   - Maps merge by key. A null value in the overlay deletes that key.
//   - Lists of mappings where every item has an "id" field merge by id;
//     matching items deep-merge, unmatched overlay items are appended,
//     an overlay item with "_delete: true" removes the matching base item.
//   - All other values (scalars, mismatched kinds, lists without ids)
//     are replaced wholesale by the overlay.
//
// base is mutated in place where possible; the returned node may be base
// (after edits) or overlay (for wholesale replacements).
func mergeOverlayNodes(base, overlay *yaml.Node) *yaml.Node {
	if base.Kind == yaml.MappingNode && overlay.Kind == yaml.MappingNode {
		return mergeMapping(base, overlay)
	}
	if base.Kind == yaml.SequenceNode && overlay.Kind == yaml.SequenceNode &&
		sequenceOfIDMaps(base) && sequenceOfIDMaps(overlay) {
		return mergeSequenceByID(base, overlay)
	}
	return overlay
}

func mergeMapping(base, overlay *yaml.Node) *yaml.Node {
	for i := 0; i < len(overlay.Content); i += 2 {
		k, v := overlay.Content[i], overlay.Content[i+1]
		if isNullScalar(v) {
			removeMapKey(base, k.Value)
			continue
		}
		if idx := mapKeyIndex(base, k.Value); idx >= 0 {
			base.Content[idx+1] = mergeOverlayNodes(base.Content[idx+1], v)
		} else {
			base.Content = append(base.Content, k, v)
		}
	}
	return base
}

func mergeSequenceByID(base, overlay *yaml.Node) *yaml.Node {
	idIdx := map[string]int{}
	for i, item := range base.Content {
		if id := mapScalarValue(item, "id"); id != "" {
			idIdx[id] = i
		}
	}
	for _, oItem := range overlay.Content {
		id := mapScalarValue(oItem, "id")
		if mapScalarValue(oItem, "_delete") == "true" {
			if i, ok := idIdx[id]; ok {
				base.Content[i] = nil
			}
			continue
		}
		if i, ok := idIdx[id]; ok {
			base.Content[i] = mergeOverlayNodes(base.Content[i], oItem)
		} else {
			idIdx[id] = len(base.Content)
			base.Content = append(base.Content, oItem)
		}
	}
	compacted := base.Content[:0]
	for _, x := range base.Content {
		if x != nil {
			compacted = append(compacted, x)
		}
	}
	base.Content = compacted
	return base
}

func sequenceOfIDMaps(s *yaml.Node) bool {
	if len(s.Content) == 0 {
		return false
	}
	for _, item := range s.Content {
		if item.Kind != yaml.MappingNode || mapKeyIndex(item, "id") < 0 {
			return false
		}
	}
	return true
}

func mapKeyIndex(m *yaml.Node, key string) int {
	if m.Kind != yaml.MappingNode {
		return -1
	}
	for i := 0; i < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return i
		}
	}
	return -1
}

func removeMapKey(m *yaml.Node, key string) {
	i := mapKeyIndex(m, key)
	if i < 0 {
		return
	}
	m.Content = append(m.Content[:i], m.Content[i+2:]...)
}

func mapScalarValue(m *yaml.Node, key string) string {
	i := mapKeyIndex(m, key)
	if i < 0 {
		return ""
	}
	v := m.Content[i+1]
	if v.Kind != yaml.ScalarNode {
		return ""
	}
	return v.Value
}

func isNullScalar(n *yaml.Node) bool {
	return n.Kind == yaml.ScalarNode && (n.Tag == "!!null" || n.Value == "null" || n.Value == "~")
}
