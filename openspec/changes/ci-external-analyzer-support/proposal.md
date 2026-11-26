# Proposal: CI support for external analyzers (non-Go repos) in the reusable AI-report workflow and gaze init --ci

## Why

Gaze can already analyze TypeScript (via `gaze-analyzer-typescript` / reading-stone) and Python (via snake-eyes) through external analyzers, and `gaze report --analyzer` works locally. The CI path does not support them yet:

- `gaze init --ci` emits nothing in a non-Go repository — it skips with a warning that the reusable workflow analyzes `./...` from the repo root and requires a Go module there.
- The reusable workflow assumes Go throughout: it takes a Go `coverprofile`, checks it with `test -f "$COVERPROFILE"`, and runs `gaze report ./... --coverprofile=...` with a fixed package pattern and no `--analyzer` or `--language`.
- The scaffolded caller template is Go-specific (`go test -coverprofile=coverage.out ./...`).

A TypeScript or Python repository therefore cannot use the shared workflow and must hand-write its own. This change extends the reusable workflow and `gaze init --ci` so non-Go repositories can use the same pipeline with the analyzer reading its own coverage data.

Evidence the analyzer path works locally: on reading-stone, `npx vitest run --coverage` then `gaze report ./src/... --analyzer gaze-analyzer-typescript --format=json` filled every section in ~17s.

## What Changes

### Reusable workflow: optional external-analyzer inputs

Add optional inputs with defaults that keep today's Go behavior unchanged:

| Input | Default | Purpose |
|---|---|---|
| `analyzer` | `""` | Passed as `--analyzer`. Empty means Go-native analysis. |
| `language` | `""` | Passed as `--language`. Depends on #282 to work without `--analyzer`; passed through when set. |
| `packages` | `./...` | Package pattern; e.g. `./src/...` keeps scaffolded `.opencode/` code out of the report. |
| `setup` | `""` | Optional shell snippet run before `gaze report`, to install and put the analyzer on PATH (e.g. `npm ci && npm run build && npm link`, or `uv tool install snake-eyes`). |
| `node-version` | `""` | When non-empty, run `actions/setup-node` with that version. When empty, skip the step. |
| `python-version` | `""` | When non-empty, run `actions/setup-python` with that version. When empty, skip the step. |

Coverage contract for analyzers: `--coverprofile` is passed only in Go mode (when `analyzer` is empty). In analyzer mode the artifact holds whatever that analyzer reads (e.g. `coverage/coverage-final.json` for reading-stone). The skipped-report path still validates that the coverage artifact arrived, for both modes: in Go mode via `test -f "$COVERPROFILE"`, in analyzer mode by verifying the artifact contains files.

`setup-go` stays (still needed to `go install` gaze).

### `gaze init --ci`: scaffold for non-Go repositories

When there is no `go.mod`, look for a known project marker instead of skipping:

- `package.json` → Node caller workflow: sets up Node, runs the test suite with coverage, uploads `coverage/`, and calls the reusable workflow with `analyzer: gaze-analyzer-typescript` and `packages: ./src/...`.
- `pyproject.toml` → Python caller wired to snake-eyes the same way: sets up Python, runs `pytest` with coverage, uploads coverage data, and passes `analyzer: gaze-analyzer-python` (snake-eyes).

Each gets its own golden file next to `testdata/golden-gaze-ai-report.yml`. Keep the current warning for repositories with no recognized marker.

### Docs

Update `docs/guides/ci-integration.md`, protocol/report references, and README to cover the external-analyzer CI path.

## Capabilities

### New Capabilities

- `ci-external-analyzer`: reusable workflow inputs and scaffold variants for external analyzers

### Modified Capabilities

- `ci-workflow` — reusable workflow now accepts analyzer-related inputs; coverage handling branches on whether an analyzer is set
- `ci-scaffolding` — `gaze init --ci` no longer skips with `go.mod` absent when a known non-Go marker exists; it scaffolds a language-appropriate caller workflow

### Removed Capabilities

- None

## Impact

- `.github/workflows/reusable_gaze_ai_report.yml` — modified: new inputs, conditional setup steps, conditional `gaze report` invocation, conditional coverage check
- `internal/scaffold/assets/ci/gaze-ai-report.yml` — modified if needed (Go template stays unchanged to preserve golden file; new templates added)
- `internal/scaffold/assets/ci/gaze-ai-report-node.yml` — new embedded template for Node
- `internal/scaffold/assets/ci/gaze-ai-report-python.yml` — new embedded template for Python
- `internal/scaffold/scaffold.go` — modified: detect `package.json` / `pyproject.toml` when `go.mod` absent; render appropriate template; expose `CIVariant` and template selection logic
- `internal/scaffold/testdata/golden-gaze-ai-report.yml` — unchanged (Go golden file preserved)
- `internal/scaffold/testdata/golden-gaze-ai-report-node.yml` — new golden file
- `internal/scaffold/testdata/golden-gaze-ai-report-python.yml` — new golden file
- `internal/scaffold/ci_test.go` — modified/extended: variant detection matrix, new golden-file tests, Node/Python workflow structure tests
- Docs — `docs/guides/ci-integration.md`, `README.md`, `docs/reference/cli/init.md`
- Website docs issue required before merge

## Constitution Alignment

Assessed against Gaze constitution v1.3.0.

### I. Accuracy — PASS

Go caller's behavior and existing golden file are unchanged; analyzer inputs default to empty and preserve today's Go path byte-for-byte. Coverage checks are explicit in both modes.

### II. Minimal Assumptions — PASS

No new assumptions about host project's language beyond existing `go.mod` check; now also checks for `package.json`/`pyproject.toml` as explicit markers. Analyzer is opt-in; Go remains default. Setup step is explicit opt-in shell; no hidden magic.

### III. Actionable Output — PASS

Scaffolded workflows are report-only with commented thresholds, same as Go. Next-steps guidance remains. CI thresholds remain human-owned.

### IV. Testability — PASS

Pure template rendering functions remain testable with golden files; new variant detection is pure and table-driven; workflow YAML structure is validated via parsing.

## Risks

- Choosing default test commands for Node/Python that don't match every project's setup — mitigated by making templates clearly editable and documenting that the user should adjust the test command to their project's test runner.
- Analyzer coverage artifacts vary in shape — mitigated by making coverage check flexible (skip or generic directory check) rather than hard-coding a file name.
- `node-version`/`python-version` defaults empty means no toolchain setup unless requested — consumer must opt in; scaffolded Node/Python templates will set sensible defaults (e.g., node 20, python 3.12) so generated workflows are immediately runnable.
