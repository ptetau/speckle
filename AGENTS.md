# Speckle — agent guide

Speckle is a spec-building tool: an agent emits a `.speckle` YAML file
describing a plan with embedded decisions; `speckle serve` renders it
as an HTML page; a human picks options and submits; the agent reads
the submission via `speckle await`, applies a YAML overlay via
`speckle patch`, and the loop continues.

## Read first

- **[`docs/development-guidelines.html`](./docs/development-guidelines.html)**
  — how we build this codebase. Open it before writing code.
- [`docs/spec.html`](./docs/spec.html) — the plan for speckle itself,
  rendered as a worked example of what `speckle serve` produces.
- [`docs/implementation-plan.md`](./docs/implementation-plan.md) —
  open work items for migrating the current flat layout to the
  modular monolith described in the guidelines.

## Operating rules

These are the load-bearing rules from the guidelines. Open the full
doc for nuance; if you only retain these:

1. **Acceptance test first, always.** No production code ships
   without a failing acceptance test that drove it.
2. **Red → green → refactor.** One phase at a time. No refactor on
   red. No new behaviour on green.
3. **New code lives in `internal/<module>/`** with a small exported
   interface and an unexported implementation. Wire in `main.go`.
4. **Modules don't reach into each other's privates.** Cross-module
   calls go through the published interface.
5. **Wrap errors with `%w`. Propagate `context.Context`** through
   anything that can block or wait.
6. **Bug fixes start with a failing reproducer.**
7. **If a change genuinely can't be acceptance-tested** (CSS, pure
   typography), say so explicitly in the commit message.

## Commands

```
go build -o speckle .       # build the CLI
go test ./...               # run all tests (the gate)
go vet ./...                # static checks
```

## Layout

```
main.go                          # CLI dispatch
serve.go await.go patch.go       # thin subcommand entrypoints
internal/
  spec/                          # YAML parse + validate (Parser interface)
  overlay/                       # yaml.Node deep-merge  (Merger interface)
  render/                        # HTML template + Markdown (Renderer interface)
  server/                        # HTTP + SSE + fsnotify + submission queue (Server interface)
examples/                        # sample .speckle files
docs/                            # spec, guidelines, plan
```

Each `internal/<module>/` exposes a small interface and a `NewX`
constructor; the implementation type is unexported. `main.go` and the
subcommand files are the only places that construct concrete
implementations and wire them together.
