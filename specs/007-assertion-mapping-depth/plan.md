# Implementation Plan: Assertion Mapping Depth

**Branch**: `007-assertion-mapping-depth` | **Date**: 2026-02-27 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `specs/007-assertion-mapping-depth/spec.md`

## Summary

Improve the assertion-to-side-effect mapping engine in
`internal/quality/mapping.go` to recognize indirect expression patterns
— selector access (`result.Field`), built-in wrapping (`len(docs)`),
and index access (`results[0]`) — as assertions on traced return
values. Currently the mapper only matches when a bare identifier in an
assertion expression has a direct `types.Object` identity match against
the `objToEffectID` map. This causes the `analysis` package (108
test-target pairs, 51% of all pairs) to score 6.9% contract coverage
despite 87.7% line coverage, because every test asserts on struct
fields of the return value rather than the variable itself. The fix is
a single new function `resolveExprRoot` that recursively unwinds
expression nodes to reach the root identifier, plus a two-pass
matching strategy in `matchAssertionToEffect` that tries direct
identity first, then indirect root resolution at lower confidence (65
vs 75). Additionally, when the target call is inside a helper function (e.g.,
`result := analyzeFunc(t, ...)` where `analyzeFunc` calls the target),
`traceReturnValues` falls back to helper return tracing — searching
the test function's AST for assignments whose RHS calls a helper that
transitively invokes the target. All changes are confined to
`internal/quality/mapping.go` and its tests. No changes to the
analysis, classification, or coverage computation pipeline.

## Technical Context

**Language/Version**: Go 1.24+
**Primary Dependencies**:
  - `go/ast` — `SelectorExpr`, `IndexExpr`, `CallExpr` node types
  - `go/types` — `types.Object` identity, `types.Info.Uses`/`Defs`,
    `types.Builtin` for built-in function detection
  - `github.com/unbound-force/gaze/internal/taxonomy` — `AssertionMapping`
    struct (Confidence field)
  - `github.com/unbound-force/gaze/internal/quality` — mapping engine
**Storage**: N/A — stateless analysis
**Testing**: Standard library `testing` only, `-race -count=1`
**Target Platform**: Any platform with Go 1.24+ toolchain
**Project Type**: Single binary CLI
**Performance Goals**: Per-pair mapping time within 2x of pre-change
  baseline; `resolveExprRoot` is O(depth) where depth is the
  expression nesting level (typically <= 5)
**Constraints**: No changes to `traceMutations`,
`ComputeContractCoverage`, or the `Assess` pipeline.
`traceReturnValues` is modified only to add a fallback path for
helper return tracing (US5) — the existing direct-call tracing
logic is unchanged. Direct identity matches must remain at
confidence 75. Additive change only — existing mapped assertions
must not regress.
**Scale/Scope**: Changes affect `matchAssertionToEffect` (two-pass
strategy), `traceReturnValues` (helper return fallback), and
`traceTargetValues` (parameter plumbing for helper verification) in
`mapping.go`. The ratchet test baseline floor is raised. Two new
test fixtures are added: `indirectmatch/` for expression resolution
patterns and `helperreturn/` for helper return tracing.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Rationale |
|-----------|--------|-----------|
| **I. Accuracy** | PASS | This change increases mapping accuracy from ~73.8% to >= 85% (FR-011). No change to side effect detection or classification. The `resolveExprRoot` function only matches when the root identifier's `types.Object` is present in the `objToEffectID` map — the same identity check used for direct matches. False positives are prevented by checking against the authoritative traced object map. Regressions are prevented by preserving the direct match path at confidence 75 (FR-012). |
| **II. Minimal Assumptions** | PASS | No annotations, restructuring, or convention changes required in user test code. The improvement works with existing Go test patterns — selector access, built-in calls, and indexing are standard language features, not framework-specific conventions. The analysis does not assume a particular testing library or coding style. |
| **III. Actionable Output** | PASS | By correctly mapping previously-unmapped assertions, contract coverage reports more accurately reflect test quality. Tests that were incorrectly reported as having 0% coverage (because they asserted on `result.Field` instead of `result` directly) now show the correct coverage percentage. This makes gap reports actionable — remaining gaps are genuine missing assertions, not mapping engine limitations. SC-001 (>= 80% weighted average) directly improves the actionability of `gaze quality` output. |

