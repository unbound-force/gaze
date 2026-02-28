# Quickstart: CRAPload Reduction

**Feature**: 009-crapload-reduction
**Date**: 2026-02-27

## Verification Workflow

This feature is verified entirely through the project's own quality analysis tool. No external services, databases, or configuration changes are needed.

### Step 1: Capture Baseline Metrics

Before any changes, run the quality analysis to capture current scores:

```bash
gaze crap --format=json ./... > baseline.json
```

Key baseline values to record:
- CRAPload: 27
- GazeCRAPload: 7
- `docscan.Filter` GazeCRAP: 72 (Q3)
- `LoadModule` CRAP: 56
- `runCrap` CRAP: 52
- `runSelfCheck` CRAP: 43
- `buildContractCoverageFunc` CRAP: 70
- `AnalyzeP1Effects` GazeCRAP: 32 (Q4)
- `AnalyzeP2Effects` GazeCRAP: 18 (Q4)

### Step 2: Verify After Each User Story

After completing each user story, run the analysis again and compare:

```bash
gaze crap --format=json ./... > after-usN.json
```

#### US1 — docscan.Filter

Check that the function moved from Q3 to Q1:
- GazeCRAP should be below 15
- Quadrant should be Q1_Safe
- Q3 quadrant count should be 0

#### US2 — LoadModule

Check that the CRAP score dropped:
- CRAP should be below 15
- Line coverage should be significantly above 0%

#### US3 — runCrap / runSelfCheck

Check that both functions improved:
- `runCrap` CRAP should be below 20
- `runSelfCheck` CRAP should be below 20
- Line coverage for both should be above 70%

#### US4 — buildContractCoverageFunc

Check that no decomposed function exceeds the threshold:
- No function in the pipeline should have CRAP above 30
- The original `buildContractCoverageFunc` should have lower CC

#### US5 — AnalyzeP1Effects / AnalyzeP2Effects

Check that all handler functions are below threshold:
- No individual handler should have GazeCRAP above 15
- All handlers should be in Q1 or Q2

### Step 3: Verify Overall Targets

After all user stories are complete:

```bash
gaze crap --format=json ./... > final.json
```

Verify success criteria:
- GazeCRAPload ≤ 4 (down from 7)
- CRAPload ≤ 24 (down from 27)
- Q3 quadrant count = 0

### Step 4: Regression Check

Run the full test suite to verify no regressions:

```bash
go test -race -count=1 -short ./...
golangci-lint run
```

Both must pass cleanly.

## Integration Scenarios

### Scenario 1: Developer Runs Quality Report

A developer runs `gaze crap ./...` and sees improved scores compared to the baseline. The "worst GazeCRAP" and "worst CRAP" lists no longer contain the 5 target functions (or they appear with scores below threshold).

**Before**: `docscan.Filter` appears as #1 worst GazeCRAP (72).
**After**: `docscan.Filter` no longer appears in the worst list.

### Scenario 2: CI Pipeline Threshold Check

The CI pipeline runs `gaze crap --max-crapload=24 --max-gaze-crapload=4 ./...` and the command exits with status 0 (thresholds satisfied).

**Before**: This command would exit with status 1 (CRAPload 27 > 24, GazeCRAPload 7 > 4).
**After**: Both thresholds are met.

### Scenario 3: Adding a New P1 Effect Category (Post-Decomposition)

After US5 is complete, a developer wants to add a new P1 effect type (e.g., `SyscallInvocation`). They:

1. Create a new handler function `detectSyscallEffects(...)` following the same signature as existing handlers.
2. Add a case to the type-switch dispatcher in `AnalyzeP1Effects`.
3. Add tests for the new handler.
4. No existing handlers need modification.

This scenario validates FR-022 (open for extension, closed for modification).
