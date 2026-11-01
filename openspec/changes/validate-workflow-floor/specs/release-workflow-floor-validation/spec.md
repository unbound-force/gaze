# Release Workflow Floor Validation Requirements

## ADDED Requirements

### Requirement: Release Tag Consistency With Workflow Floor

The release workflow MUST validate the requested release tag against the
`minWorkflowTag` floor constant embedded in `internal/scaffold/scaffold.go`
before the preflight job creates the release tag.

#### Scenario: Below-floor release tag is rejected

- **GIVEN** the repository contains the `minWorkflowTag` floor constant set to
  `v2.3.0` and the reusable AI-report workflow file in the tree
- **WHEN** a release is dispatched with a tag `v2.2.9` that is strictly below
  the floor
- **THEN** the workflow-floor validation job fails with exit status 1
- **AND** the preflight job does not run and no tag is created

#### Scenario: At-floor release tag passes

- **GIVEN** the repository contains the `minWorkflowTag` floor constant set to
  `v2.3.0` and the reusable AI-report workflow file in the tree
- **WHEN** a release is dispatched with tag `v2.3.0`, equal to the floor
- **THEN** the workflow-floor validation job passes, because this release is
  the one that ships the reusable workflow

#### Scenario: Above-floor release requires a workflow-bearing floor tag

- **GIVEN** the repository contains the `minWorkflowTag` floor constant set to
  `v2.3.0`
- **WHEN** a release is dispatched with tag `v2.4.0`, strictly above the floor
- **THEN** the validation passes only if the floor tag `v2.3.0` exists in the
  local repository AND contains `.github/workflows/reusable_gaze_ai_report.yml`
- **AND** the validation fails with exit status 1 when the floor tag is
  missing or predates the workflow file

### Requirement: Floor Extraction From Source Of Truth

The workflow-floor validator MUST derive the floor value by parsing the
`minWorkflowTag` constant from `internal/scaffold/scaffold.go`, MUST fail when
the constant is present but malformed or duplicated, and MUST skip with a
notice (exit status 0) when the tree predates floor governance and contains no
such constant.

#### Scenario: Legacy tree without the floor constant

- **GIVEN** a repository tree that contains no `minWorkflowTag` constant in
  `internal/scaffold/scaffold.go`
- **WHEN** the validator runs with any well-formed release tag
- **THEN** it exits with status 0 and emits a notice that the tree predates
  floor governance

#### Scenario: Malformed floor constant

- **GIVEN** the `minWorkflowTag` constant is set to a value that does not match
  the strict `vMAJOR.MINOR.PATCH` tag form
- **WHEN** the validator runs
- **THEN** it exits with status 1 rather than guessing the intended floor

#### Scenario: Floor-bearing tree must contain the reusable workflow

- **GIVEN** the tree contains the `minWorkflowTag` constant
- **WHEN** the tree does not contain
  `.github/workflows/reusable_gaze_ai_report.yml`
- **THEN** the validator exits with status 1 regardless of the requested tag

### Requirement: Deterministic And Unskippable Validation

The workflow-floor validation MUST run without network or GitHub API access,
against tags available in the local clone, and MUST NOT be bypassable by any
release skip input.

#### Scenario: Offline verification of the floor tag

- **GIVEN** a release dispatched with a tag above the floor
- **WHEN** the validation job runs
- **THEN** the existence and contents of the floor tag are verified using only
  the local clone (checked out with full history and tags)

### Requirement: CI Guard Against Floor Constant Drift

The CI test suite MUST assert that the repository's own `minWorkflowTag`
constant is present, parseable, and paired with the reusable workflow file in
the tree, so that relocating or removing either forces a validator update
instead of a silent skip.

#### Scenario: Floor constant relocated without validator update

- **GIVEN** the `minWorkflowTag` constant has been moved out of
  `internal/scaffold/scaffold.go`
- **WHEN** the CI test script runs its repository-invariant assertions
- **THEN** the assertions fail, surfacing the drift

## MODIFIED Requirements

None.

## REMOVED Requirements

None.
