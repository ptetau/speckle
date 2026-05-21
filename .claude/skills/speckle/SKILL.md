---
name: speckle
description: >
  Full speckle human-in-the-loop decision loop. Ensures the speckle binary is
  installed, serves a .speckle spec in the browser, blocks until a human submits
  decisions, commits a git history snapshot, then returns the decision JSON to
  the agent. Use for any task where you need a human to review and choose between
  options before continuing.
---

# Speckle skill

## Invocation

```
/speckle [path/to/plan.speckle] [--update]
```

- Omit the path to auto-detect a `.speckle` file in the working tree.
- Pass `--update` to force `go install github.com/ptetau/speckle@latest`.

---

## Step 1 — Ensure binary

Check whether `speckle` is on PATH:

```powershell
# Windows
Get-Command speckle -ErrorAction SilentlyContinue
```
```bash
# Unix
command -v speckle
```

If not found, or if `--update` was passed, install:

```bash
go install github.com/ptetau/speckle@latest
```

If the install fails (no Go, no network), stop and tell the user:
> "speckle not found and `go install` failed. Install Go from https://go.dev/dl or add speckle to PATH."

---

## Step 2 — Find the .speckle file

If a path was given as an argument, use it directly.

Otherwise, glob for `**/*.speckle` in the current working directory (excluding `.speckle-repo/`):

- **Exactly one match** → use it; tell the agent which file was chosen.
- **Multiple matches** → list them and ask the agent which to use before proceeding.
- **No match** → stop with:
  > "No .speckle file found. Write one first (see README for format), then run /speckle again."

---

## Step 3 — Start server (if not already running)

Check whether a lockfile already exists at `<spec-path>.lock`.

**If lockfile exists**: read the URL from it. Reuse the existing server.

**If no lockfile**: start the server in the background:

```bash
speckle serve <spec-path> &
```

Wait up to 3 seconds, polling every 100 ms, for the lockfile to appear. Read the URL from it:

```json
{ "url": "http://127.0.0.1:XXXXX", "port": XXXXX, "pid": XXXXX }
```

If the lockfile doesn't appear within 3 seconds, report the server's stderr and stop.

Tell the agent (and user):
> "Speckle serving at **\<URL\>** — open in browser, review the options, and click Submit."

---

## Step 4 — Await submission

Run `speckle await <spec-path>`. This blocks with no timeout until the human submits.

```bash
decisions=$(speckle await path/to/plan.speckle)
```

Capture stdout as the decisions JSON.

---

## Step 5 — Commit history (after submit)

Write the decisions JSON to a temporary file, then commit:

```bash
# Write decisions to temp file
echo "$decisions" > /tmp/speckle-decisions.json   # Unix
$decisions | Out-File -Encoding utf8 $env:TEMP\speckle-decisions.json  # Windows

# Commit spec state + decisions as sidecar (what the human saw and chose)
speckle commit --decisions /tmp/speckle-decisions.json --message submit path/to/plan.speckle
```

This creates a git commit in the spec's `.speckle-repo/` (auto-created on first run) that captures:
- `plan.speckle` — the spec as the human saw it
- `plan.decisions.json` — the raw decisions submitted

The `.speckle-manifest.json` in the project root (or spec directory) records where the history repo lives.

---

## Step 6 — Return to agent

Return the decisions JSON as the skill result. Example:

```json
{
  "spec_version": 1,
  "decisions": {
    "strategy": { "selected": "jwt", "comment": "Redis not yet deployed." },
    "layout":   { "selected": "split" }
  },
  "notes": "Revisit auth when Redis lands."
}
```

The agent acts on the decisions — typically by computing a YAML overlay and running:

```bash
speckle patch path/to/plan.speckle < overlay.yaml

# Commit the patched spec (what the agent changed)
speckle commit --message patch path/to/plan.speckle
```

Then call `/speckle` again for the next round. The server stays running; `speckle await` rejoins it.

---

## History layout

```
project/
  .git/                              ← project git (manifest lives here)
  .speckle-manifest.json             ← maps spec paths → history repo paths
  plan.speckle
  plan.speckle.lock                  ← written by serve, deleted on shutdown
  .speckle-repo/
    .git/                            ← dedicated history git repo
    .speckle-meta.json               ← detection marker
    plan.speckle                     ← spec snapshot (committed each round)
    plan.decisions.json              ← last decisions (committed each round)
```

Two commits per round:
1. `speckle: submit` — spec as served + decisions sidecar
2. `speckle: patch` — patched spec after agent applies overlay

`git log` in `.speckle-repo/` shows the full decision trail. `git show <hash>` retrieves any historical state.

---

## Stopping the server

The server is not stopped automatically — it stays up for subsequent rounds. To shut it down explicitly:

```bash
# Unix
kill $(python3 -c "import sys,json; print(json.load(open('plan.speckle.lock'))['pid'])")

# Windows PowerShell
$specklePid = (Get-Content plan.speckle.lock | ConvertFrom-Json).pid
Stop-Process -Id $specklePid -Force
Remove-Item plan.speckle.lock   # Windows: manual cleanup (no graceful SIGINT)
```
