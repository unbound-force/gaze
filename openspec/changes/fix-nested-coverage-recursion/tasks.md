<!--
  Coverage Strategy:
  - Unit (internal): generateCoverProfile refuses under GAZE_COVERAGE_RUN; no profile, no spawn
  - Unit (provider): Coverage returns ErrNestedCoverageRun; pre-generated profile still parses
  - End-to-end (manual): full non-short go test ./... terminates with at most one nested level
-->

## 1. Provider: enforce the guard

- [x] 1.1 In `internal/provider/goprovider/coverage.go`, add the `coverageRunEnv` constant and the exported `ErrNestedCoverageRun` sentinel. Use the constant in `buildGoTestCmd`.
- [x] 1.2 At the top of `generateCoverProfile`, return `ErrNestedCoverageRun` when `os.Getenv(coverageRunEnv) != ""`, before `createTempProfile`.
- [x] 1.3 Update the `runGoTestCoverage` doc comment to describe the enforced guard.
- [x] 1.4 Add `TestGenerateCoverProfile_NestedRunRefused` to `coverage_test.go`: sentinel error, empty path, empty `TMPDIR`, `PATH` cleared.
- [x] 1.5 Add `TestCoverage_NestedRunReturnsError` and `TestCoverage_NestedRunUsesPreGeneratedProfile` to `goprovider_test.go`.

## 2. Tests: skip self-analyzing tests under the guard

- [x] 2.1 [P] Add the `GAZE_COVERAGE_RUN` skip to `TestSC005_AnalysisPerformance` in `cmd/gaze/main_test.go`.
- [x] 2.2 [P] Add the `GAZE_COVERAGE_RUN` skip to `TestRunCRAPStep_RealPackage` and `TestRunProductionPipeline_RealPackage` in `internal/aireport/runner_steps_test.go`.

## 3. Verification

- [x] 3.1 `go build ./...` and `go vet ./...`.
- [x] 3.2 `go test -race -count=1 -short -timeout 15m ./...` (CI unit job).
- [x] 3.3 `go test -race -count=1 -run 'TestRunSelfCheck' -timeout 30m ./cmd/gaze/...` (CI self-check job).
- [x] 3.4 Full non-short `go test -count=1 ./...` terminates with at most one nested `go test -coverprofile` level. Only the two failures already present on `main` remain (see design Non-Goals).
