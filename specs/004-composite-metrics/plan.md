# Implementation Plan: Composite Quality Metrics

**Feature Branch**: `004-composite-metrics`  
**Spec**: `specs/004-composite-metrics/spec.md`  
**Status**: Retroactive — documents what was built prior to establishing the
Speckit pipeline. Implementation of US1 (classic CRAP), US2 scaffolding, and
US5 (CI enforcement) landed directly on `main` via commits `4b37981`,
`5ea4085`, `58b2da5`, and `2d67f76` before `plan.md` and `tasks.md` existed.

---

## Constitution Check

### I. Accuracy — PASS

CRAP scores are computed with the exact formula from the spec
(`comp^2 * (1 - cov/100)^3 + comp`). Cyclomatic complexity is delegated to
`github.com/fzipp/gocyclo`, the Go community standard. Per-function coverage
is parsed from `go test -coverprofile` output via `golang.org/x/tools/cover`,
matching `go tool cover -func` semantics. Formula accuracy is validated by
unit tests against hand-computed values for 7 known (complexity, coverage)
pairs. False negatives are prevented by defaulting to 0% coverage for
functions absent from the profile.

### II. Minimal Assumptions — PASS

The `gaze crap` command requires no source annotations. It accepts any Go
package pattern (`./...`, `./internal/...`, etc.). Coverage profile generation
falls back to auto-running `go test -coverprofile` only when no `--coverprofile`
flag is provided, so users with pre-existing CI profiles can pass them directly.
Generated files (detected via Go convention `// Code generated ... DO NOT EDIT.`)
and test files (`_test.go`) are excluded without configuration.

### III. Actionable Output — PASS

Text output is sorted by CRAP score descending, marks above-threshold functions
with `*`, and includes a "Worst Offenders" top-5 section. JSON output is
machine-readable with stable per-function fields. CI enforcement provides
non-zero exit codes with a one-line summary (e.g., `CRAPload: 3/5 (FAIL)`).
All functions are identified by package, name, file, and line number.

---

## Architecture

### Package Structure

```
internal/crap/
  crap.go       — Core types (Score, Quadrant, Summary, Report) and
                  Formula() + ClassifyQuadrant() pure functions
  analyze.go    — Analyze() entry point, Options/DefaultOptions(),
                  gocyclo integration, coverage join, buildSummary()
  coverage.go   — ParseCoverProfile(), findFunctions(), funcCoverage(),
                  resolveFilePath(), readModulePath()
  report.go     — WriteJSON(), WriteText() with lipgloss styling
  crap_test.go  — 48+ unit tests covering all exported and key unexported
                  functions (includes 21-case formula benchmark suite)
  bench_test.go — 5 benchmarks (Formula, ClassifyQuadrant, buildSummary,
                  buildCoverMap, isGeneratedFile)

cmd/gaze/main.go  — newCrapCmd(), runCrap(), writeCrapReport(),
                    printCISummary(), checkCIThresholds()
```

### Key Design Decisions

**1. gocyclo for complexity**

`github.com/fzipp/gocyclo` is the community standard for Go cyclomatic
complexity. It walks AST `FuncDecl` nodes and counts branching constructs
(`if`, `for`, `switch`, `case`, `&&`, `||`, `select`, goroutine sends).
Complexity is 1-indexed (minimum 1 for any function). SC-007 requires
complexity to match `gocyclo` output — achieved by delegating directly.

**2. `golang.org/x/tools/cover` for coverage parsing**

Rather than parsing the coverage profile format manually, the implementation
uses `cover.ParseProfiles()` from the official Go tools module. Per-function
coverage is then computed by `funcCoverage()`: it walks the profile's statement
blocks and accumulates covered/total statement counts within each function's
line/column extent. This matches `go tool cover -func` semantics (SC-008).

**3. Two-stage coverage join (exact + basename fallback)**

Coverage profiles use import-path-relative filenames
(e.g., `github.com/unbound-force/gaze/internal/crap/crap.go`) while gocyclo uses
absolute filesystem paths. `buildCoverMap()` constructs two lookup maps:

- `exact`: keyed by `(absolutePath, startLine)`
- `basename`: keyed by `(filename.go, startLine)` as fallback

This handles path mismatches (CI vs. local, symlinks) without configuration.

