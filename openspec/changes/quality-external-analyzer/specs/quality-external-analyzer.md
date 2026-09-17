## ADDED Requirements

### Requirement: External Analyzer Quality Reports

When `--analyzer` is provided to `gaze quality`, the command MUST
use the external analyzer's `test_mapping` protocol method to produce
quality reports instead of the Go-native quality pipeline.

#### Scenario: Quality report with test_mapping support

- **GIVEN** an external analyzer binary that supports `test_mapping`
- **WHEN** the user runs `gaze quality --analyzer snake-eyes --language python -- ./...`
- **THEN** the command MUST produce a quality report with per-test-function
  contract coverage, gap hints, unmapped assertions, and over-specification
  metrics derived from the analyzer's assertion mapping data

#### Scenario: Quality report JSON output

- **GIVEN** an external analyzer binary that supports `test_mapping`
- **WHEN** the user runs `gaze quality --analyzer snake-eyes --format json -- ./...`
- **THEN** the JSON output MUST conform to the existing quality JSON schema
  with `quality_reports` array and `summary` object

#### Scenario: Quality report text output

- **GIVEN** an external analyzer binary that supports `test_mapping`
- **WHEN** the user runs `gaze quality --analyzer snake-eyes -- ./...`
- **THEN** the text output MUST include the same sections as Go-native
  quality output: contract coverage, gap hints, and over-specification

### Requirement: Graceful Degradation Without test_mapping

When the external analyzer does not support `test_mapping`, the quality
command MUST degrade gracefully.

#### Scenario: Analyzer without test_mapping capability

- **GIVEN** an external analyzer binary where `Capabilities.TestMapping` is `false`
- **WHEN** the user runs `gaze quality --analyzer <name> -- ./...`
- **THEN** the command MUST print a warning to stderr indicating
  test_mapping is unavailable
- **AND** the command MUST produce a quality report with zero contract
  coverage and reason "test_mapping_unavailable"
- **AND** the command MUST exit with code 0 (no threshold flags set)

#### Scenario: Analyzer with test_mapping failure

- **GIVEN** an external analyzer binary where `test_mapping` is supported
  but the method call returns an error
- **WHEN** the user runs `gaze quality --analyzer <name> -- ./...`
- **THEN** the command MUST print a warning to stderr
- **AND** the command MUST produce a quality report with zero contract
  coverage and reason `"test_mapping_error"` (graceful degradation, not a hard error)
- **AND** the command MUST exit with code 0 (no threshold flags set)
- **AND** the command MUST exit with a non-zero exit code if threshold
  flags (`--min-contract-coverage`, `--max-over-specification`) are set
  and the zero-coverage result violates them

### Requirement: Flag Validation With External Analyzers

Certain Go-specific flags MUST be rejected when `--analyzer` is active.

#### Scenario: --target with --analyzer

- **GIVEN** an external analyzer is specified
- **WHEN** the user runs `gaze quality --analyzer <name> --target FuncName -- ./...`
- **THEN** the command MUST return an error explaining that `--target`
  is not supported with external analyzers

#### Scenario: --ai-mapper with --analyzer

- **GIVEN** an external analyzer is specified
- **WHEN** the user runs `gaze quality --analyzer <name> --ai-mapper claude -- ./...`
- **THEN** the command MUST return an error explaining that `--ai-mapper`
  is not supported with external analyzers

### Requirement: CI Threshold Evaluation

Threshold flags MUST work with external analyzer quality reports.

#### Scenario: --min-contract-coverage with external analyzer

- **GIVEN** an external analyzer producing quality reports
- **WHEN** the user runs `gaze quality --analyzer <name> --min-contract-coverage 50 -- ./...`
- **AND** the computed contract coverage is below 50%
- **THEN** the command MUST exit with a non-zero exit code

#### Scenario: --max-over-specification with external analyzer

- **GIVEN** an external analyzer producing quality reports
- **WHEN** the user runs `gaze quality --analyzer <name> --max-over-specification 10 -- ./...`
- **AND** the computed over-specification exceeds 10
- **THEN** the command MUST exit with a non-zero exit code

### Requirement: Unhide Analyzer Flags

The `--analyzer` and `--language` flags on `gaze quality` MUST be
visible in help output.

#### Scenario: Flag visibility in help

- **GIVEN** the user runs `gaze quality --help`
- **WHEN** the help output is displayed
- **THEN** `--analyzer` and `--language` MUST appear in the flags list
  (no longer hidden)

## MODIFIED Requirements

### Requirement: Quality Command Error Message

Previously: `--analyzer is not yet supported for 'gaze quality'`

The quality command MUST accept `--analyzer` and route to the external
analyzer quality pipeline instead of returning an error.

## REMOVED Requirements

None.
