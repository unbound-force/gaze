# Design: CI support for external analyzers

## Context

`internal/scaffold` today scaffolds a single Go-specific caller workflow. The reusable workflow assumes Go: it requires a `coverprofile` file, runs `gaze report ./... --coverprofile=...`, and has no analyzer concept. External analyzers (reading-stone for TypeScript, snake-eyes for Python) work locally via `gaze report ./src/... --analyzer gaze-analyzer-typescript`, with the analyzer reading coverage data itself (e.g., `coverage/coverage-final.json` from `vitest --coverage`). The CI path needs to expose the same knobs.

## Goals / Non-Goals

### Goals

- Reusable workflow accepts optional `analyzer`, `language`, `packages`, `setup`, `node-version`, `python-version` inputs, all defaulting to preserve Go behavior unchanged.
- Coverage handling: `--coverprofile` only in Go mode; in analyzer mode the artifact check validates coverage data arrived (via artifact contents, not a single file).
- `gaze init --ci` in a repo with `package.json` and no `go.mod` writes a Node caller workflow that runs the reusable workflow with `gaze-analyzer-typescript`; `pyproject.toml` similarly for Python.
- Existing Go golden file unchanged; new golden files for Node/Python.
- Skipped-report path (no `opencode-config`) still validates artifact arrival for both modes.
- Reusable workflow passes `--analyzer`, `--language`, and package pattern through and runs the report on a TypeScript repository (dogfood or fixture).

### Non-Goals

