# Implementation Plan: Contract Coverage Gap Remediation

**Branch**: `008-contract-coverage-gaps` | **Date**: 2026-02-27 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/008-contract-coverage-gaps/spec.md`

## Summary

Add direct unit tests with contract-level assertions to 8 functions
that have high GazeCRAP scores due to zero contract coverage (despite
having good line coverage through indirect tests). Also complete the
deferred spec 007 validation tasks (T037/T044) to establish a baseline
for before/after comparison. The implementation reuses existing test
fixtures and loading infrastructure — no new production code is
written; only test files are added or modified.

## Technical Context

**Language/Version**: Go 1.24+
**Primary Dependencies**: `go/packages`, `go/ast`, `go/types`, `golang.org/x/tools/go/ssa`
**Storage**: N/A (test-only changes)
**Testing**: Standard library `testing` package only (no testify)
**Target Platform**: darwin/linux (amd64/arm64)
**Project Type**: Single binary CLI (`cmd/gaze/`) with internal packages
**Performance Goals**: N/A (tests must pass within CI timeout)
**Constraints**: Tests must use `-race -count=1`; no external assertion libraries
**Scale/Scope**: 8 functions across 3 packages (`internal/classify/`, `internal/analysis/`, `cmd/gaze/`)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Evidence |
|-----------|--------|----------|
| **I. Accuracy** | PASS | Adds contract assertions to verify accuracy of side effect detection and classification signal functions. Directly supports "accuracy claims MUST be backed by automated regression tests." |
| **II. Minimal Assumptions** | PASS | No changes to user-facing analysis. Tests use existing fixtures and infrastructure. No new assumptions about host projects. |
| **III. Actionable Output** | PASS | US4/US5 dogfood Gaze's metrics to measure improvement. Before/after GazeCRAP comparison validates that Gaze's own output is actionable. |

No violations. No Complexity Tracking needed.

## Project Structure

### Documentation (this feature)

```text
specs/008-contract-coverage-gaps/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
└── tasks.md             # Phase 2 output (/speckit.tasks)
```

### Source Code (changes only)

```text
internal/
  classify/
    visibility_test.go        # NEW — direct tests for AnalyzeVisibilitySignal (FR-001..FR-003)
    godoc_test.go              # NEW — direct tests for AnalyzeGodocSignal (FR-004..FR-006)
    callers_test.go            # NEW — direct tests for AnalyzeCallerSignal (FR-007..FR-009)
  analysis/
    p1effects_test.go          # NEW — direct tests for AnalyzeP1Effects (FR-010..FR-011)
    p2effects_test.go          # NEW — direct tests for AnalyzeP2Effects (FR-012)
    returns_test.go            # NEW — direct tests for AnalyzeReturns (FR-013)
cmd/
  gaze/
    interactive_test.go        # NEW — tests for renderAnalyzeContent (FR-016..FR-017)
    main_test.go               # MODIFIED — strengthen buildContractCoverageFunc tests (FR-018..FR-019)
