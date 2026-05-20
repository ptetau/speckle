# speckle

Spec-building tool for AI agents. The agent emits a `.speckle` YAML file
describing a plan with decision points; `speckle serve` renders it as an
HTML page where a human picks options, leaves comments, and submits. The
agent reads the submission via `speckle await`, computes a YAML overlay
describing the next iteration of the plan, and applies it with
`speckle patch`. The page hot-reloads, and the cycle continues.

## Build

```
go build -o speckle .
```

## Workflow

```sh
# 1. Agent writes plan.speckle.
$ ./speckle serve examples/example.speckle &
speckle: serving examples/example.speckle on http://127.0.0.1:5765
speckle: lockfile examples/example.speckle.lock

# 2. Open the URL in a browser, pick options, leave comments, click Submit.

# 3. Agent blocks waiting for the submission, then receives it on stdout:
$ ./speckle await examples/example.speckle
{
  "spec_version": 1,
  "decisions": {
    "strategy":  { "selected": "session", "comment": "Redis already deployed." },
    "layout":    { "selected": "split" }
  },
  "notes": "Add SSO before shipping."
}

# 4. Agent decides what to change, writes overlay.yaml, applies it:
$ ./speckle patch examples/example.speckle < overlay.yaml

# 5. The running server reloads the file; the browser refreshes via SSE.
#    Loop back to step 3.
```

`speckle await` finds the running server through `<file>.speckle.lock`,
written next to the file at server startup; pass `--url=http://host:port`
to override.

## File format

```yaml
version: 1
title: Plan title
sections:
  - id: section-id
    heading: Section heading
    body: |
      Markdown prose describing this part of the plan.
    decisions:
      - id: decision-id
        prompt: Question shown to the user.
        options:
          - id: option-id
            label: Option label
            description: Short blurb (optional).
            preview:
              kind: code | html | text
              language: optional language hint (code only)
              body: |
                Preview content.
        default: option-id
        selected: null
        comment: ""
notes: ""
```

Preview kinds:

- `code` — fixed-width block, language hint sets a `lang-*` class.
- `text` — preformatted plain text.
- `html` — rendered inside `<iframe sandbox srcdoc="…">`. Sandbox is fully
  restricted: no scripts, no network, no parent access. Style with inline
  CSS only.

## Overlay format

Overlays are YAML documents that deep-merge into the spec file:

- **Maps** merge recursively. Keys present in the overlay override the
  base; keys absent are left alone.
- A **null** value in the overlay deletes that key.
- **Lists of maps where every item has an `id` field** merge by id:
  matching items deep-merge; unmatched overlay items are appended; an
  overlay item with `_delete: true` removes the matching base item.
- All other values (scalars, mismatched types, lists without ids) are
  replaced wholesale.

Example overlay — change a default, add an option, drop another:

```yaml
sections:
  - id: auth-method
    decisions:
      - id: strategy
        default: session
        options:
          - id: oidc
            label: External OIDC provider
            preview:
              kind: code
              language: js
              body: app.use(passport.authenticate('oidc'));
          - id: jwt
            _delete: true
```

## Development

- **[`AGENTS.md`](./AGENTS.md)** — read first if you're an agent (or a
  human treating the codebase like one).
- **[`docs/development-guidelines.html`](./docs/development-guidelines.html)**
  — how we build speckle: modular monolith in `internal/<module>/`,
  outside-in acceptance tests, red&nbsp;/&nbsp;green&nbsp;/&nbsp;refactor.
- **[`docs/spec.html`](./docs/spec.html)** — the plan for speckle
  itself, rendered as a worked example.
- **[`docs/implementation-plan.md`](./docs/implementation-plan.md)** —
  outstanding work to bring the codebase in line with the guidelines.

## Security

HTML previews load inside `<iframe sandbox="allow-scripts" srcdoc="…">`.
Scripts inside a preview run, so options can demo interactive widgets
(hover state, animations, counters), but the iframe has no
`allow-same-origin` — preview JS cannot read cookies, reach the parent
page, or speak to the server. The page itself trusts the `.speckle`
file, so only run `speckle serve` on files you (or your agent) produced.