**GATE RESULT: PASS** — All three principles satisfied.

**Post-design re-check**: PASS — Research phase revealed the need for
helper return tracing (US5) to reach the 80% contract coverage target.
Helper tracing uses SSA call graph verification (Principle I: no false
positives), works with existing test patterns (Principle II: no
restructuring), and enables accurate gap reporting (Principle III:
actionable output). No principle violations introduced.

## Project Structure

### Documentation (this feature)

```text
specs/007-assertion-mapping-depth/
├── spec.md              # Feature specification
├── plan.md              # This file
├── research.md          # Phase 0: design research and decisions
├── data-model.md        # Phase 1: expression resolution model
├── quickstart.md        # Phase 1: how to verify the improvement
├── checklists/
│   └── requirements.md  # Spec quality checklist
└── tasks.md             # Task breakdown (created by /speckit.tasks)
```

### Source Code (changes only)

```text
internal/
└── quality/
    ├── mapping.go            # resolveExprRoot() NEW function;
    │                         # matchAssertionToEffect() two-pass strategy
    │                         # with direct (75) and indirect (65) confidence;
    │                         # traceReturnValues() helper return fallback
    ├── quality_test.go       # TestSC003_MappingAccuracy baseline raised;
    │                         # new tests for selector, builtin, index,
    │                         # and helper return tracing patterns
    └── testdata/src/
        └── (existing fixtures sufficient; may add indirectmatch/ fixture)
```

**Structure Decision**: Single project layout. All changes are within
the existing `internal/quality` package. No new packages. The
`resolveExprRoot` function is co-located in `mapping.go` because it is
called exclusively from `matchAssertionToEffect` and shares the same
`go/ast` + `go/types` dependencies.

## Key Design Decisions

### 1. Expression Root Resolution via `resolveExprRoot`

A new unexported function `resolveExprRoot` recursively unwinds
expression nodes to find the innermost `*ast.Ident`:

- `*ast.SelectorExpr` (`x.Field`) -> recurse on `.X`
- `*ast.IndexExpr` (`x[i]`) -> recurse on `.X`
- `*ast.CallExpr` (`len(x)`) -> if `Fun` resolves to `*types.Builtin`
  with name in `{len, cap}` and `len(Args) == 1`, recurse on `Args[0]`
- `*ast.Ident` -> return (base case)
- All other node types -> return `nil` (no match)

**Why a dedicated function**: Separating root resolution from the
matching logic keeps `matchAssertionToEffect` readable, enables unit
testing of the resolution algorithm in isolation, and makes future
expression pattern additions (e.g., type assertion unwinding) a single
function change.

**Why recursive**: Expression chains like `results[0].Field.SubField`
require multiple unwinding steps: `SelectorExpr` -> `SelectorExpr` ->
`IndexExpr` -> `Ident`. Recursion handles arbitrary depth naturally.
Stack depth is bounded by Go source expression nesting, which is
typically <= 5 levels.

### 2. Two-Pass Matching Strategy

`matchAssertionToEffect` uses two passes:

1. **Pass 1 (direct)**: Walk the expression tree with `ast.Inspect`
   looking for `*ast.Ident` nodes whose `types.Object` is directly in
   `objToEffectID`. This is the existing behavior. Matches produce
   confidence 75.

2. **Pass 2 (indirect)**: If Pass 1 found no match, walk the
   expression tree again. For each non-`Ident` node (`SelectorExpr`,
   `IndexExpr`, `CallExpr`), call `resolveExprRoot` to unwind to the
   root identifier. If the root's `types.Object` is in `objToEffectID`,
   produce a match at confidence 65.

