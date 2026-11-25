## Context

The reusable AI-report workflow and `gaze init --ci` (changes
`reusable-gaze-ai-report` and `init-ci-scaffold`) deliver the
shared CI pipeline, but both assume Go: `setup-go` plus a Go
coverprofile artifact in the workflow, and a hard `go.mod`
requirement in the scaffold. The CLI's external-analyzer support
(`internal/protocol/`, `internal/adapter/`, `gaze report
--analyzer/--language`) is already provider-agnostic, and the
first non-Go consumer exists: zero-dot-force/reading-stone, where
gaze v2.5.0 works locally via `npx vitest run --coverage` and
`gaze report ./src/... --analyzer gaze-analyzer-typescript
--format=json` (zero-dot-force/reading-stone#40). Issue #375 asks
the CI layer to catch up. The README's multi-language examples
(`gaze report --analyzer snake-eyes ./src`) ground the Python
wiring.

## Goals / Non-Goals

### Goals

- Make the reusable workflow usable from non-Go repositories via
  optional, Go-preserving inputs.
- Make `gaze init --ci` scaffold Node and Python callers from
  project markers.
- Demonstrate the analyzer mode end to end on a TypeScript
  repository in gaze's own CI.
- Handle the caller-controlled `setup` input safely (CWE-77) and
  document the caller-trust model.

### Non-Goals

- Changing `gaze report` CLI semantics, including #282 (honoring
  `--language` alone) — analyzer-mode callers must pass
  `--analyzer` for now.
- Shipping analyzers with gaze; the analyzer remains
  caller-supplied.
- Changing the `gaze-version` input default or any release gate.
- Detecting languages beyond the three markers.
- Modifying the report payload or threshold semantics.

## Decisions

### D1 — `analyzer` non-empty is the single mode switch

All new inputs are inert in Go mode; a non-empty `analyzer` input
selects analyzer mode. One switch keeps mode-dependent reasoning
simple and gives Go callers zero observable change. `packages`
defaults to `./...`, matching today's hard-coded pattern, so even
analyzer-mode callers that omit it get the current behavior.

### D2 — Input surface and defaults

`analyzer` (string, `""`), `language` (string, `""`), `packages`
(string, `./...`), `setup` (string, `""`), `node-version`
(string, `""`), `python-version` (string, `""`), and
`coverage-path` (string, `""`). The workflow already binds
optional inputs to `env:` and guards on emptiness (the threshold
inputs); the new string inputs follow the same convention, so an
unset input is an empty string the script treats as absent.

### D3 — CWE-77: `setup` reaches the shell only through env bindings

The `setup` input is caller-controlled shell — a
command-injection surface (CWE-77). The workflow binds it (like
every other consumed input) via `env:` and evaluates it as a
quoted POSIX variable inside the report step's `run:` script; no
input value is interpolated into `run:` script text. This mirrors
the pattern the workflow already uses (`GAZE_RELEASE`,
`OPENCODE_RELEASE`, `COVERPROFILE`, the threshold guards).

The setup script is evaluated at the top of the report step's
`run:` block — the same shell that invokes `gaze report` — so
PATH and environment changes persist to the report command
without `GITHUB_PATH`/`GITHUB_ENV` indirection. It does not run
when the report is skipped for an absent secret, because its only
purpose is to make the analyzer available to `gaze report`.

Caller-trust model (documented in the workflow header comment and
`docs/guides/ci-integration.md`): `workflow_call` inputs are
authored by the caller repository's own maintainers — the same
principals who author the caller's own `run:` steps — not by
untrusted third parties. A caller who can set `setup` can already
change their own workflow, so the input adds no new trust
principal. The env-binding pattern removes the residual CWE-77
vector of interpolating input text into this workflow's own
script body, where it could break out of quoting and alter the
fixed script.

### D4 — Coverage contract follows the mode; `coverage-path` added

Go mode keeps the coverprofile file check and `--coverprofile`.
In analyzer mode the artifact holds whatever the analyzer reads
(for reading-stone, the vitest `coverage/` directory), so a Go
coverprofile file check would falsely fail. The skipped-report
path instead validates artifact arrival through `coverage-path`
with an existence check that accepts a file or a directory,
whenever `coverage-path` is non-empty. The alternative — dropping
the check in analyzer mode — was rejected: the acceptance
criteria require the skipped-report path to verify the coverage
artifact arrived in both modes. `--coverprofile` is never passed
in analyzer mode; the analyzer reads its own coverage format from
the workspace, and how it locates that coverage is the analyzer's
contract, not a workflow input.

### D5 — Toolchain steps: setup-go always, node/python opt-in

