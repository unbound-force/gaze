# CI Scaffolding — External Analyzer Support

## ADDED Requirements

### Requirement: Scaffold MUST detect non-Go project markers

When `gaze init --ci` is run in a directory with no `go.mod` at the repository root, it MUST check for `package.json` (Node) and `pyproject.toml` (Python) before skipping. If `package.json` exists, it MUST scaffold a Node caller workflow. If `package.json` is absent but `pyproject.toml` exists, it MUST scaffold a Python caller workflow. Only when none of the three markers exist MUST it skip with the existing warning.

#### Scenario: Node project scaffolds Node workflow

- **GIVEN** a target directory with `package.json` and no `go.mod`
- **WHEN** `gaze init --ci --gaze-version v2.5.0` runs
- **THEN** it writes `.github/workflows/gaze-ai-report.yml` containing a Node test job (setup-node, npm install/test with coverage, upload `coverage/`), and a `gaze-ai-report` job calling the reusable workflow with `analyzer: gaze-analyzer-typescript` and `packages: ./src/...`

#### Scenario: Python project scaffolds Python workflow

- **GIVEN** a target directory with `pyproject.toml` and no `go.mod`
- **WHEN** `gaze init --ci --gaze-version v2.5.0` runs
- **THEN** it writes `.github/workflows/gaze-ai-report.yml` containing a Python test job and a caller with the Python analyzer

#### Scenario: Unrecognized project still warns

- **GIVEN** a target directory with none of `go.mod`, `package.json`, or `pyproject.toml`
- **WHEN** `gaze init --ci` runs
- **THEN** it prints the existing warning ("skipping CI scaffolding (--ci): ...") and writes no workflow, while `.opencode/` scaffolding proceeds

### Requirement: Scaffolded Node/Python workflows MUST be report-only

Node and Python caller workflows MUST have threshold inputs (`max-crapload`, etc.) commented out with TODO markers, same as the Go workflow — report-only by default.

#### Scenario: Node workflow is report-only

- **GIVEN** the generated Node workflow file
- **WHEN** its `gaze-ai-report` job's `with:` block is inspected
- **THEN** `max-crapload`, `max-gaze-crapload`, and `min-contract-coverage` are present only as commented lines with TODO markers

### Requirement: Each scaffold variant MUST have a golden file

Each caller workflow variant (Go, Node, Python) MUST have a committed golden file in `internal/scaffold/testdata/` rendered from the template with fixed versions, and tests MUST verify the rendered output matches the golden file byte-for-byte.

#### Scenario: Go golden file unchanged

- **GIVEN** the existing `golden-gaze-ai-report.yml`
- **WHEN** `TestRenderCIWorkflowTemplate_GoldenFile` runs
- **THEN** it still passes — the Go template and its rendering are unchanged

#### Scenario: Node golden file matches rendered template

- **GIVEN** `internal/scaffold/testdata/golden-gaze-ai-report-node.yml`
- **WHEN** `TestRenderCIWorkflowTemplate_NodeGoldenFile` runs
- **THEN** the rendered Node template matches the golden file

## MODIFIED Requirements

### Requirement: CI scaffolding MUST be opt-in and additive

`gaze init` MUST accept a `--ci` boolean flag. Without `--ci`, the command MUST behave identically to the current implementation. With `--ci`, the command MUST additionally write `.github/workflows/gaze-ai-report.yml` into the target root — using the Go template when `go.mod` exists, the Node template when `package.json` exists, or the Python template when `pyproject.toml` exists — while the existing `.opencode/` scaffolding proceeds unchanged.

#### Scenario: Plain init is unchanged

- **GIVEN** a target directory with no prior gaze scaffolding
- **WHEN** `gaze init` runs without `--ci`
- **THEN** the `.opencode/` assets are scaffolded exactly as before and no `.github/workflows/gaze-ai-report.yml` file is created

#### Scenario: Opt-in emits the workflow (Go)

- **GIVEN** a target directory with a `go.mod` at its root
- **WHEN** `gaze init --ci` runs
- **THEN** the `.opencode/` assets are scaffolded and `.github/workflows/gaze-ai-report.yml` is created from the Go template

#### Scenario: Opt-in emits the workflow (Node)

- **GIVEN** a target directory with `package.json` and no `go.mod`
- **WHEN** `gaze init --ci` runs
- **THEN** the `.opencode/` assets are scaffolded and a Node-variant `.github/workflows/gaze-ai-report.yml` is created

## REMOVED Requirements

None — Go scaffolding behavior is preserved; only new variants are added.
