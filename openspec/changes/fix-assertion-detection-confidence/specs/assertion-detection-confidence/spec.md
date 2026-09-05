## ADDED Requirements

### Requirement: Gaze-side assertion detection confidence for external analyzers

The `internal/adapter` package MUST compute `AssertionDetectionConfidence` from
external analyzer `test_mapping` response data, without requiring protocol changes.

The computation MUST mirror the Go-native `computeDetectionConfidence` semantics:
ratio of recognized assertions to total assertions, as an integer 0-100.

An assertion MUST be considered "recognized" when its `AssertionType` field in
`protocol.AssertionMappingData` is non-empty. An empty `AssertionType` indicates
the external analyzer could not classify the assertion pattern.

Confidence is computed per **target function** (aggregating across all test functions
mapping to that target), matching how `QualityReport` entries are keyed in the
external analyzer path.

#### Scenario: All assertions recognized

- **GIVEN** 5 mappings targeting `pkg/Foo` all with non-empty `AssertionType`
- **WHEN** detection confidence is computed for target `(pkg, Foo)`
- **THEN** the result MUST be 100

#### Scenario: No assertions recognized

- **GIVEN** 3 mappings targeting `pkg/Bar` all with empty `AssertionType`
- **WHEN** detection confidence is computed for target `(pkg, Bar)`
- **THEN** the result MUST be 0

#### Scenario: Mixed recognition

- **GIVEN** 4 mappings targeting `pkg/Mixed`: 3 with `AssertionType = "equality"` and 1 with `AssertionType = ""`
- **WHEN** detection confidence is computed for target `(pkg, Mixed)`
- **THEN** the result MUST be 75 (3 * 100 / 4)

#### Scenario: Integer truncation

- **GIVEN** 3 mappings targeting `pkg/Trunc`: 1 with `AssertionType = "equality"` and 2 with `AssertionType = ""`
- **WHEN** detection confidence is computed for target `(pkg, Trunc)`
- **THEN** the result MUST be 33 (1 * 100 / 3, integer truncation toward zero)

#### Scenario: No mappings for target function

- **GIVEN** mappings exist but none target `(pkg, Unknown)`
- **WHEN** detection confidence is computed for target `(pkg, Unknown)`
- **THEN** the result MUST be 0

#### Scenario: Multiple target functions in mapping data

- **GIVEN** mappings for both target `(pkg, A)` (2 recognized / 2 total) and target `(pkg, B)` (1 recognized / 3 total)
- **WHEN** detection confidence is computed for target `(pkg, A)`
- **THEN** the result MUST be 100 (filters to target `(pkg, A)` mappings only)

#### Scenario: Aggregation across test functions

- **GIVEN** mappings targeting `(pkg, Foo)` from "TestA" (2 with `AssertionType`) and "TestB" (1 without `AssertionType`)
- **WHEN** detection confidence is computed for target `(pkg, Foo)`
- **THEN** the result MUST be 66 (2 * 100 / 3, aggregated across all test functions)

### Requirement: Detection confidence exposed via provider method

`ExternalContractCoverageProvider` MUST expose a `DetectionConfidence(pkg, function string) int`
method that returns the per-function assertion detection confidence computed during `Build`.

The method MUST return 0 for functions not present in the mapping data.
The method MUST NOT panic when called before `Build` (nil map guard).

### Requirement: Quality report pipeline integration

When `gaze quality` runs with an external analyzer (`--analyzer` flag),
`QualityReport.AssertionDetectionConfidence` MUST reflect the adapter-computed value
instead of the default 0.

The confidence MUST be populated directly in the `buildExternalQualityReports`
function (`cmd/gaze/main.go`) during report construction. This function constructs
reports for the external analyzer path — it does not call `quality.Assess`.

Note: `gaze report --analyzer` skips the quality step entirely, so detection
confidence is not relevant in the report pipeline path.

**Invariant**: When `--analyzer` is set, the Go-native `DetectAssertions` always
returns zero assertion sites for non-Go code, so the default 0 is never a valid
computed value — it always indicates "not computed." This makes direct population
(rather than a conditional overlay) safe and unambiguous.

## MODIFIED Requirements

### Requirement: AssertionDetectionConfidence accuracy for all analyzer types

`QualityReport.AssertionDetectionConfidence` MUST accurately reflect assertion
detection capability regardless of whether the Go-native or external analyzer
pipeline produced the report. Previously, external analyzer reports always
showed 0% even when assertions were successfully detected and mapped.

## REMOVED Requirements

None.
