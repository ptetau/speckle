package main

import (
	"os"
	"reflect"
	"testing"

	"gopkg.in/yaml.v3"
)

func unmarshal(t *testing.T, s string) any {
	t.Helper()
	var v any
	if err := yaml.Unmarshal([]byte(s), &v); err != nil {
		t.Fatalf("yaml: %v", err)
	}
	return v
}

func TestMergeMapDeep(t *testing.T) {
	base := unmarshal(t, "a: 1\nb:\n  c: 2\n  d: 3\n")
	overlay := unmarshal(t, "b:\n  c: 99\n  e: 4\n")
	want := unmarshal(t, "a: 1\nb:\n  c: 99\n  d: 3\n  e: 4\n")
	if got := mergeOverlay(base, overlay); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestMergeNullDeletes(t *testing.T) {
	base := unmarshal(t, "a: 1\nb: 2\n")
	overlay := unmarshal(t, "b: null\n")
	want := unmarshal(t, "a: 1\n")
	if got := mergeOverlay(base, overlay); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestMergeListByID(t *testing.T) {
	base := unmarshal(t, `
items:
  - id: a
    label: first
  - id: b
    label: second
`)
	overlay := unmarshal(t, `
items:
  - id: b
    label: updated
  - id: c
    label: new
`)
	want := unmarshal(t, `
items:
  - id: a
    label: first
  - id: b
    label: updated
  - id: c
    label: new
`)
	if got := mergeOverlay(base, overlay); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v\nwant %v", got, want)
	}
}

func TestMergeListDelete(t *testing.T) {
	base := unmarshal(t, `
items:
  - id: a
  - id: b
  - id: c
`)
	overlay := unmarshal(t, `
items:
  - id: b
    _delete: true
`)
	want := unmarshal(t, `
items:
  - id: a
  - id: c
`)
	if got := mergeOverlay(base, overlay); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v\nwant %v", got, want)
	}
}

func TestMergePlainListReplaces(t *testing.T) {
	base := unmarshal(t, "tags: [one, two, three]\n")
	overlay := unmarshal(t, "tags: [four]\n")
	want := unmarshal(t, "tags: [four]\n")
	if got := mergeOverlay(base, overlay); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestExampleSpecParses(t *testing.T) {
	b, err := os.ReadFile("examples/example.speckle")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parseSpec(b); err != nil {
		t.Fatalf("example.speckle does not parse: %v", err)
	}
}
