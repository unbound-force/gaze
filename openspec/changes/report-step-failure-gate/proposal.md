## Why

`gaze report` serves as a CI quality gate via threshold flags (`--max-crapload`,
`--max-gaze-crapload`, `--min-contract-coverage`). When an analysis step fails
(e.g., a package does not compile), the pipeline records the error in
`payload.Errors` but continues execution. The summary fields `CRAPload` and
`AvgContractCoverage` remain at Go's zero value (`int` → `0`), and threshold
evaluation proceeds against these zero values:

- `CRAPload: 0 <= limit` → **PASS** (false assurance)
- `AvgContractCoverage: 0 >= limit` → **FAIL** (misleading diagnostic —
  reports "coverage too low" instead of "data unavailable")

The `GazeCRAPload` field was already fixed in issue #108 by converting it to
`*int`, where `nil` means "unavailable" and the threshold evaluator correctly
fails when the metric is nil. `CRAPload` and `AvgContractCoverage` were not
updated at the same time, creating a type asymmetry that is the root cause of
this bug.

The existing test `TestRun_ZeroResults_CRAPStepFailed_NoGate` (runner_test.go)
enshrines this behavior as correct — it asserts `err == nil` when the CRAP step
fails with thresholds configured.

**Closes**: #102
**Related**: #108 (GazeCRAPload *int fix), #116 (zero-result gate), #103 (sibling)

## What Changes

Convert `CRAPload` and `AvgContractCoverage` from `int` to `*int` throughout
the `internal/aireport/` pipeline, mirroring the proven `GazeCRAPload` pattern
from issue #108. When an analysis step fails and its threshold is configured,
the gate reports FAIL with "unavailable" instead of silently passing with a
zero value.

## Capabilities

### New Capabilities
- None

### Modified Capabilities
- `threshold evaluation`: When a pipeline step fails and its corresponding
  threshold is configured, the threshold result is FAIL with
  `Actual: nil` and a name containing "(unavailable)" — matching the existing
  `GazeCRAPload` behavior.
- `exit code integrity`: `gaze report` returns non-zero exit code when any
  configured threshold cannot be evaluated due to step failure.

### Removed Capabilities
- None

## Impact

**Files affected** (all within `internal/aireport/`):

| File | Change |
|------|--------|
| `payload.go` | `CRAPload int` → `*int`, `AvgContractCoverage int` → `*int` |
| `threshold.go` | Add nil checks for CRAPload and AvgContractCoverage, mirroring GazeCRAPload |
| `runner_steps.go` | `crapStepResult.CRAPload` → `*int`, `qualityStepResult.AvgContractCoverage` → `*int` |
| `compact.go` | Update nil-safe handling for serialization |
| `runner.go` | Possibly adjust `checkZeroResultGate` |
| `*_test.go` | ~8 new tests, 1 updated test |

**Blast radius**: Entirely within `internal/aireport/` — no public API surface
is affected. `ReportSummary` is tagged `json:"-"` and never serialized to
external consumers.

**Behavioral change**: CI pipelines that were falsely passing due to step
failures will now correctly fail. This is the intended outcome — those pipelines
were providing false assurance.

## Constitution Alignment

Assessed against the Unbound Force org constitution.

### I. Autonomous Collaboration

**Assessment**: N/A

This change is internal to the `gaze` binary and does not affect artifact-based
communication between heroes.

### II. Composability First

**Assessment**: PASS

The change is entirely within `internal/aireport/` and does not introduce new
dependencies or coupling. The `*int` pattern is already established in the same
package.

### III. Observable Quality

**Assessment**: PASS

This fix directly improves observable quality — threshold results will
accurately distinguish "measured zero" from "data unavailable" in machine-parseable
JSON output. The `ThresholdResult.Actual` field (already `*int`) correctly
represents nil vs zero.

### IV. Testability

**Assessment**: PASS

All DI seams needed for testing already exist (`pipelineStepFuncs`, `AnalyzeFunc`,
`fakeAnalyze`). The fix adds ~8 new tests covering the previously-missing
composition scenario (step failure + threshold configured). The existing test
that enshrines the bug will be corrected.
