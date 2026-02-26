# Implementation Plan: Agent-Oriented Quality Report Enhancements

**Branch**: `006-agent-quality-report-enhancements` | **Date**: 2026-02-26 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `specs/006-agent-quality-report-enhancements/spec.md`

## Summary

Enhance `gaze quality` output to give agents (and humans) enough
information to act on missing assertions without re-implementing
Gaze's internal taxonomy knowledge. Four improvements:

1. **Unmapped assertion reasons** — each unmapped assertion carries
   a typed reason (`helper_param`, `inline_call`, `no_effect_match`)
   in both JSON and text output, enabling agents to triage correctly.
2. **Gap assertion hints** — each coverage gap carries a Go code
   snippet showing how to write the missing assertion, in both JSON
   (`gap_hints` array) and text output.
3. **Discarded returns in text** — `discarded_returns` (already in
   JSON) surfaces in text output as a distinct section.
4. **Ambiguous effects detail in text** — ambiguous effects expand
   from a count to a per-item list (data already in JSON).

## Technical Context

**Language/Version**: Go 1.24+
**Primary Dependencies**: existing — `internal/taxonomy`, `internal/quality`, `internal/report`
**Storage**: N/A (stateless)
**Testing**: Standard library `testing` package only, `-race -count=1`
**Target Platform**: Any platform with Go toolchain
**Project Type**: Single binary CLI
**Performance Goals**: No additional analysis cost; hint derivation is O(n) on gap count
**Constraints**: Additive changes only — no breaking changes to existing JSON schema
**Scale/Scope**: Changes confined to `internal/taxonomy/types.go`,
  `internal/quality/` (mapping.go, coverage.go, report.go + new hints.go),
  and `internal/report/schema.go`

## Constitution Check

| Principle | Status | Rationale |
|-----------|--------|-----------|
| **I. Accuracy** | PASS | No changes to detection or mapping logic. New fields are derived from existing, already-accurate analysis data. Unmapped reason derivation is deterministic from `site.Depth` and `objToEffectID` state. |
| **II. Minimal Assumptions** | PASS | Hints are derived solely from the taxonomy type constants — no project-specific conventions or annotations required. Generic fallback for exotic types. |
| **III. Actionable Output** | PASS | This spec directly strengthens Principle III. Reasons tell agents *why* an assertion is unmapped; hints tell agents *what code to write*; discarded returns show the most definitive gaps; ambiguous detail shows which effects to improve. |

**GATE RESULT: PASS** — Proceed to implementation.

## Project Structure

### Documentation (this feature)

```text
specs/006-agent-quality-report-enhancements/
├── spec.md              # Feature specification (this feature)
├── plan.md              # This file
└── tasks.md             # Task breakdown
```

### Source Code (changes only)

```text
internal/
├── taxonomy/
│   └── types.go              # Add UnmappedReasonType + constants,
│                             # UnmappedReason field on AssertionMapping,
│                             # GapHints field on ContractCoverage
├── quality/
│   ├── mapping.go            # Populate UnmappedReason on unmapped mappings
│   ├── hints.go              # NEW: hintForEffect() — effect type → Go snippet
│   ├── coverage.go           # Populate GapHints via hintForEffect()
│   ├── report.go             # Expand text formatter for all four improvements
│   └── quality_test.go       # New + updated tests
└── report/
    └── schema.go             # Add unmapped_reason + gap_hints to QualitySchema
```

## Key Design Decisions

### 1. Typed constant for UnmappedReasonType

Using a typed string constant (like `SideEffectType` and `Tier`) rather
than a plain string gives compile-time safety and an enumerable set of
values for the JSON schema. Three values cover all observed causes:

- `helper_param` — `site.Depth > 0`: the assertion is inside a helper
  body; Gaze cannot trace the helper's parameter objects back to the
  test's variable assignments.
- `inline_call` — depth 0, no traced return values, AND the target has
  return/error effects: the caller invoked the target inline without
  assigning the return value (e.g., `if c.Value() != 5`).
