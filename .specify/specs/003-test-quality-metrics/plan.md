# Implementation Plan: Test Quality Metrics & Reporting

**Branch**: `003-test-quality-metrics` | **Date**: 2026-02-22 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `specs/003-test-quality-metrics/spec.md`
**Clarifications**: [clarify.md](clarify.md)

## Summary

Build a test quality metrics engine that computes Contract Coverage
(ratio of contractual side effects asserted on) and
Over-Specification Score (count and ratio of incidental side effects
asserted on) for Go test functions. Uses SSA data flow analysis to
map test assertions to specific side effects detected by Spec 001
and classified by Spec 002. Call graph inference determines which
function a test is exercising. Delivered as a new `gaze quality`
CLI subcommand producing JSON and human-readable output with CI
threshold enforcement.

## Technical Context

**Language/Version**: Go 1.24+
**Primary Dependencies**:
  - `golang.org/x/tools/go/packages` (package loading)
  - `golang.org/x/tools/go/ssa` + `ssautil` (SSA for test functions)
  - `golang.org/x/tools/go/callgraph` (test-to-target pairing)
  - `github.com/spf13/cobra` (CLI framework)
  - Existing: `internal/analysis` (Spec 001), `internal/classify`
    (Spec 002), `internal/taxonomy` (domain types)
**Storage**: N/A (stateless analysis, output to stdout)
**Testing**: Standard library `testing` package only, `-race -count=1`
**Target Platform**: Any platform with Go toolchain
**Project Type**: Single binary CLI
**Performance Goals**: < 2s for a single test-target pair, < 30s for
  a package with 50 test functions (SC-004 determinism)
**Constraints**: Static analysis only (no test execution). SSA data
  flow for assertion mapping. Helper traversal bounded to 3 levels.
**Scale/Scope**: Typical Go modules with < 500 packages

## Constitution Check

*GATE: Must pass before implementation. Re-check after design.*

| Principle | Status | Rationale |
|-----------|--------|-----------|
| **I. Accuracy** | PASS | SSA data flow provides high-accuracy assertion mapping. SC-003 requires >= 90% for standard patterns. Unmapped assertions reported separately (never silently dropped). Confidence per assertion. Benchmark suite of 20+ test-target pairs (SC-001). |
| **II. Minimal Assumptions** | PASS | No annotation or restructuring required. Call graph inference finds the target automatically. Unsupported assertion libraries produce "unrecognized" warnings with detection confidence. No assumptions about test framework beyond supported patterns. |
| **III. Actionable Output** | PASS | Gaps identify specific unasserted contractual effects with type, description, and location (FR-004). Over-specified assertions include actionable suggestions (FR-002). Summary line format specified (FR-010). JSON and text output. |

**GATE RESULT: PASS** — Proceed to implementation.

## Project Structure

### Documentation (this feature)

```text
.specify/specs/003-test-quality-metrics/
├── spec.md              # Feature specification (complete)
├── clarify.md           # Resolved ambiguities and decisions
├── plan.md              # This file
└── tasks.md             # Task breakdown (next step)
```

### Source Code (repository root)

```text
cmd/gaze/
├── main.go              # Add `quality` subcommand, threshold flags
└── main_test.go         # Tests for quality CLI integration

internal/
├── analysis/            # Existing — no changes needed
├── taxonomy/
│   └── types.go         # Add AssertionMapping, QualityReport types
├── classify/            # Existing — no changes needed
├── quality/             # NEW — test quality metrics engine
│   ├── quality.go       # Assess() entry point, Options struct
│   ├── pairing.go       # Test-to-target pairing via call graph
│   ├── assertion.go     # Assertion site detection (AST patterns)
│   ├── mapping.go       # SSA data flow assertion-to-effect mapping
│   ├── coverage.go      # Contract Coverage computation
│   ├── overspec.go      # Over-Specification Score computation
│   ├── report.go        # WriteJSON(), WriteText() formatters
│   ├── quality_test.go  # Unit tests
│   ├── bench_test.go    # Benchmark tests
│   └── testdata/src/    # Test fixture packages
│       ├── welltested/  # Functions with full contract coverage
│       ├── undertested/ # Functions with gaps in coverage
│       ├── overspecd/   # Tests asserting on incidental effects
│       ├── tabledriven/ # Table-driven test patterns
│       ├── helpers/     # Test helper attribution patterns
│       └── multilib/    # testify and go-cmp assertion patterns
└── report/              # Existing
    └── schema.go        # Extend JSON Schema (additive, FR-009)
```

## Key Design Decisions

### 1. Call graph inference for test-target pairing

Rather than relying on naming convention (`TestXxx` → `Xxx`), Gaze
analyzes the test function's call graph to identify the primary
function under test. This satisfies Constitution Principle II
(Minimal Assumptions) — it works with any test naming convention.

Algorithm:
1. Build SSA for the test package
2. For each `Test*` function, walk its call graph
3. Identify calls to non-test, non-stdlib functions in the target
   package
4. If exactly one target function → automatic pairing
5. If multiple targets → per-target metrics with warning
6. If zero targets → skip with warning

The optional `--target` flag allows explicit disambiguation.

### 2. Full SSA data flow for assertion mapping

The assertion-to-side-effect mapping uses SSA data flow analysis to
trace values from the target function call through the test function's
SSA graph to assertion sites.

Pipeline:
1. Identify the target call instruction in the test SSA
2. Extract return values (via `ssa.Extract` for multi-return) and
   track them through SSA edges (phi, store, load, field access)
3. For mutation side effects, identify the value passed as
   receiver/pointer arg and track reads after the call
4. When a tracked value reaches a recognized assertion site, create
   the `AssertionMapping`
5. Values that don't reach assertions → gaps (unasserted contractual
   effects)
