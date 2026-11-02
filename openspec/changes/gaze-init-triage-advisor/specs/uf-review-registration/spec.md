## MODIFIED Requirements

### Requirement: Idempotent .uf registration merge

When the target project contains `.uf/reviewer-capabilities.yaml`
and `.uf/review-matrix.yaml`, `gaze init` MUST merge the persona
registration into them: append the reviewer entry
`{agent: divisor-gaze-testing, capability: review, scopes:
[test-quality]}` to the capabilities file if absent, and in the
matrix file remove `divisor-testing` from and add
`divisor-gaze-testing` to the `code`, `spec`, `triage`, and
`feedback` advisor lists. When any of those four advisor lists is
missing or is not a sequence, `gaze init` MUST skip the matrix
file with a warning naming the missing key and MUST NOT apply a
partial registration. The merge MUST be idempotent — a second run
MUST produce no changes.

#### Scenario: Existing .uf config is merged

- **GIVEN** a target project with both `.uf` files that do not
  yet reference `divisor-gaze-testing`
- **WHEN** `gaze init` runs
- **THEN** the capabilities file MUST contain the
  `divisor-gaze-testing` reviewer entry
- **AND** the matrix advisor lists for `code`, `spec`, `triage`,
  and `feedback` MUST contain `divisor-gaze-testing` and MUST NOT
  contain `divisor-testing`

#### Scenario: Matrix without a triage list

- **GIVEN** a target project whose `.uf/review-matrix.yaml` has
  `code`, `spec`, and `feedback` advisor lists but no `triage`
  list
- **WHEN** `gaze init` runs
- **THEN** the matrix file MUST be byte-identical to its prior
  content
- **AND** the console output MUST warn that `advisor.triage` is
  missing or invalid

#### Scenario: Idempotent second run

- **GIVEN** a target project already registered by `gaze init`
- **WHEN** `gaze init` runs again
- **THEN** both `.uf` files MUST be byte-identical (no rewrite,
  no duplicate entries)