- `no_effect_match` — all other cases: the assertion is in the test
  body, return values were traced, but no identifier in the assertion
  expression matched the traced objects. Typically a cross-target
  assertion (the assertion tests a different function's output).

The detection logic in `MapAssertionsToEffects` has access to
`site.Depth` (from `AssertionSite`) and to `objToEffectID` (built by
`traceTargetValues`). The inline-call detection needs one additional
signal: whether `findAssignLHS` returned nil *because* the call was
inside an expression rather than an assignment. This is approximated
by: depth == 0 AND `len(objToEffectID) == 0` AND the target has at
least one ReturnValue or ErrorReturn effect. If the target has only
mutation effects, `objToEffectID` may be empty for other reasons; in
that case `no_effect_match` is the correct default.

### 2. GapHints as a parallel []string field

`ContractCoverage.GapHints` is a `[]string` parallel to `Gaps
[]SideEffect`. The parallel-array pattern is used rather than a wrapper
struct for two reasons:

a. `Gaps []SideEffect` is already serialized as `SideEffectRef` in the
   JSON schema; changing the element type would be a breaking schema
   change.
b. Consumers that only need the gap side effect data (without hints)
   should not be forced to unwrap a new struct.

The parallel invariant (`len(GapHints) == len(Gaps)`) is an enforced
postcondition of `ComputeContractCoverage`.

### 3. Hint derivation in a dedicated hints.go

`hintForEffect` lives in a new `internal/quality/hints.go` file rather
than being inlined in `coverage.go` or `report.go`. This isolates hint
logic for testing and future expansion (e.g., per-language hints,
configurable templates).

### 4. Text formatter expansion

The four text output changes (FR-007 through FR-010) are all confined
to `internal/quality/report.go`. No new formatter files are needed.

## Data Pipeline Changes

The existing pipeline is unchanged:

```
loader.Load → analysis.Analyze → classify.Classify → quality.Assess → report
```

Internal changes within `quality.Assess`:

```
MapAssertionsToEffects  ← now populates UnmappedReason per unmapped site
ComputeContractCoverage ← now populates GapHints via hintForEffect()
WriteText               ← now renders 4 additional detail sections
WriteJSON               ← unchanged (taxonomy fields serialize automatically)
```

## Hint Templates by Effect Type

| Effect Type | Hint |
|---|---|
| `ErrorReturn` | `if err != nil { t.Fatal(err) }` |
| `ReturnValue` | `got := target(); // assert got == expected` |
| `ReceiverMutation` | `// assert receiver.{Target} after calling target()` (or generic if Target is empty) |
| `PointerArgMutation` | `// assert *{Target} after calling target()` |
| `SliceMutation` | `// assert slice contents after calling target()` |
| `MapMutation` | `// assert map contents after calling target()` |
| `GlobalMutation` | `// assert global state after calling target()` |
| `WriterOutput` | `// assert bytes written to {Target} after calling target()` |
| `ChannelSend` | `// assert value sent on {Target} after calling target()` |
| All other P2-P4 | `// assert {Type} side effect of target()` |

## Requirement Mapping

| FR | Requirement | Component |
|----|-------------|-----------|
| FR-001 | UnmappedReasonType type + constants | `taxonomy/types.go` |
| FR-002 | UnmappedReason field on AssertionMapping | `taxonomy/types.go` |
| FR-003 | Populate UnmappedReason in MapAssertionsToEffects | `quality/mapping.go` |
| FR-004 | GapHints field on ContractCoverage | `taxonomy/types.go` |
| FR-005 | ComputeContractCoverage populates GapHints | `quality/coverage.go` |
| FR-006 | hintForEffect for all P0/P1 types, generic for P2-P4 | `quality/hints.go` |
| FR-007 | WriteText: unmapped per-item detail + reason | `quality/report.go` |
| FR-008 | WriteText: hint line per gap | `quality/report.go` |
| FR-009 | WriteText: Discarded returns section | `quality/report.go` |
| FR-010 | WriteText: ambiguous effects per-item | `quality/report.go` |
| FR-011 | QualitySchema updated | `report/schema.go` |
