## ADDED Requirements

### Requirement: CRAPload unavailability detection

`EvaluateThresholds` MUST treat a nil `CRAPload` value as "metric unavailable"
and return a FAIL result when `MaxCrapload` is configured. The result MUST
have `Name` containing "(unavailable)", `Actual` set to `nil`, and `Passed`
set to `false`.

#### Scenario: CRAP step fails with MaxCrapload threshold configured

- **GIVEN** the CRAP analysis step failed (returned an error)
- **WHEN** `EvaluateThresholds` is called with `MaxCrapload` set to any value
- **THEN** the result for CRAPload MUST have `Passed = false`
- **AND** the result MUST have `Actual = nil`
- **AND** the result MUST have `Name` containing "(unavailable)"

#### Scenario: CRAPload is zero from successful analysis

- **GIVEN** the CRAP analysis step succeeded with zero CRAPload functions
- **WHEN** `EvaluateThresholds` is called with `MaxCrapload` set to 5
- **THEN** the result for CRAPload MUST have `Passed = true`
- **AND** the result MUST have `Actual = intPtr(0)` (not nil)

### Requirement: AvgContractCoverage unavailability detection

`EvaluateThresholds` MUST treat a nil `AvgContractCoverage` value as "metric
unavailable" and return a FAIL result when `MinContractCoverage` is configured.
The result MUST have `Name` containing "(unavailable)", `Actual` set to `nil`,
and `Passed` set to `false`.

#### Scenario: Quality step fails with MinContractCoverage threshold configured

- **GIVEN** the Quality analysis step failed (returned an error)
- **WHEN** `EvaluateThresholds` is called with `MinContractCoverage` set to any value
- **THEN** the result for AvgContractCoverage MUST have `Passed = false`
- **AND** the result MUST have `Actual = nil`
- **AND** the result MUST have `Name` containing "(unavailable)"

#### Scenario: AvgContractCoverage is zero from successful analysis

- **GIVEN** the Quality analysis step succeeded with zero average contract coverage
- **WHEN** `EvaluateThresholds` is called with `MinContractCoverage` set to 50
- **THEN** the result for AvgContractCoverage MUST have `Passed = false`
- **AND** the result MUST have `Actual = intPtr(0)` (not nil)

### Requirement: Run returns error on unavailable threshold data

`Run` MUST return a non-nil error when any configured threshold cannot be
evaluated because the corresponding analysis step failed.

#### Scenario: CRAP step fails with threshold flags set

- **GIVEN** `gaze report` is invoked with `--max-crapload=5`
- **AND** the CRAP analysis step fails (e.g., package does not compile)
- **WHEN** the pipeline completes
- **THEN** `Run` MUST return a non-nil error
- **AND** the exit code MUST be non-zero

#### Scenario: Multiple steps fail with multiple thresholds set

- **GIVEN** `gaze report` is invoked with `--max-crapload=5 --min-contract-coverage=50`
- **AND** both the CRAP and Quality steps fail
- **WHEN** the pipeline completes
- **THEN** `Run` MUST return a non-nil error indicating threshold evaluation failure

## MODIFIED Requirements

### Requirement: CRAPload type in ReportSummary

`ReportSummary.CRAPload` MUST be `*int` (pointer to int) to distinguish
"metric computed as zero" (`intPtr(0)`) from "metric unavailable" (`nil`).

Previously: `CRAPload int` — zero value was ambiguous between "healthy" and
"analysis failed".

### Requirement: AvgContractCoverage type in ReportSummary

`ReportSummary.AvgContractCoverage` MUST be `*int` (pointer to int) to
distinguish "metric computed as zero" (`intPtr(0)`) from "metric unavailable"
(`nil`).

Previously: `AvgContractCoverage int` — zero value was ambiguous between
"zero coverage" and "analysis failed".

### Requirement: Pipeline step result types

`crapStepResult.CRAPload` MUST be `*int`. `qualityStepResult.AvgContractCoverage`
MUST be `*int`. These fields MUST only be set to non-nil values when their
respective steps succeed.

Previously: Both were `int`, set unconditionally.

## REMOVED Requirements

None.