**4. Nullable fields for GazeCRAP stub wire (FR-004, FR-015)**

`Score.GazeCRAP`, `Score.ContractCoverage`, and `Score.Quadrant` are declared
as pointer types with `json:"...,omitempty"` tags. They remain `nil` in all
current `Analyze()` outputs because contract coverage (Specs 002-003) is not
yet implemented. When Spec 003 is complete, the activation path is already
wired: `buildSummary()` gates `GazeCRAPload`, `GazeCRAPThreshold`, and
`QuadrantCounts` on the presence of non-nil `GazeCRAP` fields. No structural
changes to `Score` or `Summary` will be needed — only population of the fields.

**5. Temporary coverage profile**

When auto-generating the coverage profile, `generateCoverProfile()` writes to
a `os.CreateTemp("", "gaze-cover-*.out")` path and `defer os.Remove()`s it.
This avoids clobbering any existing `cover.out` in the user's working directory
and leaves no artifacts on disk.

**6. isGeneratedFile scans only until `package` clause**

Generated-file detection reads the file line by line and stops at the first
`package` line, matching the Go specification for generated file headers. A
`map[string]bool` cache in `Analyze()` prevents re-reading the same file for
multiple functions declared within it.

**7. Testable CLI pattern**

`newCrapCmd()` delegates all logic to `runCrap(crapParams)`. `crapParams`
includes `io.Writer` fields for stdout and stderr, enabling unit tests without
subprocess execution. This mirrors the pattern established in Specs 001-002.

---

## Dependency Graph

```
                        ┌───────────────────────────┐
                        │ github.com/fzipp/gocyclo   │
                        │ golang.org/x/tools/cover   │
                        └─────────────┬─────────────┘
                                      │
                                      ▼
                        ┌───────────────────────────┐
                        │ US1: Classic CRAP          │
                        │ internal/crap/             │
                        │ cmd/gaze: gaze crap        │
                        │ Status: COMPLETE           │
                        └─────────────┬─────────────┘
                                      │
              ┌───────────────────────┤
              │                       │
              ▼                       ▼
  ┌─────────────────┐    ┌────────────────────────┐
  │ Spec 001        │    │ US2: GazeCRAP          │
  │ Spec 002        │───▶│ Wire: complete         │
  │ Spec 003        │    │ Activation: pending    │
  │ (not yet built) │    │ Spec 003               │
  └─────────────────┘    └───────────┬────────────┘
                                     │
                                     ▼
                         ┌────────────────────────┐
                         │ US3: Quadrant report   │
                         │ Logic: complete        │
                         │ Activation: pending    │
                         │ GazeCRAP               │
                         └───────────┬────────────┘
                                     │
                             ┌───────┴───────┐
                             ▼               ▼
                    ┌─────────────┐  ┌─────────────┐
                    │ US4:        │  │ US5: CI     │
                    │ self-check  │  │ enforcement │
                    │ NOT BUILT   │  │ COMPLETE    │
                    └─────────────┘  └─────────────┘
```

---

## Scope Delivered vs. Deferred

### Delivered (US1 + US3 logic + US5)

| FR | Requirement | Implementation |
|----|-------------|----------------|
| FR-001 | Cyclomatic complexity via gocyclo | `analyze.go`: `gocyclo.Analyze()` |
| FR-002 | Parse `go test -coverprofile` | `coverage.go`: `ParseCoverProfile()` |
| FR-003 | Classic CRAP formula | `crap.go`: `Formula()` |
| FR-005 | CRAPload count | `analyze.go`: `buildSummary()` |
| FR-006 | Default threshold=15, configurable | `analyze.go`: `DefaultOptions()` + CLI flags |
| FR-007 | Quadrant classification logic | `crap.go`: `ClassifyQuadrant()` + 4 constants |
| FR-008 | JSON + text output | `report.go`: `WriteJSON()` + `WriteText()` |
| FR-009 | JSON per-function fields | `crap.go`: `Score` struct with JSON tags |
| FR-011 | `--max-crapload` / `--max-gaze-crapload` | `cmd/gaze/main.go`: `checkCIThresholds()` |
| FR-012 | Auto-generate or accept `--coverprofile` | `analyze.go`: `generateCoverProfile()` |
| FR-013 | Exclude `_test.go` + generated files | `analyze.go`: `isGeneratedFile()` |
| FR-014 | Join by file path + line number | `analyze.go`: `buildCoverMap()` + `lookupCoverage()` |