- Implementing #282 (`gaze report --language` alone) — callers pass `--analyzer` directly; `language` is a passthrough for future use.
- Auto-detecting analyzer test commands beyond a sensible default (user edits the scaffolded workflow's test step as needed).
- Changing Gaze's own dogfood job beyond adding a fixture/demonstration if needed.

## Decisions

### D1: Reusable workflow inputs — additive, backward-compatible

New inputs with defaults that preserve Go behavior:

- `analyzer` (string, default `""`): when non-empty, passed as `--analyzer=$analyzer`
- `language` (string, default `""`): when non-empty, passed as `--language=$language`
- `packages` (string, default `"./..."`): package pattern argument to `gaze report`; replaces hard-coded `./...`
- `setup` (string, default `""`): shell snippet run before `gaze report` (e.g., `npm ci && npm run build && npm link`); when empty, step skipped
- `node-version` (string, default `""`): when non-empty, `actions/setup-node` step
- `python-version` (string, default `""`): when non-empty, `actions/setup-python` step

Existing inputs (`coverage-artifact`, `coverprofile`, `max-crapload`, etc.) unchanged. `coverprofile` retains its default `coverage.out` but is only passed to `gaze report` when `analyzer` is empty.

Rationale: additive inputs keep Go callers byte-identical. The `packages` input generalizes the hard-coded `./...` so Node callers can use `./src/...` to exclude `.opencode/` and other non-source directories.

### D2: Coverage contract — branch on analyzer presence

- **Go mode** (`analyzer == ""`): retain current behavior — `COVERPROFILE` env from `inputs.coverprofile`, `test -f "$COVERPROFILE"` in both the report path (implicit via gaze's own check) and the skipped-report path (explicit `test -f`), and `gaze report ... --coverprofile="$COVERPROFILE"`.
- **Analyzer mode** (`analyzer != ""`): do NOT pass `--coverprofile`; the analyzer reads coverage from the artifact's contents itself. The artifact handoff contract remains: caller uploads coverage data (e.g., `coverage/` dir), reusable workflow downloads it. The skipped-report path validates artifact arrival by checking that the downloaded artifact directory is non-empty (or that a caller-provided coverage path exists, if we add `coverage-path` input). For v1, the simplest is to drop the `test -f "$COVERPROFILE"` check when `analyzer` is set, and instead check that the artifact directory contains files (or just that the download succeeded — download-artifact already fails if artifact missing, but we need to validate contents).

For the skipped-report path, we need a check that works for both modes. Options:
- Drop the `test -f` check when analyzer is set (as issue suggests) — artifact download success is the check.
- Or add `coverage-path` input checked instead.

Chosen: drop the `test -f` check when `analyzer` is set, and add a conditional check that verifies the artifact download produced files. Concretely, in the skipped-report step:

```
if [ -z "$ANALYZER" ]; then
  test -f "$COVERPROFILE" || { error; exit 1; }
else
  # analyzer mode: verify coverage artifact was downloaded (non-empty download)
  # The artifact extraction already happened; we can check that the workspace has coverage data
  # For now, just ensure at least one file exists (or coverage-path if provided)
fi
```

If we add `coverage-path` later, it can be an additional optional input.

Rationale: minimal change, preserves Go validation, and analyzer mode's artifact shape varies (file vs directory) so a generic "artifact arrived" check is more appropriate than a hard-coded file name.

### D3: `gaze report` invocation — conditional flag assembly

Shell logic in the reusable workflow's `Run gaze AI report` step:

```
ANALYZER_ARG=""
if [ -n "$ANALYZER" ]; then
  ANALYZER_ARG="--analyzer=$ANALYZER"
fi
LANGUAGE_ARG=""
if [ -n "$LANGUAGE" ]; then
  LANGUAGE_ARG="--language=$LANGUAGE"
fi
PACKAGES="${PACKAGES:-./...}"
COVERPROFILE_ARG=""
if [ -z "$ANALYZER" ] && [ -n "$COVERPROFILE" ]; then
  COVERPROFILE_ARG="--coverprofile=$COVERPROFILE"
fi

gaze report $PACKAGES \
  --ai=opencode \
  $MODEL_ARG \
  $ANALYZER_ARG \
  $LANGUAGE_ARG \
  $COVERPROFILE_ARG \
  $THRESHOLDS
```

Rationale: shell-assembled args keep the workflow readable and avoid GitHub Actions expression complexity. `packages` replaces the hard-coded `./...` positional argument.

### D4: Toolchain setup steps — conditional

After checkout, before download-coverage or gaze install:

- `Set up Node` step: `if: inputs.node-version != ''`, uses `actions/setup-node@v4` with `node-version: ${{ inputs.node-version }}`
- `Set up Python` step: `if: inputs.python-version != ''`, uses `actions/setup-python@v5` with `python-version: ${{ inputs.python-version }}`
- `Setup step` (user's `setup` shell): `if: inputs.setup != ''`, `run: ${{ inputs.setup }}` with `shell: bash`

`setup-go` remains unconditional (needed to `go install` gaze).

SHA pins for setup-node/setup-python will be taken from current published action versions at implementation time.

### D5: Scaffold variant detection — `go.mod` else `package.json` else `pyproject.toml`

In `scaffold.Run`:

1. `resolveCIPin` runs first (fail-closed version check, unchanged).
2. `checkGoMod` runs; if `go.mod` exists, `completeCIContext` proceeds as today (Go variant) — `ciVariant = "go"`, render `assets/ci/gaze-ai-report.yml`.
3. If `go.mod` absent, check for `package.json` (file exists at target root). If present, `ciVariant = "node"`, render `assets/ci/gaze-ai-report-node.yml` with `goVersion` still needed? Node variant still needs `goVersion` for the reusable workflow's `setup-go`? Actually, the reusable workflow's test job won't need Go for Node, but the scaffolded Node caller workflow's test job will use Node, not Go. The `goVersion` placeholder is Go-specific; Node template will have `__NODE_VERSION__` placeholder instead, derived from `package.json` engines or default `20`.

Similarly, if `package.json` absent but `pyproject.toml` exists, `ciVariant = "python"`, render `assets/ci/gaze-ai-report-python.yml`.

If none of the three markers exist, `ci.skipped = true` and existing skip warning prints (preserving current warning for unrecognized repos).

Design trade-off: Node/Python templates still need `gazeVersion` for the `uses:` pin. `goVersion` is Go-variant only; Node/Python variants have their own toolchain version placeholders. We can introduce `nodeVersion` and `pythonVersion` fields on `ciContext` or derive them from marker files. For v1, use static defaults: `nodeVersion = "24"`, `pythonVersion = "3.12"` (current LTS), with no file parsing — user edits if needed.

Rationale: simple, deterministic, no file parsing complexity for Node/Python versions. Go version parsing remains for Go variant only.

### D6: New embedded templates — three variants

- `assets/ci/gaze-ai-report.yml` — Go, unchanged (preserves golden file).
- `assets/ci/gaze-ai-report-node.yml` — Node caller:

```
name: Gaze AI Report
on: push/pull_request branches [main]
permissions: contents: read
concurrency: CI-012
jobs:
  test:
    runs-on: ubuntu-latest
    timeout-minutes: 30
    steps:
      - Checkout (pinned)
      - Setup Node (setup-node@SHA, node-version: "__NODE_VERSION__")
      - Run npm ci or npm install + test with coverage (e.g., npm ci && npm test -- --coverage)
      - Upload artifact: coverage/ with name coverage-out
  gaze-ai-report:
    needs: test
    permissions: contents: read
    uses: unbound-force/gaze/...@__GAZE_VERSION__
    with:
      coverage-artifact: coverage-out
      gaze-version: __GAZE_VERSION__
      analyzer: gaze-analyzer-typescript
      packages: ./src/...
      # node-version optionally? Or rely on test job's Node?
```

For Node, the reusable workflow's `node-version` input is not needed in the caller if the analyzer is already installed via `setup` step? But the reusable workflow's `setup` runs in the report job, not the test job. The analyzer needs to be on PATH during `gaze report`. So Node caller could pass `setup: npm ci && npm run build && npm link` or similar to install analyzer in report job. Alternatively, the test job's coverage upload is sufficient — analyzer reads coverage files, not Node runtime? Actually, `gaze-analyzer-typescript` is a Node binary; it needs Node to run. So the report job needs Node too. We can either have the caller pass `node-version` to the reusable workflow so it sets up Node in the report job, or have the caller handle setup via `setup` input.

Chosen: Node caller passes `analyzer`, `packages`, and optionally `node-version` and `setup` to the reusable workflow. The test job handles test+coverage upload; the report job handles analyzer installation via `setup` and `node-version`.

Similarly for Python: `analyzer: gaze-analyzer-python` or `snake-eyes`, `python-version`.

### D7: Template rendering — per-variant placeholders

Each template has its own placeholders:
- Go: `__GAZE_VERSION__`, `__GO_VERSION__`
- Node: `__GAZE_VERSION__`, `__NODE_VERSION__`
- Python: `__GAZE_VERSION__`, `__PYTHON_VERSION__`

`renderCIWorkflowTemplate` generalizes to handle variant-specific placeholders, or we add `renderCIWorkflowTemplateNode` etc. Simpler: one function with multiple placeholder replacements, all validated.

### D8: Tests — new golden files and matrices

- Existing Go golden file test unchanged (must still pass).
- New tests: `TestRenderCIWorkflowTemplate_NodeGoldenFile` and `PythonGoldenFile` compare rendered Node/Python templates against committed golden files.
- New YAML structure tests for Node/Python templates (permissions, concurrency, test job steps, reusable workflow inputs including `analyzer`).
- New variant detection tests: `TestDetectCIVariant` matrix (go.mod present → go, package.json → node, pyproject.toml → python, none → skipped).
- Reusable workflow input validation: ensure new inputs have correct defaults and don't break existing Go inputs.

### D9: Docs

- `docs/guides/ci-integration.md`: add section on external analyzers — how to use `analyzer`/`language`/`packages`/`setup` inputs, example Node caller workflow, Python caller workflow.
- `docs/reference/cli/init.md`: update `--ci` flag docs to mention non-Go detection.
- `README.md`: brief note on external-analyzer CI support.

## Risks

- Node test command varies (`npm test`, `vitest`, `jest`): template chooses a sensible default (`npm ci` + `npm test -- --coverage`), documented as editable. The scaffolded workflow is user-owned, so the user can change it.
- Coverage artifact shape varies: Node's `coverage/` dir vs Go's `coverage.out` file — reusable workflow's analyzer-mode check must handle directory artifacts (check existence of artifact directory, not specific file).
- SHA pins for setup-node/setup-python need to be current at ship time.
- Residual risk — analyzer-mode execution: the reusable workflow's analyzer mode has no automated execution in this repository. The dogfood job (`test.yml` gaze-ai-report) runs Go mode with `build-from-source: true`, and the analyzer-mode shell plumbing (argument assembly, `eval "$ANALYZER_SETUP"`, the download-conclusion arrival check) is verified only by static contract tests pinning the workflow YAML — a shell defect outside the pinned fragments would ship green. This matches repo precedent for workflow changes. A follow-up canary (the `install-canary.yml` precedent) running the reusable workflow once in analyzer mode against a fixture repository would close the gap.

## Constitution Check

Same as proposal: all four principles PASS for same reasons, plus new variants keep tool-owned files correct and golden-file tested.

## Alternatives Considered

- Single template with conditionals: rejected — Go/Node/Python test jobs are too different; separate templates are clearer and each can be golden-file tested independently.
- Detecting Node/Python versions from `package.json` engines / `pyproject.toml` requires-python: deferred — static defaults keep implementation simple; version detection can be added later if demand emerges.
- Adding `coverage-path` input now: deferred — dropping the `test -f` check when analyzer is set satisfies acceptance; `coverage-path` can be added later without breaking change.
