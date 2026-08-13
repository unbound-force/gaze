<!--
  [P] marks tasks eligible for parallel execution.
  Add [P] when a task: (a) touches different files from
  other [P] tasks in the group, (b) has no dependency
  on prior tasks in the group, (c) can safely execute
  without ordering constraints.
  Do NOT add [P] when tasks modify the same file —
  parallel workers will cause merge conflicts.
  Tasks without [P] run sequentially first, then [P]
  tasks run in parallel.
-->

## 1. Type migration: `int` → `*int`

Convert `CRAPload` and `AvgContractCoverage` from `int` to `*int` in all
pipeline structs. This is the foundational change — all subsequent tasks
depend on it.

- [ ] 1.1 Change `ReportSummary.CRAPload` from `int` to `*int` and
  `ReportSummary.AvgContractCoverage` from `int` to `*int` in
  `internal/aireport/payload.go`. Update GoDoc comments to match
  the `GazeCRAPload` pattern ("Nil means the metric was not computed").

- [ ] 1.2 Change `crapStepResult.CRAPload` from `int` to `*int` in
  `internal/aireport/runner_steps.go` (line 87). Update the
  `runCRAPStep` success path to use `intPtr(rpt.Summary.CRAPload)`
  (line 130).

- [ ] 1.3 Change `qualityStepResult.AvgContractCoverage` from `int` to
  `*int` in `internal/aireport/runner_steps.go` (line 141). Update the
  quality step success path to use `intPtr(...)` for the computed
  average. **Note**: task 1.2 and 1.3 touch the same file and MUST
  be done together.

- [ ] 1.4 Change `compactSummary.CRAPload` from `int` to `*int` and
  `compactSummary.AvgContractCoverage` from `int` to `*int` in
  `internal/aireport/compact.go` (lines 27, 29). Add `omitempty` JSON
  tags to match `GazeCRAPload` pattern. Update `CompactForAI`
  assignment sites (lines 165, 167) to propagate `*int` values
  correctly. **Note**: `compactCRAPField` (lines 232, 237) copies from
  `crap.Summary` which has its own `CRAPload int` and
  `AvgContractCoverage *float64` — these are different types in a
  different package and are NOT affected by this change.

- [ ] 1.5 Update `runProductionPipeline` in `internal/aireport/runner.go`
  (lines 297, 313) — the assignment `payload.Summary.CRAPload = crapRes.CRAPload`
  already lives in the success branch, so it will naturally propagate the
  `*int`. Confirm that the error branch does NOT set CRAPload — the nil
  default is the desired semantics. Same for `AvgContractCoverage` at
  line 313. **Note**: upstream `crap.Summary.CRAPload` remains `int` — the
  `*int` wrapper is applied at the `aireport` pipeline boundary, not in the
  `crap` package.

## 2. Threshold evaluation: nil-check guards

Add nil-unavailability handling to `EvaluateThresholds` for `CRAPload`
and `AvgContractCoverage`, mirroring the existing `GazeCRAPload`
pattern (threshold.go:48-57).

- [ ] 2.1 Update the `CRAPload` threshold block (threshold.go:32-44) to
  check `summary.CRAPload == nil` first. When nil: append a FAIL result
  with `Name: "CRAPload (unavailable)"`, `Actual: nil`, `Passed: false`.
  When non-nil: dereference and compare as before, using
  `Actual: summary.CRAPload`.

- [ ] 2.2 Update the `AvgContractCoverage` threshold block
  (threshold.go:72-84) to check `summary.AvgContractCoverage == nil`
  first. When nil: append a FAIL result with
  `Name: "AvgContractCoverage (unavailable)"`, `Actual: nil`,
  `Passed: false`. When non-nil: dereference and compare as before.
  **Note**: tasks 2.1 and 2.2 touch the same file and MUST be done
  together.

## 3. Tests: threshold unavailability

Add unit tests for `EvaluateThresholds` covering the new nil-handling
paths.

