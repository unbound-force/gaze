# CI Workflow Requirements — External Analyzer Mode

<!--
  Requirement numbering allocation (meta-prompt MP-014):
  - This change allocates CW-FR-100 .. CW-FR-105 for ci-workflow.
  - CW-FR-106 and upward are reserved for future ci-workflow
    changes.
  - CW-FR-001 .. CW-FR-099 are held for retrospective numbering of
    the unnumbered requirements already carried by the active
    `reusable-gaze-ai-report` delta, which predates this
    convention; ids here start above that band to avoid
    collisions.
  - Sibling capability: ci-scaffolding requirements (CS-FR-*) live
    in specs/ci-scaffolding/spec.md within this change.
-->

## ADDED Requirements

### Requirement: External-analyzer mode MUST be opt-in with Go-preserving defaults

CW-FR-100

The reusable workflow MUST expose optional external-analyzer
inputs — `analyzer`, `language`, `packages`, `setup`,
`node-version`, `python-version`, and `coverage-path` — with
Go-preserving defaults. When `analyzer` is empty, the workflow MUST
behave exactly as before this change. The `gaze-version` input
default MUST NOT change; the `validate-ai-report-default` release
gate pins it to the release tag.

#### Scenario: Input surface and defaults

- **GIVEN** `.github/workflows/reusable_gaze_ai_report.yml` after
  this change
- **WHEN** its `workflow_call` inputs are inspected
- **THEN** `analyzer`, `language`, `setup`, `node-version`,
  `python-version`, and `coverage-path` are optional strings
  defaulting to empty, and `packages` is an optional string
  defaulting to `./...`

#### Scenario: Go caller is unchanged

- **GIVEN** a caller that passes none of the new inputs
- **WHEN** the reusable workflow runs
- **THEN** the job executes the same steps in the same order with
  the same commands as before this change, and the report runs
  with `--coverprofile` against the extracted coverprofile

#### Scenario: gaze-version default is untouched

- **GIVEN** the workflow file after this change
- **WHEN** its `gaze-version` input default is inspected
- **THEN** the default is exactly the value the release gate
  `validate-ai-report-default` requires for the release tag

### Requirement: Report arguments MUST route by mode

CW-FR-101

When `analyzer` is non-empty, the report step MUST invoke
`gaze report` with `--analyzer` set to the `analyzer` input, with
`--language` set to the `language` input when that input is
non-empty, with the `packages` input in place of the hard-coded
`./...` pattern, and MUST NOT pass `--coverprofile`. When
`analyzer` is empty, the report step MUST keep the current
`--coverprofile` invocation and pass no analyzer flags.

#### Scenario: Analyzer inputs pass through

- **GIVEN** a caller passing `analyzer: gaze-analyzer-typescript`,
  `language: typescript`, and `packages: ./src/...`
- **WHEN** the report step runs
- **THEN** the command carries `--analyzer=gaze-analyzer-typescript`,
  `--language=typescript`, and the pattern `./src/...`, and does
  not carry `--coverprofile`

#### Scenario: Go mode keeps the coverprofile flag

- **GIVEN** a caller leaving `analyzer` empty
- **WHEN** the report step runs
- **THEN** the command carries `--coverprofile` and no analyzer
  flags

### Requirement: The setup input MUST be injection-safe

CW-FR-102

The caller-provided `setup` script MUST reach shell execution only
through an `env:` binding referenced as a quoted POSIX variable,
and workflow input values MUST NOT be interpolated directly into
`run:` script text. The setup script MUST be evaluated in the same
shell as the report invocation so that PATH and environment
changes persist to `gaze report`, and MUST NOT run when the report
is skipped for an absent secret. The workflow MUST document the
caller-trust model.

#### Scenario: Caller-trust model is documented

- **GIVEN** the workflow file after this change
- **WHEN** its header comment is inspected
- **THEN** it documents that `workflow_call` inputs are authored by
  the caller repository's own maintainers — the same principals
  who author the caller's own `run:` steps — not by untrusted
  third parties

#### Scenario: Setup runs through an env binding

