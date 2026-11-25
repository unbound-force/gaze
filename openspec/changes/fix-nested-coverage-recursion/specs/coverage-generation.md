## ADDED Requirements

### Requirement: Nested coverage generation is refused

`generateCoverProfile` MUST return `ErrNestedCoverageRun` without creating a coverage profile and without spawning `go test` when the `GAZE_COVERAGE_RUN` environment variable is set to any non-empty value.

#### Scenario: Generation inside a coverage collection
- **GIVEN** `GAZE_COVERAGE_RUN` is set in the environment
- **WHEN** `GoLineCoverageProvider.Coverage` is called with an empty `coverProfile`
- **THEN** it MUST return an error for which `errors.Is(err, goprovider.ErrNestedCoverageRun)` is true
- **AND** no `go test` subprocess SHALL be started
- **AND** no temporary coverage profile SHALL be created

#### Scenario: Pre-generated profile inside a coverage collection
- **GIVEN** `GAZE_COVERAGE_RUN` is set in the environment
- **WHEN** `GoLineCoverageProvider.Coverage` is called with a valid `coverProfile`
- **THEN** it MUST parse that profile and return results as it does outside a coverage collection

#### Scenario: Normal generation unaffected
- **GIVEN** `GAZE_COVERAGE_RUN` is not set
- **WHEN** `GoLineCoverageProvider.Coverage` is called with an empty `coverProfile`
- **THEN** it MUST generate coverage via `go test -coverprofile` as before

### Requirement: Self-analyzing tests skip under the recursion guard

Every gaze test that generates coverage through the real `GoLineCoverageProvider` (without a pre-generated profile) MUST skip when `GAZE_COVERAGE_RUN` is non-empty, regardless of which test adds coverage generation in the future. Current examples include `TestRunSelfCheck_TextFormat`, `TestRunSelfCheck_JSONFormat`, `TestSC005_AnalysisPerformance`, `TestRunCRAPStep_RealPackage`, `TestRunProductionPipeline_RealPackage`, and any test that repairs `TestSC002_GazeCRAPloadMatchBetweenCrapAndReport`'s generation leg; the enumeration is illustrative, not exhaustive.

#### Scenario: Full local suite terminates
- **GIVEN** the gaze module
- **WHEN** `go test ./...` is run without `-short`
- **THEN** it MUST terminate, with at most one level of nested `go test -coverprofile`

## MODIFIED Requirements

### Requirement: Self-check test skip guards

Previously: only `TestRunSelfCheck_TextFormat` and `TestRunSelfCheck_JSONFormat` were required to check `GAZE_COVERAGE_RUN`. Recursion prevention depended on every self-analyzing test having that check. Now it is enforced in `generateCoverProfile` (see above), and the test skips keep nested runs from failing and wasting work.

## REMOVED Requirements

None.
