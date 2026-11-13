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

## MODIFIED Requirements

None.

## REMOVED Requirements

None.
