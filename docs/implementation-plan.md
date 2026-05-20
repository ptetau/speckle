# Speckle — implementation plan

The codebase is currently a flat `package main` with all files at the
repo root. This plan migrates to the modular monolith layout described
in [`development-guidelines.html`](./development-guidelines.html).

Each step lands as its own PR / commit chain. **The build and `go test
./...` stay green between every step.** Acceptance tests for the
behaviour being moved land *before* the move, never after.

## Target layout

```
speckle/
├── main.go                          # composition root only
├── cmd_serve.go                     # thin CLI dispatch
├── cmd_await.go
├── cmd_patch.go
├── internal/
│   ├── spec/                        # YAML parse + validate
│   │   ├── spec.go                  # Parser interface, Spec types
│   │   └── parser.go                # unexported implementation
│   ├── overlay/                     # yaml.Node deep-merge
│   │   ├── overlay.go               # Merger interface
│   │   └── merge.go
│   ├── server/                      # HTTP + SSE + fsnotify + queue
│   │   ├── server.go                # Server interface + constructor
│   │   ├── handlers.go              # HTTP handlers
│   │   ├── watch.go                 # fsnotify reload
│   │   └── submissions.go           # submit/await channel
│   └── render/                      # HTML template + Markdown
│       ├── render.go                # Renderer interface
│       ├── render_impl.go
│       └── template.html
├── examples/
└── docs/
```

## Step 0 — baseline acceptance tests

Build the safety net before any code moves.

- [ ] **HTTP acceptance:** spin up the current server (still
      `package main`) inside `httptest.NewServer`; POST a submission;
      assert `GET /await` returns the same JSON. Live in
      `serve_acceptance_test.go` for now.
- [ ] **CLI acceptance:** `speckle patch` against a temp `.speckle`
      file; assert the file changed as expected, the version/title
      ordering survives, and re-parsing the result succeeds. Live in
      `patch_acceptance_test.go`.
- [ ] **Validation acceptance:** feed `speckle serve` (or a direct
      `parseSpec` call from a `*_test.go`) a duplicate-id spec; assert
      a sensible error.

These are the regression net for the whole migration. Don't move
anything until they're in place and green.

## Step 1 — extract `internal/spec`

- [ ] Create `internal/spec/spec.go`: exported `Parser` interface,
      exported value types (`Spec`, `Section`, `Decision`, `Option`,
      `Preview`), exported constructor `NewParser`.
- [ ] Move `parseSpec` body into `internal/spec/parser.go` as method
      on unexported `parser` struct.
- [ ] Add unit tests in `internal/spec/parser_test.go`
      (`package spec_test`) covering each validation branch
      (duplicate id, unknown preview kind, missing fields, version
      mismatch).
- [ ] Update root callers (`serve.go`, `patch.go`, `await.go` if any)
      to accept a `spec.Parser` and call through the interface.
- [ ] Acceptance tests from Step 0 still pass.

## Step 2 — extract `internal/overlay`

- [ ] Create `internal/overlay/overlay.go`: exported `Merger`
      interface with one method (`Merge(base, overlay *yaml.Node)
      *yaml.Node`), constructor `NewMerger`.
- [ ] Move `mergeOverlayNodes` and helpers into
      `internal/overlay/merge.go` as methods on the unexported type.
- [ ] Migrate `overlay_test.go` to `internal/overlay/merge_test.go`
      (`package overlay_test`), drive through `NewMerger().Merge`.
- [ ] Update `patch.go` to take an `overlay.Merger`.

## Step 3 — extract `internal/render`

- [ ] Create `internal/render/render.go`: exported `Renderer`
      interface (`Render(w io.Writer, s *spec.Spec) error`),
      constructor `NewRenderer` that owns the parsed template.
- [ ] Move `render.go` body + `template.html` (with `//go:embed`)
      into `internal/render/`.
- [ ] Update `serve.go` to accept a `render.Renderer` and call
      through it from the index handler.
- [ ] Renderer should accept the embedded `spec.Spec` from
      `internal/spec` — no own duplicate types.

## Step 4 — extract `internal/server`

This is the biggest step; split into substeps:

### 4a. Carve out the file watcher
- [ ] `internal/server/watch.go` with unexported `watcher` type +
      `newWatcher(path, onChange)` constructor. Lifecycle is
      `Start(ctx)`, returns on `ctx.Done()`.
- [ ] Replace inline fsnotify code in `serve.go` with the new type.

### 4b. Carve out the submission queue
- [ ] `internal/server/submissions.go` with unexported `queue`
      handling the buffered-channel + drain-on-submit semantics.
      Methods: `Push(Submission)`, `Await(ctx) (Submission, error)`.
- [ ] Replace inline `pendingSubmit` / `submitMu` logic with the
      queue.

### 4c. Assemble the server
- [ ] `internal/server/server.go`: exported `Server` type with
      `http.Handler`, `Start(ctx)`, `Shutdown`. Constructor
      `New(parser spec.Parser, renderer render.Renderer, path string,
      logger *slog.Logger) *Server`.
- [ ] `internal/server/handlers.go`: HTTP handler methods on the
      server, using the parser, renderer, watcher, queue.
- [ ] Promote Step 0's HTTP acceptance test into
      `internal/server/server_acceptance_test.go`
      (`package server_test`).
- [ ] Root `serve.go` shrinks to ~30 lines: parse flags, construct
      modules, call `server.Start(ctx)`.

## Step 5 — collapse subcommand entrypoints

- [ ] Move each subcommand into `cmd_<name>.go` at root, kept short
      (≤ 50 lines each). They handle: flag parsing, constructing
      modules from `internal/`, kicking off the work.
- [ ] `main.go` is the dispatcher — switch on `os.Args[1]`, call the
      matching `runCmd`. No business logic, no I/O.

## Step 6 — tidy

- [ ] Remove any remaining package-level globals (except `//go:embed`
      assets).
- [ ] Add `context.Context` to anything that blocks. In particular:
      `server.Start(ctx)`, `queue.Await(ctx)`, `watcher.Start(ctx)`.
- [ ] Introduce `log/slog` logger constructed in `main.go`, passed
      into modules that need it. Replace `fmt.Fprintln(os.Stderr, ...)`
      lifecycle logs with `slog.Info` / `slog.Warn`.
- [ ] `go vet ./...`, `gofmt -s -l .` clean; consider adding
      `staticcheck` to CI.

## Step 7 — update `AGENTS.md`

- [ ] Replace the "Layout (today)" section with the post-migration
      layout. Keep the operating rules as-is.

## Out of scope

These were considered and rejected (see `docs/spec.html`):

- BDD tooling (Gherkin / godog).
- Browser-driven acceptance tests.
- Splitting into multiple binaries (`cmd/<name>/`). Trigger that only
  if a second binary lands.

## Definition of done

- [ ] All Step 0 acceptance tests still green.
- [ ] `internal/spec/`, `internal/overlay/`, `internal/render/`,
      `internal/server/` exist with the documented shape.
- [ ] No file under `internal/<a>/` references an unexported
      identifier from `internal/<b>/` (compiler-enforced).
- [ ] `main.go` and `cmd_*.go` total ≤ 200 lines combined, contain
      no business logic.
- [ ] `AGENTS.md` reflects the new layout.
