# Clarification: Test Quality Metrics & Reporting

**Spec**: `specs/003-test-quality-metrics/spec.md`
**Date**: 2026-02-22
**Status**: Resolved

## Constitution Alignment Check

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Accuracy | PASS | Assertion mapping uses full SSA data flow for high accuracy. Unmapped assertions reported separately. Confidence indicator on mapping. |
| II. Minimal Assumptions | PASS | No annotation or restructuring required. Unsupported assertion libraries reported as "unrecognized" with confidence note. |
| III. Actionable Output | PASS | FR-004 (gap reporting), FR-010 (summary line), suggestions for over-specified assertions. |

---

## Resolved Ambiguities

### A-01: What constitutes an "assertion" — RESOLVED

An **assertion** is defined as a control-flow branch in test code that
evaluates the output or observable effect of the target function and
terminates or reports failure if the evaluation fails. Enumerated
patterns for v1:

**Standard library patterns:**
- `if got != want { t.Errorf(...) }` — comparison + failure report
- `if err != nil { t.Fatal(err) }` — error nil check (counts as
  assertion on the error return side effect)
- `t.Errorf(...)` / `t.Fatalf(...)` — direct failure (assertion only
  if preceded by a comparison involving target output)
- `t.Error("...")` without comparison — NOT an assertion (no mapping
  to a specific side effect; counted as unmapped)

**testify patterns:**
- `assert.Equal(t, got, want)` / `require.Equal(t, got, want)`
- `assert.NoError(t, err)` / `require.NoError(t, err)`
- `assert.Nil(t, err)` / `require.Nil(t, err)`
- All `assert.*` / `require.*` variants that take a `got` argument

**go-cmp patterns:**
- `if diff := cmp.Diff(want, got); diff != "" { t.Errorf(...) }`

Assertion detection is a two-level model:
1. **Detection**: Finding assertion sites in test code (syntactic)
2. **Mapping**: Linking each assertion to a specific side effect via
   SSA data flow analysis

### A-02: Assertion-to-side-effect mapping mechanism — RESOLVED

**Approach: Full SSA data flow analysis.**

Gaze builds SSA form for the test function and traces values from the
target function call site through the SSA graph to assertion sites.

Pipeline:
1. Identify the target function call in the test body
2. Track each return value through SSA `Value` edges (phi nodes,
   extracts, stores, loads)
3. When a tracked value reaches an assertion site (comparison,
   testify call, cmp.Diff), create the mapping
4. For mutation side effects, track the mutated value (receiver,
   pointer arg) from the call site to assertion sites
5. Values that reach assertions → mapped. Values that don't → gaps.
   Assertions on values not traced from the target → unmapped.

Unmapped assertions are acceptable and reported as such. The mapping
reports a confidence score per assertion.

### A-03: Test-to-target pairing mechanism — RESOLVED

**Approach: Call graph inference.**

Gaze analyzes the test function body to identify the primary function
under test:

1. Build the call graph for the test function
2. Identify calls to functions in the target package (excluding
   stdlib, test helpers, and setup/teardown)
3. If exactly one target function is called → automatic pairing
4. If multiple target functions are called → report all, compute
   metrics per target separately, warn the user
5. If no target function is identified → skip with warning

No `--target` flag required for the common case. An optional
`--target` flag is available for disambiguation.

### A-04: Relationship with Spec 004 (CRAP integration) — RESOLVED

Spec 003 produces a standalone `QualityReport` via the new `gaze
quality` subcommand. The contract coverage percentage from the report
is the value that Spec 004 consumes to compute GazeCRAP.

**Integration boundary**: Spec 003 exposes a `quality.Assess()`
function that returns `QualityReport` with `ContractCoverage` data.
Spec 004's `crap.Analyze()` calls this function to populate
`Score.ContractCoverage` and compute `GazeCRAP`. The CRAP integration
is Spec 004's responsibility (T050).

### A-05: Unsupported assertion libraries — RESOLVED

Assertions from unsupported libraries are reported as **"unrecognized
assertions"** with a count. The quality report includes an
`assertion_detection_confidence` field (0-100) indicating what
fraction of test code was successfully parsed for assertion patterns.

- High confidence (>= 90%): most assertions recognized
- Low confidence (< 70%): warning that coverage may be understated
- This is distinct from mapping confidence (per-assertion)

Users see: "Contract Coverage: 75% (3/4) | Assertion Detection: 85%
(some assertions may not be recognized)"

### A-06: Definition of "asserted on" — RESOLVED

A side effect is **"asserted on"** when the test contains a
control-flow path that branches on the value produced by that side
effect. Specifically:

- **Return values**: asserted if the return value is captured and
  reaches an assertion site via SSA data flow
