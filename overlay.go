package main

import "fmt"

// mergeOverlay deep-merges overlay into base with these rules:
//   - Maps merge by key. Null values in the overlay delete the key.
//   - Lists of maps that all carry an "id" field merge by id; matching items
//     deep-merge, unmatched overlay items are appended. An overlay item with
//     "_delete: true" removes the matching base item.
//   - All other values (scalars, mismatched types, lists without ids) are
//     replaced wholesale by the overlay.
func mergeOverlay(base, overlay any) any {
	bMap, bIsMap := base.(map[string]any)
	oMap, oIsMap := overlay.(map[string]any)
	if bIsMap && oIsMap {
		out := make(map[string]any, len(bMap)+len(oMap))
		for k, v := range bMap {
			out[k] = v
		}
		for k, v := range oMap {
			if v == nil {
				delete(out, k)
				continue
			}
			if existing, ok := out[k]; ok {
				out[k] = mergeOverlay(existing, v)
			} else {
				out[k] = v
			}
		}
		return out
	}

	bList, bIsList := base.([]any)
	oList, oIsList := overlay.([]any)
	if bIsList && oIsList && listOfIDMaps(bList) && listOfIDMaps(oList) {
		return mergeListByID(bList, oList)
	}

	return overlay
}

func listOfIDMaps(l []any) bool {
	if len(l) == 0 {
		return false
	}
	for _, item := range l {
		m, ok := item.(map[string]any)
		if !ok {
			return false
		}
		if _, has := m["id"]; !has {
			return false
		}
	}
	return true
}

func mergeListByID(base, overlay []any) []any {
	idx := map[string]int{}
	result := make([]any, 0, len(base))
	for _, item := range base {
		m := item.(map[string]any)
		id := fmt.Sprint(m["id"])
		idx[id] = len(result)
		result = append(result, m)
	}
	for _, item := range overlay {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		id := fmt.Sprint(m["id"])
		if del, _ := m["_delete"].(bool); del {
			if i, ok := idx[id]; ok {
				result[i] = nil
			}
			continue
		}
		if i, ok := idx[id]; ok {
			result[i] = mergeOverlay(result[i], m)
		} else {
			idx[id] = len(result)
			result = append(result, m)
		}
	}
	out := result[:0]
	for _, x := range result {
		if x != nil {
			out = append(out, x)
		}
	}
	return out
}