**Why two passes**: This ensures existing direct matches are never
degraded. A direct identity match is always preferred over an indirect
root resolution. The two-pass approach also prevents a scenario where
a selector expression's `.Sel` field identifier happens to match a
traced object (e.g., a field named `err` matching a traced `err`
variable) — Pass 1 handles this correctly through the existing
`ast.Inspect` walk, and Pass 2 only fires if Pass 1 found nothing.

**Alternative rejected — single-pass with priority**: A single pass
that checks both direct and indirect matches would require tracking
the "best" match and its confidence, adding complexity. The two-pass
approach is simpler and has no measurable performance cost since
expression trees are small.

### 3. Confidence Values: 75 (direct) vs 65 (indirect)

- **75**: Existing value for SSA-traced direct identity matches. The
  assertion references the exact variable assigned from the target
  call. High confidence because the `types.Object` identity is exact.
- **65**: New value for indirect matches. The assertion references a
  derived expression (field access, indexing, built-in wrapping) on
  a traced variable. The match is correct but the indirection means
  the assertion could theoretically be on an unrelated property of the
  return value (e.g., `result.InternalField` asserting on an
  implementation detail rather than the return contract). A 10-point
  reduction signals this slight uncertainty to downstream consumers.

**Why not 50 or 70**: 50 would be too low — indirect matches are
still valid SSA-traced mappings, not heuristic guesses. 70 would be
too close to direct matches, making the distinction meaningless for
threshold-based filtering. 65 provides clear separation while
remaining above the 50% "uncertain" threshold.

### 4. Built-in Allowlist: `len` and `cap` Only

Only `len` and `cap` are unwound because:
- They are **value-inspecting** — they read a property of their
  argument without modifying it
- They are the only built-ins commonly used in assertion expressions
  (e.g., `if len(got) != 5`)
- Side-effecting built-ins (`append`, `delete`, `close`, `copy`,
  `make`, `new`, `clear`) would produce semantically wrong mappings —
  calling `append(got, item)` is not an assertion on `got`

Detection uses `types.Info.Uses` to resolve the `Fun` identifier to a
`*types.Builtin` object, then checks the `Name()`. This avoids false
matches on user-defined functions named `len`.

### 5. No Changes to TraceMutations or the Object Map Strategy

The `objToEffectID` map built by `traceReturnValues` and
`traceMutations` is already correct — it maps `types.Object` (the
variable `result` in `result := target()`) to effect IDs. The problem
is solely in `matchAssertionToEffect` which cannot see through
expression wrappers to reach these objects. Modifying the trace
functions to also register field objects would be incorrect — a field
like `result.SideEffects` has a different `types.Object` (the struct
field declaration) than `result` (the local variable).

Note: `traceReturnValues` IS modified by Design Decision 6 (Helper
Return Value Tracing) to add a fallback path when `findAssignLHS`
fails. The existing direct-call tracing logic and the `objToEffectID`
map-building strategy remain unchanged. `traceMutations` is not
modified.

### 6. Helper Return Value Tracing as Fallback

When `findAssignLHS` fails to find a direct assignment for the target
call (because the target is called inside a helper), the mapper
searches the test function's AST for assignments whose RHS is a call
to any function that, according to the SSA call graph, transitively
invokes the target. The LHS variable of that assignment is then added
to `objToEffectID` at indirect confidence.