`setup-go` runs in all modes because gaze itself is installed via
`go install`. SHA-pinned `setup-node`/`setup-python` steps (the
same pinning discipline gaze's own CI uses) run only when their
inputs are non-empty, keeping Go runs unchanged.

### D6 — `gaze-version` default untouched

The `validate-ai-report-default` release gate pins the default to
the release tag; this change does not touch it. A default bump,
if ever needed, is its own change (the
`bump-ai-report-default-*` precedent).

### D7 — Marker precedence: go.mod > package.json > pyproject.toml

`gaze init --ci` keeps its current behavior when `go.mod` exists
and generalizes otherwise. This supersedes the narrower "require
a Go module at the repository root" requirement in the
still-active `init-ci-scaffold` delta; when both changes sync to
main specs, the marker-based requirement replaces it (the old
delta's other requirements are unaffected). Repositories with no
recognized marker keep the skip-with-warning path, reworded to
name the three markers.

### D8 — Per-flavor templates; Go golden unchanged by construction

The Go template stays at
`internal/scaffold/assets/ci/gaze-ai-report.yml` with
byte-identical output; Node and Python get their own asset files
in the same subtree (`gaze-ai-report-node.yml`,
`gaze-ai-report-python.yml`). Goldens: the existing
`golden-gaze-ai-report.yml` is untouched; new
`golden-gaze-ai-report-node.yml` and
`golden-gaze-ai-report-python.yml` sit beside it. The `ci/`
subtree remains excluded from the `.opencode/` mirror sync, so
the nine-file embedded-asset manifest is unchanged.

### D9 — Node caller shape (grounded in reading-stone)

Per zero-dot-force/reading-stone#40: SHA-pinned `setup-node`,
`npx vitest run --coverage`, upload `coverage/` as the coverage
artifact, and `analyzer: gaze-analyzer-typescript` with
`packages: ./src/...` (commented as the adjustment point).
Thresholds stay commented (report-only), matching the Go flavor's
default.

### D10 — Python caller shape (grounded in the README)

The README's multi-language examples use `gaze report --analyzer
snake-eyes ./src`, so the Python caller passes
`analyzer: snake-eyes` with `packages: ./src`. The test and
coverage commands are pinned by the golden file; if snake-eyes
documents a different canonical invocation at implementation
time, the golden is the single place to adjust. The structural
shape mirrors the Node caller: SHA-pinned `setup-python`, a
coverage-generating test step, artifact upload, analyzer
passthrough, commented thresholds. This is an assumption recorded
here because no local snake-eyes evidence exists (the
sibling-repos list is empty).

### D11 — TypeScript demonstration: fixture job plus structural pinning

A caller cannot exercise the commit-under-test's workflow file
via `uses:` while checking out a different repository (the called
workflow's checkout follows the caller's repository), so gaze's
CI gets a fixture job that replays the analyzer-mode step
sequence directly: check out gaze at the triggering ref, check
out reading-stone at a pinned ref into a subdirectory, build gaze
from the commit under test, then run the caller setup and
`gaze report --analyzer=gaze-analyzer-typescript --format=json`
in report-only mode with no secret. The workflow file's own
contract (new inputs, defaults, mode gating, env-binding pattern)
is pinned by structural assertions that parse the YAML — the
same discipline `release_workflow_test.go` applies to
`release.yml`. The alternative — a `repository`/`ref` input on
the reusable workflow's checkout — was rejected for expanding the
trust surface and the change's scope.

### Constitution alignment carried from the proposal

The proposal's four PASS assessments rest on these decisions:
Autonomous Collaboration (D1, D2 — typed, self-describing inputs
with Go-preserving defaults), Composability First (D1, D5, D6 —
opt-in mode, no new mandatory dependencies, Go behavior
unchanged), Observable Quality (D3, D4, D11 — documented trust
model, observable artifact validation, machine-checked workflow
contract), and Testability (D8, D9, D10, D11 — golden-pinned
flavors and a runnable fixture).

## Risks / Trade-offs

- **Caller trust in `setup`** — accepted and documented (D3); the
  input is no more privileged than the caller's own steps, and
  the env-binding pattern removes the interpolation vector.
- **Snake-eyes specifics unverified locally** — the Python golden
  pins README-derived commands (D10); implementation verifies
  against snake-eyes and adjusts the golden in the same change.
- **Supersession across active deltas** — D7 records the
  reconciliation for spec sync; both deltas remain valid
  independently until then.
- **Fixture job depends on a pinned external ref** — a
  reading-stone breakage fails gaze's CI visibly; the pin makes
  the failure diagnosable, and the job is report-only so no
  secret is involved.
- **`--language` alone not honored (#282)** — analyzer-mode
  callers must pass `--analyzer`; documented in the guide as the
  current contract.
