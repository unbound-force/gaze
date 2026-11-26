# Tasks: CI support for external analyzers

## 1. Reusable workflow — optional external-analyzer inputs

- [x] 1.1 In `.github/workflows/reusable_gaze_ai_report.yml`: add new `workflow_call` inputs with defaults preserving Go behavior — `analyzer` (string, default `""`), `language` (string, default `""`), `packages` (string, default `"./..."`), `setup` (string, default `""`), `node-version` (string, default `""`), `python-version` (string, default `""`). Keep existing inputs (`model`, `coverage-artifact`, `coverprofile`, `max-crapload`, etc.) unchanged. Ensure Go caller's inputs still work unchanged.
- [x] 1.2 In `.github/workflows/reusable_gaze_ai_report.yml`: add conditional toolchain setup steps after `Set up Go` — `Set up Node` (when `node-version` non-empty, `actions/setup-node` pinned), `Set up Python` (when `python-version` non-empty, `actions/setup-python` pinned), and `Setup analyzer` shell step (when `setup` non-empty, `run: ${{ inputs.setup }}` with `shell: bash`). Keep `setup-go` unconditional.
- [x] 1.3 In `.github/workflows/reusable_gaze_ai_report.yml`: modify `Run gaze AI report` step to pass `analyzer`, `language`, and `packages` through — shell-assemble `ANALYZER_ARG`, `LANGUAGE_ARG`, `PACKAGES` (from `inputs.packages`), and conditional `COVERPROFILE_ARG` (only when `analyzer` is empty). Change `gaze report ./...` to `gaze report $PACKAGES` with the assembled args. `--coverprofile` only in Go mode.
- [x] 1.4 In `.github/workflows/reusable_gaze_ai_report.yml`: modify `Report skipped` step to handle both modes — when `analyzer` is empty, `test -f "$COVERPROFILE"` as today; when `analyzer` is non-empty, drop the file check (or check that artifact download produced files). Ensure skipped-report path still validates artifact arrival for both modes.

## 2. Scaffold — Node and Python caller templates

- [x] 2.1 Create `internal/scaffold/assets/ci/gaze-ai-report-node.yml` template: provenance header, `name: Gaze AI Report`, `on: push/pull_request` [main], workflow-level `permissions: contents: read`, CI-012 concurrency, `test` job (checkout pinned, setup-node with `__NODE_VERSION__`, `npm ci` + `npm test -- --coverage` or equivalent, upload `coverage/` as `coverage-out`), `gaze-ai-report` job (needs test, permissions, uses `__GAZE_VERSION__`, with `coverage-artifact: coverage-out`, `gaze-version: __GAZE_VERSION__`, `analyzer: gaze-analyzer-typescript`, `packages: ./src/...`, commented thresholds, `secrets: opencode-config`). Must pass yamllint.
- [x] 2.2 Create `internal/scaffold/assets/ci/gaze-ai-report-python.yml` template: same structure for Python — setup-python with `__PYTHON_VERSION__`, test command (e.g., `pip install` + `pytest --cov`), upload coverage artifact, caller with `analyzer` for snake-eyes. Must pass yamllint.
- [x] 2.3 In `internal/scaffold/scaffold.go`: add variant detection — when `go.mod` absent, check for `package.json` → `node` variant, `pyproject.toml` → `python` variant, else skip with existing warning. Add `ciVariant` type, `detectCIVariant` helper, per-variant rendering (Go uses `__GO_VERSION__`, Node uses `__NODE_VERSION__`, Python uses `__PYTHON_VERSION__`), and update `resolveCIVariant` / `completeCIContext` to handle new variants. Keep Go golden file unchanged.
- [x] 2.4 Create golden files: `internal/scaffold/testdata/golden-gaze-ai-report-node.yml` and `golden-gaze-ai-report-python.yml` rendered from templates with fixed versions (e.g., `gaze v2.5.0`, `node 20`, `python 3.12`).

## 3. Scaffold tests and reusable workflow validation

- [x] 3.1 In `internal/scaffold/ci_test.go`: add variant detection matrix test (`TestDetectCIVariant`), Node/Python golden-file tests (`TestRenderCIWorkflowTemplate_NodeGoldenFile`, `PythonGoldenFile`), Node/Python YAML structure tests (permissions, concurrency, test job steps, reusable workflow inputs including `analyzer`), and ensure existing Go tests still pass unchanged.
- [x] 3.2 In `.github/workflows/reusable_gaze_ai_report.yml` validation (if applicable): ensure new inputs have correct types/defaults and don't break Go callers; add fixture job or documentation showing the workflow runs on a TypeScript repository (or manual verification via reading-stone).

## 4. Docs

- [x] 4.1 In `docs/guides/ci-integration.md`: add section on external analyzers — explain `analyzer`/`language`/`packages`/`setup`/`node-version`/`python-version` inputs, show example Node caller workflow (from scaffolding), Python caller workflow, and coverage contract difference (analyzer reads its own coverage data).
- [x] 4.2 In `docs/reference/cli/init.md`: update `--ci` flag docs to describe non-Go marker detection (`package.json` → Node, `pyproject.toml` → Python) and that `go.mod` absence no longer always skips.
- [x] 4.3 In `README.md` (if applicable): brief note on external-analyzer CI support.

## 5. Verification

- [x] 5.1 Verify Go caller's behavior and existing golden file unchanged: run `go test ./internal/scaffold -run TestRenderCIWorkflowTemplate_GoldenFile` and `go test -race -count=1 -short ./...`.
- [x] 5.2 In a temp dir with only `package.json` and no `go.mod`, run `gaze init --ci --gaze-version v2.5.0` and verify it writes a Node caller workflow with `analyzer: gaze-analyzer-typescript`, report-only thresholds commented out, and correct `uses:` pin.
- [x] 5.3 Verify reusable workflow passes `--analyzer`, `--language`, and package pattern through (inspect workflow YAML and, if possible, run on a fixture TypeScript repo or via dogfood job).
- [x] 5.4 Verify skipped-report path still checks coverage artifact for both modes (inspect workflow's skipped-report step for conditional check).
