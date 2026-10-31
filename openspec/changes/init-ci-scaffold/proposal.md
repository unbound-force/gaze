# Init CI Scaffold: `gaze init --ci`

## Why

PR #300 shipped the provider-agnostic reusable workflow
(`.github/workflows/reusable_gaze_ai_report.yml`), but adopting it still
requires copy-pasting the consumer example from
`docs/guides/ci-integration.md` and hand-editing it. The biggest footgun
is version pinning: the `uses:` reference must point at a release tag
that actually contains the workflow — it first ships in `v2.3.0`, and
older tags fail to resolve at run time with an opaque
"workflow not found" error that gives the adopter no hint which
versions are valid. The remaining copy-paste steps are equally
error-prone: substituting the right Go version, re-pinning the actions
the example shows unpinned (`actions/upload-artifact@v4` in the docs),
setting up the `OPENCODE_CONFIG_CONTENT` secret as single-line strict
JSON, and deciding threshold values.

A deterministic installer that knows its own version eliminates the
below-floor class of pinning failures and collapses the rest of the
adoption path into one command plus one secret. (Validation is
offline and lexical — it cannot also prove a tag above the floor
exists, so the `--gaze-version` help text directs users to an
existing release tag; the residual failure mode is recorded in the
design's Risks.) The scaffold output is
green-to-commit before the secret exists (the reusable workflow's
fork-safe skip already validates the artifact handoff on secret-less
runs), so a team can land the workflow first and configure the secret
at leisure.

The generated starter is report-only by default: threshold inputs are
emitted commented-out, so the workflow exits 0 until a human decides
governance gates. No tool picks thresholds on the user's behalf
(gatekeeping value: quality gates are human decisions).

## What Changes

### `gaze init --ci` flag

An additive `--ci` flag on `gaze init`. Plain `gaze init` behavior is
unchanged; `--ci` additionally emits `.github/workflows/gaze-ai-report.yml`.
The feature targets Go-module-at-root repositories (the reusable
workflow's own limitation): when `go.mod` is absent from the target
root, the command warns and skips CI scaffolding while the `.opencode/`
scaffolding proceeds normally.

### Self-contained starter workflow

The generated workflow owns its own test + upload job (checkout →
setup-go → `go test -race -count=1 -short -coverprofile=coverage.out
./...` → upload-artifact) and a `gaze-ai-report` job (`needs: test`,
`permissions: contents: read`) that calls
`unbound-force/gaze/.github/workflows/reusable_gaze_ai_report.yml@<version>`
with the coverage artifact, the `gaze-version` input at the same
resolved pin (so the CI-installed binary matches the pin the
consumer calibrated thresholds with), and the `OPENCODE_CONFIG_CONTENT`
secret. No
YAML surgery on the consumer's existing workflows. The test job's
actions carry the same SHA pins gaze's own CI uses; the Go version is
read from the consumer's `go.mod` directive and validated against a
version grammar (`major[.minor[.patch]]` plus optional pre-release)
before substitution, falling back to `1.25` with a warning when the
directive is absent, unreadable, or malformed. A caller-owned concurrency
group (the CI-012 pattern) is included. Threshold inputs are emitted
commented-out with a TODO marker.

### Version pinning

The `uses:` version is resolved from the build-time gaze version and
validated as `>= v2.3.0` (the first tag containing the workflow). Dev
builds fail closed with guidance to install gaze from a release tag or
pass `--gaze-version`. A `--gaze-version` override flag receives the
same validation.

### Idempotency

Skip-if-present: users edit thresholds in the generated file, so it
must never be clobbered — the CI file is not added to `isToolOwned`
and `--force` is the only overwrite path, matching user-owned
`.opencode/` assets. Rerun with an existing file reports
"skipped (exists)".

### Template placement

The template lives at `internal/scaffold/assets/ci/gaze-ai-report.yml`
with `__GAZE_VERSION__` / `__GO_VERSION__` placeholders. The `ci/`
subtree is excluded from the `.opencode/` asset walk and from
`TestEmbeddedAssetsMatchSource` — no `.opencode/` mirror exists for it.

### Console next-steps output

`gaze init --ci` prints a CI next-steps section: the
`gh secret set OPENCODE_CONFIG_CONTENT` one-liner, a `jq -c` minify
tip, the second-secret masking pattern pointer, and a paste-able
threshold-guidance prompt (read the generated workflow, run
`gaze report ./... --format=json`, propose permissive initial
thresholds from the summary metrics, human approves final values,
uncomment the threshold inputs).

### Out of scope

Phase 2 (`/gaze thresholds` slash command for the recurring ratchet
loop) is decoupled to a separate follow-up change; this change is
scoped to `init`.

## Capabilities

### New Capabilities

- `ci-scaffolding`: `gaze init --ci` scaffolds a self-contained,
  report-only-by-default caller workflow for the reusable AI-report
  workflow, with a version pin validated against the earliest
  workflow-shipping release tag

### Modified Capabilities

- None — the reusable workflow itself, its contract, and the existing
  `gaze init` `.opencode/` scaffolding behavior are unchanged

### Removed Capabilities

- None

## Impact

- `internal/scaffold/scaffold.go` — modified: `Options.CI`,
  `Options.GazeVersion`, version resolution and `>= v2.3.0`
  validation, go.mod directive parsing with grammar validation and
  `1.25` fallback, `ci/` subtree handling in the asset walk, template
  placeholder substitution, CI section in `printSummary`
- `internal/scaffold/assets/ci/gaze-ai-report.yml` — new embedded
  template (SHA-pinned actions, `__GAZE_VERSION__`/`__GO_VERSION__`
  placeholders, commented-out threshold inputs, provenance header)
- `internal/scaffold/` test files — new: version-resolution matrix,
  go.mod directive-parser matrix, golden-file + YAML-parse validation
  of generated output, end-to-end directive-substitution and fallback
  assertions, next-steps section assertions, skip-if-present rerun,
  plain-init-untouched, mirror-test exclusion
  from the `ci/` subtree
- `cmd/gaze/main.go` — modified: init command wiring for `--ci` and
  `--gaze-version`
- `go.mod` — `golang.org/x/mod` promoted from indirect to direct (the
  `semver` package; already in the module graph via `x/tools` — no new
  supply-chain surface)
- Docs — `README.md`, `docs/reference/cli/init.md`,
  `docs/guides/ci-integration.md` (install section), and an AGENTS.md
  note that the `ci/` asset subtree is excluded from the `.opencode/`
  mirror sync
- The new YAML asset must pass yamllint (MegaLinter parity)
- Website Documentation Gate: applies — a new user-facing flag. A docs
  issue in `unbound-force/website` must be opened before merge
  (captured as a task).

## Constitution Alignment

Assessed against the Gaze project constitution
(`.specify/memory/constitution.md`, v1.3.0).

### I. Accuracy

**Assessment**: PASS

The emitted `uses:` pin is validated against the `v2.3.0` floor before
any file is written; dev builds and invalid overrides fail closed with
guidance instead of emitting a pin that cannot resolve at run time.
Generated output is byte-stable and golden-file tested, and the
generated workflow is YAML-parse validated in tests.

### II. Minimal Assumptions

**Assessment**: PASS

The flag is additive — plain `gaze init` behavior is byte-identical.
The Go version is read from the consumer's own `go.mod` directive
rather than assumed. Two unavoidable assumptions are recorded, each
explicit rather than silently ignored: (1) a Go module rooted at the
repository root — the reusable workflow's own limitation — enforced
at the entry point (warn and skip CI scaffolding when `go.mod` is
absent, never silently ignored); (2) `main` as the default branch in
the generated `on:` filters — mitigated by the provenance header's
instruction to edit the filters, though the failure mode is silent
for affected consumers (a `master`/`trunk` default branch yields a
workflow that never triggers until the filters are hand-edited).

### III. Actionable Output

**Assessment**: PASS

The console next-steps section gives concrete commands (secret-set
one-liner, `jq -c` minify, second-secret masking pointer) rather than
pointers to prose. Commented TODO markers in the generated workflow
identify exactly what a human must decide. Thresholds stay
human-owned: the scaffold emits no governance gates, only the
scaffolding around them.

### IV. Testability

**Assessment**: PASS

Version resolution and validation is a pure function tested as a
matrix (valid release, dev build, pre-floor version, floor-tag
pre-release, override, invalid strings). The go.mod directive parser
is equally pure and equally matrix-tested (grammar-conforming values
pass through; trailing whitespace and CR are stripped; absent,
unreadable, and malformed directives fall back to `1.25` with a
warning — a value outside the grammar is never emitted). Placeholder
substitution is pure. File emission
reuses the existing skip-if-present test patterns against temp
directories; generated output is asserted against a golden file,
parsed as YAML, and checked end-to-end for directive substitution (a
`go 1.24` module yields `go-version: "1.24"`; an unparseable
directive yields `1.25` plus the printed warning). The console
next-steps section is asserted item-by-item on success and asserted
absent on the missing-go.mod skip. No external services required.
