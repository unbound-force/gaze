## Context

Issue #102 reports that `gaze report` exits 0 (PASS) when an analysis step
fails and threshold flags are configured. The root cause is a type asymmetry:
`GazeCRAPload` is `*int` (nil = unavailable, correctly fails gate per #108),
while `CRAPload` and `AvgContractCoverage` are bare `int` (zero value = passes
gate silently).

The `ci-gate-integrity` change (issues #101, #108, #116) established the `*int`
pattern for `GazeCRAPload` but did not extend it to `CRAPload` and
`AvgContractCoverage`. This design completes that migration.

## Goals / Non-Goals

### Goals
- Eliminate the zero-value masquerade for `CRAPload` and `AvgContractCoverage`
  by converting both to `*int` in all pipeline structs
- Make `EvaluateThresholds` fail when a configured threshold's metric is
  unavailable (nil), matching existing `GazeCRAPload` behavior
- Ensure `Run` returns non-nil error when any threshold fails due to
  unavailable data
- Preserve the partial-failure pipeline design — steps that fail should still
  produce partial payloads for the AI adapter

### Non-Goals
- Changing `runProductionPipeline` to return errors on step failure (this would
  break the intentional partial-failure design)
- Adding new threshold flags or metrics
- Modifying the JSON output schema for `--format=json` (the `compact.go`
  structs need mechanical `*int` updates but the serialized field names and
  semantics remain the same)

## Decisions

### D1: Mirror the GazeCRAPload `*int` pattern

**Decision**: Convert `CRAPload` and `AvgContractCoverage` from `int` to `*int`
in `ReportSummary`, `crapStepResult`, `qualityStepResult`, and the compact
structs.

**Rationale**: The `*int` pattern was proven in #108 and is already well-tested.
It makes the nil-vs-zero distinction structural (enforced by the type system)
rather than behavioral (enforced by error-checking conventions). The existing
nil-check code in `EvaluateThresholds` for `GazeCRAPload` (threshold.go:48-57)
serves as an exact template.

**Alternatives rejected**:
- **Check `payload.Errors` in `EvaluateThresholds`**: Couples threshold
  evaluation to the error reporting structure. Requires maintaining the
  invariant "error set implies summary field unreliable" outside the type
  system — fragile and easy to break when adding new metrics.
- **Make `runProductionPipeline` return error**: Breaks the intentional
  partial-failure design. The AI adapter still benefits from classification and
  docscan data even when CRAP fails.

### D2: Set `*int` fields only in the success branch

**Decision**: In `runCRAPStep` and `runQualityForPackage`, the `*int` fields
are set only when the step succeeds. On failure, they remain `nil` (Go default
for pointer types).

**Rationale**: This is already the pattern for `GazeCRAPload` in
`runCRAPStep`. The field is set to `&summary.GazeCRAPload` only inside the
success path. `CRAPload` and `AvgContractCoverage` should follow the same
pattern.

### D3: Update existing bug-enshrining test

**Decision**: `TestRun_ZeroResults_CRAPStepFailed_NoGate` must be updated
from asserting `err == nil` to asserting `err != nil` with a message indicating
threshold evaluation failure due to unavailable data.

**Rationale**: This test currently documents the bug as correct behavior. The
test name suggests it validates the zero-result gate bypass, but it actually
validates the false-pass scenario. After the fix, a failed CRAP step with
thresholds configured must produce a non-nil error.

### D4: Compact struct alignment

**Decision**: Update `compactSummary.CRAPload` from `int` to `*int` and
`compactSummary.AvgContractCoverage` to match (already `*float64` in one
location, `int` in another — align to `*int`).

**Rationale**: The compact structs are the JSON-serialization boundary. The
`omitempty` tag already exists on `GazeCRAPload` and should be applied
consistently to `CRAPload` and `AvgContractCoverage` in the compact path.

## Risks / Trade-offs

### Risk: CI pipelines that were falsely passing will start failing

**Likelihood**: Medium — any pipeline where the CRAP step intermittently fails
(e.g., flaky compilation in analyzed packages) will see new failures.

**Mitigation**: This is the correct behavior. The proposal and issue both
identify this as the intended outcome. Release notes should call out this
behavioral change explicitly.

### Risk: `*int` type change propagation

**Likelihood**: Certain — the change touches `ReportSummary`, `crapStepResult`,
`qualityStepResult`, compact structs, and all test files that construct these
structs.

**Mitigation**: The blast radius is entirely within `internal/aireport/`. The
`intPtr()` helper already exists. Grep for `CRAPload` and `AvgContractCoverage`
struct literal assignments to find all callsites (Dewey learning from #108:
"grep for all struct literals containing the field name").

### Trade-off: `*int` adds nil-check complexity

Every read of `CRAPload` or `AvgContractCoverage` now requires a nil check.
This is accepted because:
1. The alternative (bare `int`) silently passes CI gates — a worse outcome
2. The pattern is already established for `GazeCRAPload` with no ergonomic issues
3. The `intPtr()` helper makes construction clean
