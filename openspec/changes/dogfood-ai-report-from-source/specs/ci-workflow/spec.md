# CI Workflow Requirements

## ADDED Requirements

### Requirement: Reusable AI report MAY build gaze from source

`.github/workflows/reusable_gaze_ai_report.yml` MUST accept an optional
boolean `build-from-source` input defaulting to `false`. When `false`, the
workflow MUST install the `gaze-version` release exactly as before. When
`true`, the workflow MUST build gaze from the checked-out tree and MUST ignore
`gaze-version`, after verifying the checked-out module is
`github.com/unbound-force/gaze/v2`. Both paths MUST log `gaze --version`.

#### Scenario: Default install is unchanged

- **GIVEN** a caller that does not set `build-from-source`
- **WHEN** the reusable workflow job runs
- **THEN** gaze is installed with
  `go install github.com/unbound-force/gaze/v2/cmd/gaze@<gaze-version>`
- **AND** the job log records the installed binary's `gaze --version`

#### Scenario: Gaze repository builds the commit under test

- **GIVEN** the gaze repository calls the workflow with
  `build-from-source: true`
- **WHEN** the reusable workflow job runs on a push or same-repo pull request
- **THEN** gaze is built from the checked-out commit with
  `go install ./cmd/gaze`
- **AND** `gaze-version` is not used

#### Scenario: Non-gaze caller enabling build-from-source fails fast

- **GIVEN** a repository whose module is not
  `github.com/unbound-force/gaze/v2` calls the workflow with
  `build-from-source: true`
- **WHEN** the install step runs
- **THEN** the job fails with an `::error::` naming the expected and actual
  module paths and directing the caller to `gaze-version`

### Requirement: Dogfood job MUST report with the commit under test

The `gaze-ai-report` job in `.github/workflows/test.yml` MUST call the
reusable workflow with `build-from-source: true`.

#### Scenario: Push to main dogfoods main

- **GIVEN** a push to `main`
- **WHEN** the `gaze-ai-report` job runs
- **THEN** the report is produced by gaze built from the pushed commit, not
  by a pinned release
