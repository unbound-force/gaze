# CI Workflow — External Analyzer Support

## ADDED Requirements

### Requirement: Reusable workflow MUST support optional external-analyzer inputs

The reusable gaze AI-report workflow MUST accept optional `workflow_call` inputs `analyzer` (string, default `""`), `language` (string, default `""`), `packages` (string, default `"./..."`), `setup` (string, default `""`), `node-version` (string, default `""`), and `python-version` (string, default `""`). When `analyzer` is empty, the workflow MUST behave identically to today (Go-native analysis with `coverprofile`).

#### Scenario: Go caller unchanged

- **GIVEN** a caller that passes only `coverage-artifact` and `coverprofile` (no `analyzer`)
- **WHEN** the reusable workflow runs
- **THEN** it behaves exactly as before: `gaze report ./... --coverprofile=coverage.out` with no `--analyzer`/`--language` flags

#### Scenario: Analyzer input passed through

- **GIVEN** a caller that passes `analyzer: gaze-analyzer-typescript`, `packages: ./src/...`
- **WHEN** the reusable workflow runs its `Run gaze AI report` step
- **THEN** it executes `gaze report ./src/... --analyzer=gaze-analyzer-typescript` (plus optional `--language` and without `--coverprofile`)

#### Scenario: Packages input replaces hard-coded pattern

- **GIVEN** `packages: ./src/...`
- **WHEN** the report command is constructed
- **THEN** the positional argument is `./src/...`, not `./...`

### Requirement: Reusable workflow MUST conditionally set up analyzer toolchains

When `node-version` is non-empty, the workflow MUST run `actions/setup-node` with that version. When `python-version` is non-empty, it MUST run `actions/setup-python`. When `setup` is non-empty, it MUST run that shell snippet before `gaze report` (to install/link the analyzer).

#### Scenario: Node toolchain setup skipped by default

- **GIVEN** `node-version: ""` (default)
- **WHEN** the workflow runs
- **THEN** no `setup-node` step executes

#### Scenario: Setup snippet runs when provided

- **GIVEN** `setup: "npm ci && npm run build && npm link"`
- **WHEN** the workflow reaches the setup step
- **THEN** that shell command executes with `bash`

### Requirement: Coverage contract MUST branch on analyzer presence

`--coverprofile` MUST be passed only in Go mode (when `analyzer` is empty). In analyzer mode, the workflow MUST NOT pass `--coverprofile` (the analyzer reads coverage data from the artifact's contents). The skipped-report path MUST still validate that the coverage artifact arrived, for both modes.

#### Scenario: Go mode passes coverprofile

- **GIVEN** `analyzer: ""`, `coverprofile: coverage.out`
- **WHEN** the report command runs
- **THEN** it includes `--coverprofile=coverage.out` and the skipped path checks `test -f "$COVERPROFILE"`

#### Scenario: Analyzer mode omits coverprofile

- **GIVEN** `analyzer: gaze-analyzer-typescript`
- **WHEN** the report command runs
- **THEN** it omits `--coverprofile` and the skipped path does not check for a specific file but still validates artifact arrival

## MODIFIED Requirements

### Requirement: Coverage MUST arrive via artifact handoff

The reusable workflow MUST obtain coverage data by downloading the caller-named artifact from the calling workflow run. In Go mode it MUST accept the path via the `coverprofile` input (default `coverage.out`). In analyzer mode the artifact holds whatever that analyzer reads, and the `coverprofile` input is ignored.

#### Scenario: Caller uploads coverage.out (Go)

- **GIVEN** the calling workflow uploaded an artifact containing `coverage.out`
- **WHEN** the reusable workflow job runs in Go mode
- **THEN** the artifact is downloaded and the report runs with `--coverprofile` pointing at the extracted file

#### Scenario: Caller uploads coverage/ (Node)

- **GIVEN** the calling workflow uploaded an artifact containing `coverage/coverage-final.json`
- **WHEN** the reusable workflow job runs with `analyzer: gaze-analyzer-typescript`
- **THEN** the artifact is downloaded and the report runs without `--coverprofile`, with the analyzer reading `coverage/coverage-final.json` from the workspace

## REMOVED Requirements

None — the reusable workflow itself, its contract, and existing inputs remain.

