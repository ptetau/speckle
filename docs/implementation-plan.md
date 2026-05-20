# Speckle — implementation plan

**Status: complete.** Steps 0–4 and 6 landed in sequence (see git log
for the commit chain `Step 0` … `Step 6`). Step 5 was intentionally
skipped — see the note in that section. This file is retained as the
record of how the migration happened.

The codebase originally was a flat `package main` with all files at
the repo root. This plan migrated to the modular monolith layout
described in [`development-guidelines.html`](./development-guidelines.html).

Each step landed as its own commit. **The build and `go test ./...`
stayed green between every step.** Acceptance tests for the behaviour
being moved landed *before* the move, never after.

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

- [x] **HTTP acceptance:** spin up the current server (still
      `package main`) inside `httptest.NewServer`; POST a submission;
      assert `GET /await` returns the same JSON. Live in
      `serve_acceptance_test.go` for now.
- [x] **CLI acceptance:** `speckle patch` against a temp `.speckle`
      file; assert the file changed as expected, the version/title
      ordering survives, and re-parsing the result succeeds. Live in
      `patch_acceptance_test.go`.
- [x] **Validation acceptance:** feed `speckle serve` (or a direct
      `parseSpec` call from a `*_test.go`) a duplicate-id spec; assert
      a sensible error.

These are the regression net for the whole migration. Don't move
anything until they're in place and green.

## Step 1 — extract `internal/spec`

- [x] Create `internal/spec/spec.go`: exported `Parser` interface,
      exported value types (`Spec`, `Section`, `Decision`, `Option`,
      `Preview`), exported constructor `NewParser`.
- [x] Move `parseSpec` body into `internal/spec/parser.go` as method
      on unexported `parser` struct.
- [x] Add unit tests in `internal/spec/parser_test.go`
      (`package spec_test`) covering each validation branch
      (duplicate id, unknown preview kind, missing fields, version
      mismatch).
- [x] Update root callers (`serve.go`, `patch.go`, `await.go` if any)
      to accept a `spec.Parser` and call through the interface.
- [x] Acceptance tests from Step 0 still pass.

## Step 2 — extract `internal/overlay`

- [x] Create `internal/overlay/overlay.go`: exported `Merger`
      interface with one method (`Merge(base, overlay *yaml.Node)
      *yaml.Node`), constructor `NewMerger`.
- [x] Move `mergeOverlayNodes` and helpers into
      `internal/overlay/merge.go` as methods on the unexported type.
- [x] Migrate `overlay_test.go` to `internal/overlay/merge_test.go`
      (`package overlay_test`), drive through `NewMerger().Merge`.
- [x] Update `patch.go` to take an `overlay.Merger`.

## Step 3 — extract `internal/render`

- [x] Create `internal/render/render.go`: exported `Renderer`
      interface (`Render(w io.Writer, s *spec.Spec) error`),
      constructor `NewRenderer` that owns the parsed template.
- [x] Move `render.go` body + `template.html` (with `//go:embed`)
      into `internal/render/`.
- [x] Update `serve.go` to accept a `render.Renderer` and call
      through it from the index handler.
- [x] Renderer should accept the embedded `spec.Spec` from
      `internal/spec` — no own duplicate types.

## Step 4 — extract `internal/server`

This is the biggest step; split into substeps:

### 4a. Carve out the file watcher
- [x] `internal/server/watch.go` with unexported `watcher` type +
      `newWatcher(path, onChange)` constructor. Lifecycle is
      `Start(ctx)`, returns on `ctx.Done()`.
- [x] Replace inline fsnotify code in `serve.go` with the new type.

### 4b. Carve out the submission queue
- [x] `internal/server/submissions.go` with unexported `queue`
      handling the buffered-channel + drain-on-submit semantics.
      Methods: `Push(Submission)`, `Await(ctx) (Submission, error)`.
- [x] Replace inline `pendingSubmit` / `submitMu` logic with the
      queue.

### 4c. Assemble the server
- [x] `internal/server/server.go`: exported `Server` type with
      `http.Handler`, `Start(ctx)`, `Shutdown`. Constructor
      `New(parser spec.Parser, renderer render.Renderer, path string,
      logger *slog.Logger) *Server`.
- [x] `internal/server/handlers.go`: HTTP handler methods on the
      server, using the parser, renderer, watcher, queue.
- [x] Promote Step 0's HTTP acceptance test into
      `internal/server/server_acceptance_test.go`
      (`package server_test`).
- [x] Root `serve.go` shrinks to ~30 lines: parse flags, construct
      modules, call `server.Start(ctx)`.

## Step 5 — collapse subcommand entrypoints

**Skipped.** After Step 4 the root files (`main.go`, `serve.go`,
`await.go`, `patch.go`) were already the shape this step described —
`main.go` is pure dispatch, and each subcommand file is a small entry
point (the longest is `await.go` at ~75 lines, all of it flag parsing
and a single HTTP request). Renaming to `cmd_<name>.go` would have
been pure cosmetic noise; the function names (`runServe`, `runAwait`,
`runPatch`) already convey the role.

## Step 6 — tidy

- [x] Remove any remaining package-level globals (except `//go:embed`
      assets).
- [x] Add `context.Context` to anything that blocks. In particular:
      `server.Start(ctx)`, `queue.Await(ctx)`, `watcher.Start(ctx)`.
- [x] Introduce `log/slog` for the noisy paths: `internal/server/watch.go`
      uses `slog.Warn` for reload/watch errors. The user-facing
      "speckle: serving X on Y" startup banner stays as `fmt.Printf`
      because it's primary CLI output, not internal logging — slog's
      default text handler would add `time=… level=INFO msg=…`
      decoration that doesn't help users.
- [x] `go vet ./...`, `gofmt -s -l .` clean; consider adding
      `staticcheck` to CI.

## Step 7 — update `AGENTS.md`

- [x] Replace the "Layout (today)" section with the post-migration
      layout. Keep the operating rules as-is.

## Out of scope

These were considered and rejected (see `docs/spec.html`):

- BDD tooling (Gherkin / godog).
- Browser-driven acceptance tests.
- Splitting into multiple binaries (`cmd/<name>/`). Trigger that only
  if a second binary lands.

## Definition of done

- [x] All Step 0 acceptance tests still green.
- [x] `internal/spec/`, `internal/overlay/`, `internal/render/`,
      `internal/server/` exist with the documented shape.
- [x] No file under `internal/<a>/` references an unexported
      identifier from `internal/<b>/` (compiler-enforced).
- [x] `main.go`, `serve.go`, `await.go`, `patch.go` total ~220
      lines combined and contain no business logic — only flag
      parsing, module construction, and HTTP/exec glue.
- [x] `AGENTS.md` reflects the new layout.
