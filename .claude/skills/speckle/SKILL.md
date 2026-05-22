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
- Pass `--update` to force reinstall even if the binary exists.

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

If not found **or** `--update` was passed, install:

```bash
go install github.com/ptetau/speckle@v0.2.0
```

After install, verify `speckle commit` exists (older cached `@latest` may be pre-v0.2.0):

```bash
speckle --help 2>&1 | grep -q commit || go install github.com/ptetau/speckle@v0.2.0
```

If install fails (no Go, no network), stop:
> "speckle not found and `go install` failed. Install Go from https://go.dev/dl or add speckle to PATH."

### Full CLI reference (v0.2.0)

| Command | Purpose |
|---|---|
| `speckle serve <file>`   | Render spec as HTML, accept submissions |
| `speckle await <file>`   | Block until submit; print decisions JSON to stdout |
| `speckle patch <file>`   | Apply YAML overlay from stdin |
| `speckle commit <file>`  | Snapshot spec+decisions to git history repo |
| `speckle new <file>`     | Scaffold a starter spec |
| `speckle validate <file>`| Parse and emit JSON errors; exit 0/1 |
| `speckle log <file>`     | List past decision rounds |
| `speckle show <file> <ref>` | Print spec+decisions at a git ref |

---

## Step 2 — Find the .speckle file

If a path was given as an argument, use it directly.

Otherwise, glob for `**/*.speckle` in the current working directory (excluding `.speckle-repo/`):

- **Exactly one match** → use it; tell the user which file was chosen.
- **Multiple matches** → list them and ask which to use before proceeding.
- **No match** → stop with:
  > "No .speckle file found. Run `speckle new <file.speckle>` to scaffold one, then run /speckle again."

---

## Step 3 — Start server (if not already running)

Check whether a lockfile exists at `<spec-path>.lock`.

**If lockfile exists**: read the URL from it. Then verify the server is alive:

```bash
# Unix
curl -sf <url> > /dev/null && echo alive || echo dead
```
```powershell
# Windows
try { Invoke-WebRequest -Uri <url> -TimeoutSec 2 -UseBasicParsing | Out-Null; "alive" } catch { "dead" }
```

- **alive** → reuse the existing server.
- **dead** → delete the stale lockfile, start a fresh server (same as no-lockfile path below).

**If no lockfile** (or stale one deleted): start server in background:

```bash
# Unix
speckle serve <spec-path> &

# Windows PowerShell
Start-Process -NoNewWindow speckle -ArgumentList "serve","<spec-path>"
```

Wait up to 3 seconds (poll every 100 ms) for the lockfile to appear. Read the URL:

```json
{ "url": "http://127.0.0.1:XXXXX", "port": XXXXX, "pid": XXXXX }
```

If lockfile doesn't appear in 3 seconds, report stderr and stop.

Tell the user:
> "Speckle serving at **\<URL\>** — open in browser, review the options, and click Submit."

---

## Step 4 — Await submission

Run `speckle await <spec-path>`. Blocks until the human submits. Capture stdout as decisions JSON.

```bash
# Unix
decisions=$(speckle await path/to/plan.speckle)
```

```powershell
# Windows — run as background task, read output file when complete
# (the Bash tool can also run this blocking; read the task output file after notification)
speckle await path/to/plan.speckle
```

---

## Step 5 — Commit history (after submit)

Write decisions to a temp file, then commit:

```bash
# Unix
echo "$decisions" > /tmp/speckle-decisions.json
speckle commit --decisions /tmp/speckle-decisions.json --message submit path/to/plan.speckle
```

```powershell
# Windows
$decisions | Out-File -Encoding utf8 "$env:TEMP\speckle-decisions.json"
speckle commit --decisions "$env:TEMP\speckle-decisions.json" --message submit path/to/plan.speckle
```

This commits to `.speckle-repo/`:
- `plan.speckle` — spec as the human saw it
- `plan.decisions.json` — raw decisions JSON

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

**If the JSON contains an `inbox` field**: process it through the LLM to generate a YAML overlay
before patching. The inbox holds raw ideas the human typed per dimension.

The agent applies decisions by computing a YAML overlay and running:

```bash
speckle patch path/to/plan.speckle < overlay.yaml

# Commit the patched spec
speckle commit --message patch path/to/plan.speckle
```

Then call `/speckle` again for the next round. The server stays running; `speckle await` rejoins it.

---

## History layout

```
project/
  .git/                              ← project git (manifest lives here)
  .speckle-manifest.json             ← maps spec paths → history repo paths (gitignored)
  plan.speckle
  plan.speckle.lock                  ← written by serve, deleted on shutdown (gitignored)
  .speckle-repo/                     ← dedicated history git repo (gitignored working copy)
    .git/
    .speckle-meta.json               ← detection marker
    plan.speckle                     ← spec snapshot per commit
    plan.decisions.json              ← decisions sidecar per commit
```

Two commits per round:
1. `speckle: submit` — spec as served + decisions sidecar
2. `speckle: patch` — spec after agent applies overlay

`speckle log plan.speckle` lists all rounds.
`speckle show plan.speckle <ref>` retrieves any historical state.

---

## Stopping the server

```bash
# Unix
kill $(python3 -c "import sys,json; print(json.load(open('plan.speckle.lock'))['pid'])")
```

```powershell
# Windows
$specklePid = (Get-Content plan.speckle.lock | ConvertFrom-Json).pid
Stop-Process -Id $specklePid -Force
Remove-Item plan.speckle.lock   # manual cleanup — no graceful SIGINT on Windows
```