### Stub-wired, awaiting Spec 003 (US2)

| FR | Requirement | Status |
|----|-------------|--------|
| FR-004 | GazeCRAP formula with contract coverage | Fields declared (`*float64`, `*Quadrant`); formula not yet invoked; activation gated on Spec 003 |
| FR-005 (partial) | GazeCRAPload | `buildSummary()` populates only when `GazeCRAP != nil` |
| — | Summary aggregate fields | `AvgGazeCRAP *float64`, `AvgContractCoverage *float64`, `WorstGazeCRAP []Score` declared in `Summary` struct as nullable/omitempty stubs; population logic in `buildSummary()` activates when `hasGazeCRAP` is true |

### Not yet implemented (US4)

| FR | Requirement | Status |
|----|-------------|--------|
| FR-010 | `gaze self-check` command | Not implemented; no `self-check` cobra command registered |
| FR-015 | Stderr warning when GazeCRAP unavailable | COMPLETE — `runCrap()` in `cmd/gaze/main.go` emits "note: GazeCRAP unavailable" to stderr when `GazeCRAPload == nil` |

---

## Success Criteria Mapping

| SC | Criterion | Status |
|----|-----------|--------|
| SC-001 | CRAP formula accuracy ±0.01 for 20+ functions | PASS — 7 individual tests + 21-case table-driven `TestFormula_BenchmarkSuite` = 28 total hand-computed (comp, cov) pairs |
| SC-002 | GazeCRAP uses contract coverage correctly | NOT TESTABLE — pending Spec 003 |
| SC-003 | CRAPload count matches threshold | PASS — `TestBuildSummary_CRAPload` |
| SC-004 | Quadrant classification for all 4 quadrants | PASS — 6 quadrant unit tests |
| SC-005 | `gaze self-check` completes successfully | NOT IMPLEMENTED |
| SC-006 | CI threshold enforcement exits non-zero | PASS — 6 `checkCIThresholds` tests + 6 `printCISummary` tests |
| SC-007 | Complexity matches gocyclo output | PASS — delegates directly to `gocyclo.Analyze()` |
| SC-008 | Coverage matches `go tool cover -func` | PASS — uses same `cover.ParseProfiles()` + statement-block counting |

---

## Test Coverage Summary

**`internal/crap/crap_test.go`** — 48+ tests, package `crap`

- `Formula`: 7 individual tests + 21-case table-driven `TestFormula_BenchmarkSuite` (SC-001: 28 total hand-computed pairs)
- `ClassifyQuadrant`: 6 tests (Q1–Q4, at-threshold boundary, independent thresholds)
- `buildSummary`: 3 tests (CRAPload counting, worst-5 ordering, empty input)
- `WriteJSON`: 1 test (valid JSON output)
- `WriteText`: 2 tests (summary sections present, threshold marker `*`)
- `isGeneratedFile`: 4 tests (generated, not generated, after-package-clause, nonexistent)
- `resolvePatterns`: 4 tests (`./...`, `./sub`, bare, multiple)
- `buildCoverMap`: 2 tests (basic entries, empty input)
- `lookupCoverage`: 3 tests (exact match, basename fallback, no match → 0)
- `recvTypeString` / `findFunctions`: 3 tests (pointer/value recv, no-body skipped, invalid file)
- `funcCoverage`: 2 tests (overlapping blocks, empty profile)
- `resolveFilePath`: 3 tests (absolute path, module-relative, no go.mod)
- `readModulePath`: 2 tests (valid go.mod, missing go.mod)
- `shortenPath`: 5 tests (/internal/, /cmd/, /pkg/, long-path fallback, short path)

**`internal/crap/bench_test.go`** — 5 benchmarks

- `BenchmarkFormula`, `BenchmarkClassifyQuadrant`, `BenchmarkBuildSummary`,
  `BenchmarkBuildCoverMap`, `BenchmarkIsGeneratedFile_NotGenerated`

**`cmd/gaze/main_test.go`** — CRAP CLI tests

- `writeCrapReport`: 2 tests
- `printCISummary`: 6 tests
- `checkCIThresholds`: 6 tests
- `runCrap` format validation: 1 test