6. Assertions on values not traced from target → unmapped

### 3. Two-level assertion detection

**Level 1 — Detection** (AST-based): Find assertion sites in test
code by matching syntactic patterns for stdlib, testify, and go-cmp.
This is fast and produces a list of assertion locations.

**Level 2 — Mapping** (SSA-based): Link each detected assertion to
a specific side effect by tracing SSA data flow. This is the
expensive part and produces `AssertionMapping` instances.

### 4. Helper traversal bounded to 3 levels

Test helper functions (those accepting `*testing.T` or `*testing.B`)
are traversed up to 3 levels deep. Assertions in helpers beyond
depth 3 are reported as unmapped with a warning.

`t.Run` sub-test closures are traversed at depth 0 (they are
inlined into the parent's analysis). The union of assertions across
all sub-tests determines the test's aggregate coverage.

### 5. Asymmetric analysis scope

Side effect detection (Spec 001) is non-transitive — only the target
function's direct body. Assertion detection (Spec 003) IS transitive
within the test function's call graph (helpers, sub-tests, up to 3
levels). This is by design: the target's side effects are a fixed
set, but the test may exercise them through various helper patterns.

## Data Pipeline

```
loader.Load(pattern)
  → analysis.Analyze(pkg, opts)         [Spec 001: detect side effects]
    → classify.Classify(results, opts)  [Spec 002: label contractual/incidental]
      → quality.Assess(results, testPkg, opts)  [Spec 003: compute metrics]
        → quality.Report (JSON / text / CI exit code)
```

The `quality.Assess()` function is the entry point. It:
1. Loads the test package for the target
2. Pairs test functions to target functions (call graph inference)
3. Detects assertion sites in each test function (AST)
4. Maps assertions to side effects (SSA data flow)
5. Computes Contract Coverage and Over-Specification per pair
6. Aggregates to package-level summary

## Integration with Spec 004 (CRAP)

Spec 003 exposes a `quality.ContractCoverageForFunc()` helper that
returns the contract coverage percentage for a specific function.
Spec 004's `crap.Analyze()` calls this to populate
`Score.ContractCoverage` and compute `GazeCRAP`. The CRAP integration
is Spec 004's responsibility (T050), not this spec's.

## Requirement Mapping

### Fully implemented by this plan

| FR | Requirement | Component |
|----|-------------|-----------|
| FR-001 | Contract Coverage formula | `coverage.go` |
| FR-002 | Over-Specification count + ratio | `overspec.go` |
| FR-003 | Assertion-to-side-effect mapping | `mapping.go` |
| FR-004 | Gap reporting (unasserted contractual effects) | `coverage.go` |
| FR-005 | Stable IDs for diffable results | Already implemented (`taxonomy.GenerateID`) |
| FR-006 | `--min-contract-coverage` flag | `cmd/gaze/main.go` |
| FR-007 | `--max-over-specification` flag | `cmd/gaze/main.go` |
| FR-008 | Default report-only mode | `cmd/gaze/main.go` |
| FR-009 | JSON output extends Spec 002 schema | `report.go` + `report/schema.go` |
| FR-010 | Human-readable summary line | `report.go` |
| FR-011 | Table-driven test support | `mapping.go` (t.Run traversal) |
| FR-012 | Helper function attribution | `mapping.go` (bounded traversal) |
| FR-013 | Ambiguous effects excluded from metrics | `coverage.go`, `overspec.go` |
| FR-014 | Go 1.24+ support | Consistent with existing codebase |
| FR-015 | v1 assertion patterns (stdlib, testify, go-cmp) | `assertion.go` |
| FR-016 | Package-level summary | `quality.go` |

### Test Coverage

**Test fixtures** in `internal/quality/testdata/src/`:
- `welltested/` — functions with 100% contract coverage
- `undertested/` — functions with known gaps
- `overspecd/` — tests asserting on incidental effects
- `tabledriven/` — `t.Run` sub-test patterns
- `helpers/` — test helper attribution patterns
- `multilib/` — testify and go-cmp assertion patterns

**Benchmark suite** (SC-001): 20+ test-target pairs with hand-computed
expected coverage. Each fixture has a known set of contractual effects
and a test that asserts on a known subset.

**Acceptance tests**: Named `TestSC001_*` through `TestSC007_*`
matching the success criteria.

## Phased Implementation

### Phase 1: Types and Infrastructure
Define domain types (`AssertionMapping`, `ContractCoverage`,
`OverSpecificationScore`, `QualityReport`, `PackageSummary`).
Set up the `internal/quality/` package with `Options` and stubs.

### Phase 2: Test-Target Pairing
Implement call graph inference in `pairing.go`. Build SSA for
test packages, identify target functions, handle multi-target
and no-target cases.

### Phase 3: Assertion Detection
Implement AST-based assertion site detection in `assertion.go`.
Cover stdlib, testify, and go-cmp patterns. Produce a list of
assertion sites with type and location.

### Phase 4: SSA Data Flow Mapping
Implement the core mapping engine in `mapping.go`. Trace values
from target call through SSA to assertion sites. Handle return
values, error returns, mutations. Support helper traversal (3
levels) and `t.Run` sub-tests.

### Phase 5: Metrics Computation
Implement Contract Coverage (`coverage.go`) and
Over-Specification Score (`overspec.go`). Aggregate to package
summary. Exclude ambiguous effects from both metrics.

### Phase 6: Output and CLI
Implement `WriteJSON()`, `WriteText()` in `report.go`. Extend
JSON Schema in `report/schema.go`. Add `gaze quality` subcommand
with threshold flags.

### Phase 7: Tests and Validation
Build test fixtures. Write unit tests for each component.
Write acceptance tests for SC-001 through SC-007. Benchmark
performance.
