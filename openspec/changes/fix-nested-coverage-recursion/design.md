## Context

`buildGoTestCmd` sets `GAZE_COVERAGE_RUN=1` on the `go test -coverprofile` subprocess (`fix-hardcoded-short-flag`, D1). That design relied on every self-analyzing test checking the variable and skipping (D4), and named only the two `TestRunSelfCheck_*` E2E tests. `TestSC005_AnalysisPerformance` runs the same full pipeline on `./...` without the check, so a non-short local `go test ./...` recurses without bound.

## Goals / Non-Goals

### Goals
- Make unbounded recursion impossible, not merely avoided by test convention.
- Keep nested runs cheap: tests that generate real coverage skip under the variable.
- Leave the `--coverprofile` path and all non-nested behavior unchanged.

### Non-Goals
- Fixing `TestSC002_GazeCRAPloadMatchBetweenCrapAndReport` and `TestRunReport_CoverProfile_UnparseableContent`, which already fail on `main` without `-short`. They are unrelated to recursion.
- Adding a depth counter or allowing N levels of nesting.

## Decisions

### D1: Refuse nested generation in `generateCoverProfile`

**Decision**: When `GAZE_COVERAGE_RUN` is non-empty, `generateCoverProfile` returns `ErrNestedCoverageRun` immediately: no temp profile, no subprocess.

**Rationale**: The variable means "this process is already inside a gaze coverage collection". Generating again from there can only re-run the requesting test. For gaze's own suite that is infinite, and for a user module it is at best duplicate work. Enforcing it at the single place that spawns `go test` covers tests that forget the skip, future tests, and user code that calls gaze from its tests. The check is `!= ""`, consistent with the existing skip guards.

**Alternatives considered**:
- *Only add the skip to SC005*: fixes today's instance and relies on every future author remembering it. That is exactly the gap that caused this bug.
- *Depth counter (allow N levels)*: no legitimate use for nested generation has been identified, and a counter adds state to carry through the environment.

### D2: Skip self-analyzing tests under the variable

**Decision**: Add the existing `GAZE_COVERAGE_RUN` skip to `TestSC005_AnalysisPerformance`, `TestRunCRAPStep_RealPackage` and `TestRunProductionPipeline_RealPackage`.

**Rationale**: With D1 alone, these tests would fail inside the nested run (D1 returns an error). That marks the nested `go test` as failed and pushes coverage through partial-profile recovery. Skipping keeps nested runs clean and fast, matching the self-check tests.

### D3: Pre-generated profiles still work when nested

**Decision**: The guard sits in `generateCoverProfile`, not in `Coverage`, so `Coverage(..., coverProfile)` with a supplied profile works regardless of the variable.

**Rationale**: Parsing an existing profile spawns nothing and cannot recurse. It is also the escape hatch the error message recommends.

## Coverage Strategy

- **Unit (internal)**: `generateCoverProfile` under `GAZE_COVERAGE_RUN=1` returns `ErrNestedCoverageRun`, returns an empty path, and creates no file in `TMPDIR`. `PATH` is cleared so an attempted spawn would show as an exec error.
- **Unit (provider)**: `Coverage` with an empty profile under the variable returns `ErrNestedCoverageRun` (`errors.Is`). `Coverage` with a pre-generated profile under the variable still returns results.
- **End-to-end (manual)**: a full non-short `go test ./...` completes, with at most one level of nested `go test -coverprofile`.

## Risks / Trade-offs

### R1: A user relying on nested generation now gets an error

If a user's test suite invokes gaze coverage generation on another module while gaze measures it, that inner call now errors. This is unlikely, the error explains it, and `--coverprofile` restores the behavior. Before this change the same setup on the same module would recurse forever.