- **GIVEN** the workflow file after this change
- **WHEN** every `run:` step that consumes a workflow input is
  inspected
- **THEN** the input arrives via `env:` and the script references
  it as a quoted POSIX variable, with no input interpolation inside
  the `run:` script body

#### Scenario: Setup precedes the report in one shell

- **GIVEN** a caller passing a `setup` script that installs an
  analyzer onto PATH
- **WHEN** the report step runs
- **THEN** the setup script executes first in the same shell and
  `gaze report` finds the analyzer on PATH

#### Scenario: Empty setup is a no-op

- **GIVEN** a caller leaving `setup` empty
- **WHEN** the report step runs
- **THEN** no setup evaluation occurs and the report runs unchanged

### Requirement: Coverage-artifact validation MUST follow the mode

CW-FR-103

In Go mode (`analyzer` empty), the skipped-report path MUST keep
validating the downloaded artifact with a coverprofile file
existence check against the `coverprofile` input. In analyzer mode,
the artifact holds whatever the analyzer reads, so the
skipped-report path MUST NOT require a Go coverprofile file;
instead, whenever `coverage-path` is non-empty, it MUST validate
artifact arrival with an existence check (file or directory)
against `coverage-path`.

#### Scenario: Go mode keeps the file check

- **GIVEN** a Go caller whose artifact contains `coverage.out`
- **WHEN** the report-skipped path runs
- **THEN** the step asserts the coverprofile file exists, exactly
  as before this change

#### Scenario: Analyzer mode validates a coverage directory

- **GIVEN** an analyzer-mode caller whose artifact contains the
  `coverage/` directory and whose `coverage-path` is `coverage/`
- **WHEN** the report-skipped path runs
- **THEN** the step asserts `coverage/` exists inside the
  extracted artifact and succeeds without requiring a coverprofile
  file

#### Scenario: Missing artifact fails in analyzer mode

- **GIVEN** an analyzer-mode caller whose `coverage-path` names a
  path absent from the downloaded artifact
- **WHEN** the report-skipped path runs
- **THEN** the step fails with an error naming the expected path
  and the artifact

### Requirement: Toolchain setup MUST cover analyzer runtimes

CW-FR-104

The `setup-go` step MUST run in every mode, because gaze itself is
installed via `go install`. The workflow MUST provide SHA-pinned
`setup-node` and `setup-python` steps consuming the `node-version`
and `python-version` inputs; each step MUST be skipped when its
input is empty.

#### Scenario: Node toolchain installs when requested

- **GIVEN** an analyzer-mode caller passing `node-version: "22"`
- **WHEN** the workflow runs
- **THEN** a SHA-pinned `setup-node` step configures Node 22 before
  the report step

#### Scenario: Empty toolchain inputs skip their steps

- **GIVEN** a caller leaving `node-version` and `python-version`
  empty
- **WHEN** the workflow runs
- **THEN** neither `setup-node` nor `setup-python` runs, and the
  job's step sequence is otherwise unchanged

### Requirement: The analyzer mode MUST be demonstrated in gaze's CI

CW-FR-105

Gaze's own CI MUST include a fixture job that exercises the
analyzer-mode pipeline end to end on a TypeScript repository
(reading-stone at a pinned ref): analyzer setup, coverage
generation, and `gaze report --analyzer` in report-only mode with
no secret required. The reusable workflow's input contract MUST
additionally be pinned by structural assertions that parse the
workflow file and verify the new inputs, their defaults, the mode
gating, and the `env:`-binding pattern.

#### Scenario: Fixture job produces a TypeScript report

- **GIVEN** gaze's CI after this change
- **WHEN** the fixture job runs
- **THEN** it checks out reading-stone at the pinned ref, runs the
  analyzer-mode steps, and `gaze report --analyzer` exits 0

#### Scenario: Workflow contract is structurally pinned

- **GIVEN** the reusable workflow file
- **WHEN** the structural assertions run
- **THEN** the new inputs, their defaults, the mode gating, and the
  `env:`-binding pattern are verified against the parsed YAML

## MODIFIED Requirements

None.

## REMOVED Requirements

None.
