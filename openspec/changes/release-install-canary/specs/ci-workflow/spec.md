# CI Workflow Requirements

## ADDED Requirements

### Requirement: Consumer install path MUST be exercised against the latest release

The repository MUST run a scheduled workflow
(`.github/workflows/install-canary.yml`) that installs the latest published
GitHub release of gaze with the same commands as the reusable AI-report
workflow's `Install gaze (release)` step, on that workflow's default Go
toolchain, and fails when the install fails, when `gaze --version` fails, or
when the installed module version differs from the release tag. The workflow
MUST also be runnable on demand via `workflow_dispatch`, MUST NOT check out
the repository, and MUST NOT require secrets other than `github.token`.

#### Scenario: Latest release installs cleanly

- **GIVEN** the latest GitHub release is `vX.Y.Z`
- **WHEN** the canary runs on its schedule
- **THEN** it runs
  `go install github.com/unbound-force/gaze/v2/cmd/gaze@vX.Y.Z` and
  `gaze --version`
- **AND** the binary's build info reports module
  `github.com/unbound-force/gaze/v2` at `vX.Y.Z`
- **AND** the job succeeds

#### Scenario: Install path is broken

- **GIVEN** `go install` of the latest release fails, or the installed
  module version is not the release tag
- **WHEN** the canary runs
- **THEN** the job fails, and a version mismatch is reported with an
  `::error::` naming the expected tag and the installed version

## MODIFIED Requirements

None.

## REMOVED Requirements

None.