- **Error returns**: `if err != nil { t.Fatal(err) }` counts as
  asserting on the error return (verifying contract: "does not
  error")
- **Mutations**: asserted if the mutated value is read after the call
  and reaches an assertion site
- **Discarded returns** (`_ = Foo()`): definitively unasserted
- **Captured but unchecked** (`result, _ := Foo()`): unasserted for
  the result, discarded for the error

---

## Resolved Design Decisions

### D-01: CLI entry point — New `gaze quality` subcommand

Quality metrics get their own subcommand, consistent with `gaze crap`.
This keeps the `analyze` command focused on side effect detection and
the `quality` command focused on test quality metrics.

### D-02: Over-Specification Score — count AND ratio

FR-002 defines the count. Additionally, an over-specification ratio
is computed: `incidental_assertions / total_assertions`. This enables
comparison across functions with different numbers of side effects.

### D-03: Ambiguous effects — excluded from denominator

Contract Coverage denominator is contractual effects only. Ambiguous
effects are excluded entirely from both Contract Coverage and
Over-Specification calculations. They appear in a separate "ambiguous"
section of the report.

**Consequence**: Improving classification confidence (ambiguous →
contractual) may decrease coverage score by increasing the
denominator. This is the intended behavior — the metric becomes more
accurate as classification improves.

### D-04: Exit codes — exit 1 for all threshold violations

All CI threshold violations (`--min-contract-coverage`,
`--max-over-specification`) use exit code 1, consistent with
`gaze crap`.

### D-05: Helper traversal — bounded to 3 levels

Test helper function call graph traversal is bounded to 3 levels
deep. Assertions in helpers at depth > 3 are not attributed to the
calling test and are counted as unmapped with a warning.

The boundary for "helper" is: functions that accept `*testing.T` or
`*testing.B` as a parameter.

### D-06: Typo fix — "Assertion" → "Assertion"

The spec's Key Entities section uses "AssertionMapping" — this is a
typo. The correct spelling is **"AssertionMapping"**. All type names
and references use the corrected spelling: `AssertionMapping` →
`AssertionMapping`.

Note: The spec.md itself is not modified (it is a historical record).
The plan and implementation use the corrected spelling.

---

## Confirmed Assumptions

### Assumption 1: Asymmetric analysis scope

Side effect detection (Spec 001) is non-transitive (direct function
body only). Assertion detection (Spec 003) IS transitive within the
test function's call graph — it follows helper calls (up to 3 levels)
and `t.Run` sub-test closures.

### Assumption 2: Classification is a prerequisite

The metrics pipeline is: `analyze` → `classify` → `quality-metrics`.
Running `gaze quality` implies classification. If `--classify` is not
explicitly set, the `quality` subcommand enables it automatically.

### Assumption 3: Both internal and external test packages

Both `package foo` and `package foo_test` test files are in scope.
Gaze scans `*_test.go` files in the target package directory.

### Assumption 4: All analysis is static

Assertion detection uses AST + SSA analysis. No test execution or
runtime instrumentation. Consistent with Gaze's existing approach.

---

## Risks (with mitigations)

### R-01: SSA data flow mapping accuracy — HIGH

Full SSA data flow is powerful but may struggle with:
- Struct field access chains (`result.Foo.Bar`)
- Map/slice indexing (`m["key"]`)
- Type assertions and conversions
- Values passed through channels or goroutines

**Mitigation**: Report mapping confidence per assertion. Target
>= 95% for direct returns, >= 85% for mutations, >= 75% for complex
patterns. Unmapped is an acceptable fallback.

### R-02: Helper traversal complexity — MEDIUM

Bounded to 3 levels, but popular test utility packages (testify
itself) wrap multiple internal calls. The `*testing.T` parameter
boundary helps, but some helpers don't take `*testing.T` directly.

**Mitigation**: Document the limitation. Consider expanding the
boundary to "functions in the same module" in a future iteration.

### R-03: Table-driven test union may overcount — MEDIUM

Union coverage across sub-tests means no single row needs full
coverage. This is arguably correct but may mask gaps.

**Mitigation**: Report both union coverage (spec definition) and note
the sub-test count. Per-row coverage is deferred to a future spec.

### R-04: Scope of this spec is large — HIGH

This spec introduces:
- A new SSA-based assertion detection engine
- A new SSA data flow mapping system
- A new `quality` CLI subcommand
- New test fixture infrastructure (meta-tests)
- JSON schema extension
- CI threshold enforcement

This is comparable in scope to Specs 001 + 002 combined.

**Mitigation**: Phase the implementation carefully. Ship assertion
detection and contract coverage (US1) first, then over-specification
(US2), then progress tracking (US3) and CI (US4).
