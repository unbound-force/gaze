## MODIFIED Requirements

### Requirement: Gaze repository .uf registration

The gaze repository's `.uf` review config MUST register
`divisor-gaze-testing`: the capabilities file gains a
`test-quality` review entry, and the matrix advisor lists swap
`divisor-testing` for `divisor-gaze-testing` in the `code`,
`spec`, `triage`, and `feedback` modes. The generic
`divisor-testing` capabilities entry and the `uf.review-council.md`
persona table MUST remain unchanged.

#### Scenario: Capabilities entry appended

- **GIVEN** `.uf/reviewer-capabilities.yaml`
- **WHEN** the registration is applied
- **THEN** a `divisor-gaze-testing` reviewer entry with
  capability `review` and scopes `[test-quality]` MUST exist
  alongside the unchanged `divisor-testing` entry

#### Scenario: Advisor swap in four modes

- **GIVEN** `.uf/review-matrix.yaml`
- **WHEN** the registration is applied
- **THEN** the `code`, `spec`, `triage`, and `feedback` advisor
  lists MUST contain `divisor-gaze-testing` and MUST NOT contain
  `divisor-testing`

#### Scenario: Review council inclusion

- **GIVEN** the registered persona
- **WHEN** a review council, triage, or dispatch plan is produced
  for the gaze repository
- **THEN** `divisor-gaze-testing` MUST participate as an
  advisor-sourced test-quality reviewer
