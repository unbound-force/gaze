# CI Workflow Requirements

## ADDED Requirements

### Requirement: Reusable AI report default MUST match its release tag

The `gaze-version` input default in
`.github/workflows/reusable_gaze_ai_report.yml` MUST equal the release tag
of the commit that ships it. For the v2.4.0 release, the default MUST be
`v2.4.0`.

#### Scenario: Pinned caller omits gaze-version

- **GIVEN** a caller with `uses: unbound-force/gaze/.github/workflows/reusable_gaze_ai_report.yml@v2.4.0`
- **AND** the caller does not set `gaze-version` or `build-from-source`
- **WHEN** the reusable workflow job runs
- **THEN** gaze is installed with
  `go install github.com/unbound-force/gaze/v2/cmd/gaze@v2.4.0`

### Requirement: Release MUST fail when the default does not match the tag

`release.yml` MUST run an offline gate before preflight that fails unless the
reusable AI-report workflow's `gaze-version` default equals the release tag
exactly, pre-release suffix included.

#### Scenario: Stale default blocks the release

- **GIVEN** the reusable workflow's `gaze-version` default is `v2.4.0`
- **WHEN** `release.yml` runs with tag `v2.5.0`
- **THEN** the `validate-ai-report-default` job fails with an `::error::`
  naming both versions
- **AND** preflight and the release do not run

#### Scenario: Unparsable default fails closed

- **GIVEN** the workflow has zero or more than one `gaze-version` default
- **WHEN** the gate runs
- **THEN** it exits with status 2 instead of passing

## MODIFIED Requirements

None.

## REMOVED Requirements

None.