**Why fallback-only**: Direct target call tracing is more precise
(the assignment is known to receive the target's return values). Helper
tracing is inherently less certain because the helper may transform,
wrap, or partially return the target's result. Limiting it to a
fallback ensures direct tracing is always preferred.

**Why depth-1 only**: The helper must directly call the target
function (verified via SSA). Transitive chains (helper A calls helper
B which calls target) are excluded to prevent false positive traces
through long call chains. Depth-1 covers the dominant pattern in the
`analysis` package (`analyzeFunc` -> `AnalyzeFunctionWithSSA`).

**Alternative rejected — full transitive tracing**: Would require
walking the complete SSA call graph from each helper, increasing
complexity and false positive risk. The `helpers` test fixture shows
that depth-1 is sufficient for real-world patterns.

## Data Pipeline

The existing pipeline is unchanged:

```
loader.Load -> analysis.Analyze -> classify.Classify -> quality.Assess -> report
```

Internal changes within `quality.Assess`:

```
MapAssertionsToEffects
  traceTargetValues        <- MODIFIED: helper return fallback
    traceReturnValues      <- MODIFIED: fallback to helper tracing
                              when findAssignLHS fails
  matchAssertionToEffect   <- MODIFIED: two-pass strategy with
                              resolveExprRoot for indirect matches
ComputeContractCoverage    <- UNCHANGED (consumes mappings)
ComputeOverSpecification   <- UNCHANGED (consumes mappings)
```

The only observable difference: more assertions map successfully,
causing `ComputeContractCoverage` to report higher coverage
percentages and fewer gaps.

## Requirement Mapping

| FR | Requirement | Component | Status |
|----|-------------|-----------|--------|
| FR-001 | Resolve selector expressions to root ident | `quality/mapping.go` | Planned |
| FR-002 | Handle arbitrarily deep selector chains | `quality/mapping.go` | Planned |
| FR-003 | Resolve built-in calls (len, cap) | `quality/mapping.go` | Planned |
| FR-004 | Limit to value-inspecting built-ins only | `quality/mapping.go` | Planned |
| FR-005 | Resolve index expressions | `quality/mapping.go` | Planned |
| FR-006 | Combine index + selector resolution | `quality/mapping.go` | Planned |
| FR-007 | Indirect matches at lower confidence | `quality/mapping.go` | Planned |
| FR-008 | Direct=75, Indirect=65 | `quality/mapping.go` | Planned |
| FR-009 | No false positive matches | `quality/mapping.go` | Planned |
| FR-010 | Resolve traced args in function calls | `quality/mapping.go` | Planned |
| FR-011 | Raise ratchet floor to >= 85% | `quality/quality_test.go` | Planned |
| FR-012 | No regressions in existing mappings | `quality/quality_test.go` | Planned |
| FR-013 | Helper return value fallback tracing | `quality/mapping.go` | Planned |
| FR-014 | Verify helper calls target via SSA | `quality/mapping.go` | Planned |
| FR-015 | Helper tracing limited to depth 1 | `quality/mapping.go` | Planned |

## Expression Resolution Examples

| Expression | AST Type | Resolution Chain | Root | Confidence |
|------------|----------|------------------|------|------------|
| `got` | `Ident` | (direct) | `got` | 75 |
| `err` | `Ident` | (direct) | `err` | 75 |
| `result.SideEffects` | `SelectorExpr` | `.X` -> `result` | `result` | 65 |
| `result.A.B.C` | `SelectorExpr`x3 | `.X` -> `.X` -> `.X` -> `result` | `result` | 65 |
| `len(docs)` | `CallExpr(builtin)` | `Args[0]` -> `docs` | `docs` | 65 |
| `results[0]` | `IndexExpr` | `.X` -> `results` | `results` | 65 |
| `results[0].Field` | `SelectorExpr>IndexExpr` | `.X` -> `.X` -> `results` | `results` | 65 |
| `countEffects(result.SideEffects, ...)` | `CallExpr(user)` | `Args[0]` -> `.X` -> `result` | `result` | 65 |
| `user.Name` (testify arg) | `SelectorExpr` | `.X` -> `user` | `user` | 65 |
| `append(got, item)` | `CallExpr(builtin)` | NOT unwound (side-effecting) | — | — |
| `localVar.Field` | `SelectorExpr` | `.X` -> `localVar` | `localVar` (not in map) | — |
