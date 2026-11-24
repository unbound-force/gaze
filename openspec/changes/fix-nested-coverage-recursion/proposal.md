## Why

`fix-hardcoded-short-flag` replaced the blanket `-short` recursion guard with `GAZE_COVERAGE_RUN=1`, which `buildGoTestCmd` sets on the `go test -coverprofile` subprocess. Only a test that checks the variable and skips is protected, and only `TestRunSelfCheck_TextFormat` and `TestRunSelfCheck_JSONFormat` do. `TestSC005_AnalysisPerformance` (`cmd/gaze/main_test.go`) also runs the real pipeline on `./...` and has no such check.

So a plain `go test ./...` without `-short` on the gaze module recurses without bound: SC005 runs `go test -coverprofile ./...`, whose `gaze.test` runs SC005 again, and so on. CI never hits this because it runs `-short` or `-run TestRunSelfCheck`. A local run does. On 2026-11-09 an agent's local run left an orphaned chain that gained a level about once a minute; after 10 minutes it held 23 processes, ~5 CPU cores and 6.9 GB RSS, plus ~15k leftover `go-build*` temp directories.

`TestRunCRAPStep_RealPackage` and `TestRunProductionPipeline_RealPackage` (`internal/aireport/runner_steps_test.go`) also generate coverage inside the nested run. They target `internal/config`, so they terminate, but they repeat work for nothing.

## What Changes

1. **Enforce the guard where coverage is generated.** `generateCoverProfile` returns `ErrNestedCoverageRun` when `GAZE_COVERAGE_RUN` is already set, before creating a temp profile or spawning `go test`. Recursion can then never go deeper than one level, whichever test (or user code) asks for coverage.
2. **Skip the remaining self-analyzing tests under the guard.** Add the same `GAZE_COVERAGE_RUN` skip the self-check tests use to `TestSC005_AnalysisPerformance`, `TestRunCRAPStep_RealPackage` and `TestRunProductionPipeline_RealPackage`, so a nested run spends no time on them and does not fail.

## Capabilities

### New Capabilities
- `goprovider.ErrNestedCoverageRun`: sentinel error returned by line-coverage generation inside a gaze coverage collection.

### Modified Capabilities
- `generateCoverProfile`: refuses to run when `GAZE_COVERAGE_RUN` is set. The pre-generated `--coverprofile` path is unchanged.
- Self-analyzing tests in `cmd/gaze` and `internal/aireport`: skip under `GAZE_COVERAGE_RUN`.

### Removed Capabilities
- None.

## Impact

- **Files modified**: `internal/provider/goprovider/coverage.go`, `internal/provider/goprovider/coverage_test.go`, `internal/provider/goprovider/goprovider_test.go`, `cmd/gaze/main_test.go`, `internal/aireport/runner_steps_test.go`.
- **Behavioral change**: when gaze runs inside a gaze coverage collection (for example, a user's test invokes gaze's coverage generation on its own module while gaze measures that module), coverage generation now fails fast with a clear error instead of recursing. The error names `--coverprofile` as the way to supply coverage. Outside a gaze coverage collection, nothing changes.
- **CI impact**: none. CI uses `-short` and `--coverprofile=coverage.out`, and the self-check job already skips under the variable.
- **Semver**: bug fix (PATCH).

## Constitution Alignment

Assessed against the Gaze project constitution (`.specify/memory/constitution.md`).

### I. Accuracy

**Assessment**: PASS

Coverage results are unchanged for every normal run. A nested run previously produced no result at all (it never finished), so failing fast loses nothing.

### II. Minimal Assumptions

**Assessment**: PASS

Users need no new flags or annotations. The guard relies only on the variable gaze already sets on its own subprocess.

### III. Actionable Output

**Assessment**: PASS

The new error says why generation was refused and what to do instead (`--coverprofile`).

### IV. Testability

**Assessment**: PASS

The guard is a pure environment check covered by unit tests through `t.Setenv`, with no subprocess needed. `PATH` is cleared in the tests so that any spawn attempt would surface as a different error.