```

**Structure Decision**: Test-only changes following Go convention of
`*_test.go` files alongside source in the same directory. Group A
tests (`classify/`) use the external test package pattern
(`classify_test`). Group B tests (`analysis/`) use the external test
package pattern (`analysis_test`). Group C tests (`cmd/gaze/`) use the
internal test package pattern (`package main`) since
`renderAnalyzeContent` is unexported.

## Implementation Strategy

### Group A: Classification Signal Tests

**Approach**: Direct unit tests calling each exported signal function
with crafted inputs. These functions have simple signatures — no
complex fixture loading needed for visibility and godoc. The caller
signal requires loaded packages for cross-package analysis.

**AnalyzeVisibilitySignal** (`visibility_test.go`):
- Inputs: `*ast.FuncDecl` and `types.Object` constructed from loaded
  fixture packages. Use the existing `contracts` fixture which has
  exported functions, exported return types, and receiver methods.
- Table-driven test with cases for each dimension combination.
- Assert on `Weight` (exact int) and `Reasoning` (substring match).

**AnalyzeGodocSignal** (`godoc_test.go`):
- Inputs: `*ast.FuncDecl` (with `Doc` field set) and `effectType`.
- Can use the `contracts` fixture (has godoc-annotated functions like
  `GetVersion`, `SetPrimary`, `LoadProfile`) or construct minimal
  `*ast.FuncDecl` with `Doc` comment groups programmatically (simpler
  for keyword coverage).
- Table-driven test covering each contractual keyword × effectType
  combination, incidental keywords, and priority ordering.

**AnalyzeCallerSignal** (`callers_test.go`):
- Inputs: `types.Object` and `[]*packages.Package`.
- Use the existing `callers` fixture package (calls functions in
  `contracts`). Load all testdata packages via `loadTestPackages(t)`,
  find the target `types.Object` from `contracts`, and pass all
  packages as `modulePkgs`.
- Test weight tiers by selecting functions with known caller counts.

### Group B: Analysis Core Tests

**Approach**: Load existing test fixtures via the `loadTestdataPackage`
/ `cachedTestPackage` helpers (already available in `analysis_test.go`).
Extract `*ast.FuncDecl` via `FindFuncDecl` (test export). Extract
`pkg.Fset` and `pkg.TypesInfo` from the loaded package. Call each
analysis function directly and assert on the returned
`[]taxonomy.SideEffect`.

**Key insight**: The existing helpers (`loadTestdataPackage`,
`cachedTestPackage`, `FindFuncDecl`, `hasEffect`, `countEffects`,
`effectWithTarget`) are all in `analysis_test.go` using the
`analysis_test` package. New test files (`p1effects_test.go`, etc.)
in the same package can reuse these helpers directly.

**Test pattern** (per function × effect type):
```go
func TestAnalyzeP1Effects_Direct_GlobalMutation(t *testing.T) {
    pkg := loadTestPackageFromCache(t, "p1effects")
    fd := analysis.FindFuncDecl(pkg, "MutateGlobal")
    effects := analysis.AnalyzeP1Effects(pkg.Fset, pkg.TypesInfo, fd, pkg.PkgPath, "MutateGlobal")
    if !hasEffect(effects, taxonomy.GlobalMutation) { t.Error(...) }
    // Assert Type, Tier, Description
}
```

### Group C: CLI Layer Tests

**`renderAnalyzeContent`** (`interactive_test.go`):
- Construct `[]taxonomy.AnalysisResult` structs directly (no fixture
  loading — it's a pure rendering function).
- Assert output string contains expected function names, tier labels,
  and effect descriptions.
- Verify truncation at 50 characters.
- Use `package main` (internal test) since function is unexported.

**`buildContractCoverageFunc`** (strengthen `main_test.go`):
- Modify `TestBuildContractCoverageFunc_WelltestedPackage` to:
  - Assert `fn != nil` (currently allows nil as passing).
  - Assert `ok == true` for known function.
  - Assert `pct > 0` (not just `pct >= 0`).
- Keep `testing.Short()` guard since it runs the full pipeline.

### Baseline and Verification (US4)

- Run `gaze quality --format=json ./...` before any code changes.
- Record weighted average contract coverage.
- Run spec 007 quickstart.md validation steps.
- After all code changes, re-run and compare.

## Requirement Mapping

| Requirement | Component | Status |
|-------------|-----------|--------|
| FR-001..FR-003 | `classify/visibility_test.go` | Planned |
| FR-004..FR-006 | `classify/godoc_test.go` | Planned |
| FR-007..FR-009 | `classify/callers_test.go` | Planned |
| FR-010..FR-011 | `analysis/p1effects_test.go` | Planned |
| FR-012 | `analysis/p2effects_test.go` | Planned |
| FR-013 | `analysis/returns_test.go` | Planned |
| FR-014 | All Group B test files | Planned |
| FR-015 | All Group B test files (scope control) | Planned |
| FR-016..FR-017 | `cmd/gaze/interactive_test.go` | Planned |
| FR-018..FR-019 | `cmd/gaze/main_test.go` (modify) | Planned |
| FR-020..FR-021 | Baseline measurement (manual) | Planned |
| FR-022..FR-023 | Post-change verification (manual) | Planned |