- [ ] 3.1 Add `TestEvaluateThresholds_CRAPload_Unavailable` to
  `internal/aireport/threshold_test.go`. Verify that when
  `Summary.CRAPload` is nil and `MaxCrapload` is set, the result is
  FAIL with `Actual == nil` and `Name` contains "(unavailable)".

- [ ] 3.2 Add `TestEvaluateThresholds_AvgContractCoverage_Unavailable`
  to `internal/aireport/threshold_test.go`. Same pattern for
  `AvgContractCoverage` + `MinContractCoverage`.

- [ ] 3.3 Add `TestEvaluateThresholds_CRAPload_ZeroIsValid` to
  `internal/aireport/threshold_test.go`. Verify that
  `CRAPload = intPtr(0)` with `MaxCrapload = 5` produces PASS with
  `Actual = intPtr(0)` — confirming zero-from-success is
  distinguishable from nil-from-failure.

- [ ] 3.4 Update `TestEvaluateThresholds_NilPayload` in
  `internal/aireport/threshold_test.go` — with nil payload, all
  summary fields are nil (not zero), so all configured thresholds
  should now FAIL with "(unavailable)". Expected assertion changes:
  `passed=true` → `passed=false`, `results[0].Passed=true` →
  `results[0].Passed=false`, `results[0].Actual == nil`, and
  `results[0].Name` contains "(unavailable)".

  **Note**: tasks 3.1-3.4 all touch `threshold_test.go` and MUST be
  implemented as a single batch to avoid conflicts.

## 4. Tests: integration (Run end-to-end)

Add integration tests verifying `Run` returns error when step failure
+ threshold are combined.

- [ ] 4.1 Update `TestRun_ZeroResults_CRAPStepFailed_NoGate` in
  `internal/aireport/runner_test.go` — change assertion from
  `err == nil` to `err != nil`. The test should verify that when the
  CRAP step fails and `MaxCrapload` is set, `Run` returns an error.
  Update the test name to reflect the new behavior (e.g.,
  `TestRun_CRAPStepFailed_ThresholdSet_ReturnsError`).

- [ ] 4.2 Add `TestRun_QualityStepFailed_ContractCoverageThreshold_ReturnsError`
  to `internal/aireport/runner_test.go`. Verify that when the quality
  step fails and `MinContractCoverage` is set, `Run` returns an error.

- [ ] 4.3 Add `TestRun_CRAPStepFailed_NoThreshold_ReturnsNil` to
  `internal/aireport/runner_test.go`. Verify that when the CRAP step
  fails but NO threshold flags are set, `Run` still returns nil
  (preserving the partial-failure design).

## 5. Tests: pipeline internal

Update pipeline internal tests to reflect `*int` types.

- [ ] 5.1 Update `fakeSteps` and assertions in
  `internal/aireport/pipeline_internal_test.go` to use `*int` for
  `CRAPload` and `AvgContractCoverage`. Verify existing tests compile
  and pass with the type changes.

- [ ] 5.2 Add `TestRunProductionPipeline_CRAPStepFails_CRAPloadIsNil`
  to `internal/aireport/pipeline_internal_test.go`. Verify that when
  the CRAP step fails, `payload.Summary.CRAPload` is nil (not zero).

## 6. Verification

- [ ] 6.1 Run `go build ./cmd/gaze` — verify clean compilation.
- [ ] 6.2 Run `go test -race -count=1 -short ./internal/aireport/...` —
  verify all tests pass.
- [ ] 6.3 Run `golangci-lint run` — verify no lint violations.
- [ ] 6.4 Run `go test -race -count=1 -short ./...` — verify no
  regressions outside `internal/aireport/`.
- [ ] 6.5 Constitution alignment verification: confirm Principle I
  (Accuracy — no false positives from zero-value masquerade),
  Principle III (Actionable Output — threshold results distinguish
  zero from unavailable), and Principle IV (Testability — all new
  paths are covered by tests).
