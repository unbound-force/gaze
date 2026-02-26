# Implementation Plan: Agent-Oriented Quality Report Enhancements

**Branch**: `006-agent-quality-report-enhancements` | **Date**: 2026-02-26 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `specs/006-agent-quality-report-enhancements/spec.md`
**Clarifications**: [spec.md#clarifications](spec.md)

## Summary

Enhance `gaze quality` text and JSON output with four agent-oriented
improvements: (1) `unmapped_reason` typed field on `AssertionMapping`
explains *why* an assertion is unmapped so agents can triage correctly;
(2) `gap_hints` and `discarded_return_hints` parallel string arrays on
`ContractCoverage` provide Go code snippets telling agents *what assertion
code to write* for each gap and discarded return; (3) the text formatter
expands three previously count-only sections (unmapped assertions,
ambiguous effects, discarded returns) to per-item lists with locations;
(4) the JSON schema is updated additively for all new fields. All changes
are confined to `internal/taxonomy`, `internal/quality`, and
`internal/report/schema.go`. No changes to the analysis or classification
pipeline. **Implementation is complete** — this plan documents the
delivered design and implementation decisions.

## Technical Context

**Language/Version**: Go 1.24+
**Primary Dependencies**:
  - `github.com/unbound-force/gaze/internal/taxonomy` (domain types)
  - `github.com/unbound-force/gaze/internal/quality` (metrics engine)
  - `github.com/unbound-force/gaze/internal/report` (JSON schema)
  - `github.com/charmbracelet/lipgloss` (text formatter styling)
**Storage**: N/A — stateless analysis, output to stdout
**Testing**: Standard library `testing` only, `-race -count=1`
**Target Platform**: Any platform with Go 1.24+ toolchain
**Project Type**: Single binary CLI
**Performance Goals**: O(n) on gap/effect count; no measurable latency
  addition — hint derivation is a single switch statement per effect
**Constraints**: Additive JSON schema changes only (omitempty on all new
  fields); no breaking changes to existing consumers
**Scale/Scope**: Typical Go modules; changes affect only the quality
  reporting layer, not the analysis or SSA pipeline

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Rationale |
|-----------|--------|-----------|
| **I. Accuracy** | PASS | No changes to side effect detection or assertion mapping logic. New fields (`UnmappedReason`, `GapHints`, `DiscardedReturnHints`) are derived from existing, already-accurate analysis data. `classifyUnmappedReason` is deterministic from `site.Depth` and `objToEffectID` state. Known v1 limitation (mixed return+mutation inline-call misclassification) documented and accepted. |
| **II. Minimal Assumptions** | PASS | Hints are derived solely from the taxonomy `SideEffectType` constants — no project-specific conventions, annotations, or restructuring required. Generic fallback for all exotic P2–P4 types ensures no assumption about which effects a package uses. |
| **III. Actionable Output** | PASS | This spec directly strengthens Principle III. `UnmappedReason` turns opaque unmapped counts into a decision tree. `GapHints` and `DiscardedReturnHints` tell agents precisely what Go assertion code to write. Per-item lists for ambiguous effects enable targeted GoDoc fixes. |

**GATE RESULT: PASS** — All three principles satisfied.

**Post-design re-check**: PASS — Clarification session confirmed hints on
discarded returns (FR-009a), inline-call heuristic limitation documented,
versioning deferred to first release. No principle violations introduced.

## Project Structure

### Documentation (this feature)

```text
specs/006-agent-quality-report-enhancements/
├── spec.md              # Feature specification (Complete, clarified)
├── plan.md              # This file
├── research.md          # Phase 0: technology research and decisions
├── data-model.md        # Phase 1: new taxonomy types and fields
├── quickstart.md        # Phase 1: how to observe the new output
└── tasks.md             # Task breakdown (complete, all checked)
```

### Source Code (changes only)

```text
internal/
├── taxonomy/
│   └── types.go              # UnmappedReasonType + 3 constants,
│                             # AssertionMapping.UnmappedReason,
│                             # ContractCoverage.GapHints,
│                             # ContractCoverage.DiscardedReturnHints
├── quality/
│   ├── mapping.go            # classifyUnmappedReason(), hasReturnEffects()
│   ├── hints.go              # NEW: hintForEffect() — effect type → Go snippet
│   ├── coverage.go           # GapHints populated alongside Gaps
│   ├── quality.go            # DiscardedReturnHints populated inline
│   ├── report.go             # All 4 text formatter improvements
│   ├── hints_test.go         # NEW: internal tests for hintForEffect + reasons
│   └── quality_test.go       # 10 new external tests (T011–T014 + clarify Q1)
└── report/
    └── schema.go             # unmapped_reason, gap_hints,
                              # discarded_return_hints added to QualitySchema
```

**Structure Decision**: Single project layout. All changes are additive
within existing packages. No new packages required — `hints.go` is a new
file inside `internal/quality` because hint derivation is quality-specific
logic and should not pollute the taxonomy package.

## Key Design Decisions

### 1. Typed constant for UnmappedReasonType

A typed string constant (mirroring `SideEffectType` and `Tier`) rather
than a plain string provides:
- Compile-time safety when switching on reason values
- An enumerable closed set for the JSON schema (`enum` constraint)
- Alignment with existing taxonomy conventions

Three values cover all observed causes — `helper_param`, `inline_call`,
`no_effect_match` — and are sufficient for agent decision-making without
requiring finer granularity in v1.

### 2. Parallel []string arrays for hints (GapHints, DiscardedReturnHints)

Chosen over a wrapper struct (`QualityGap`) or adding `AssertionHint` to
`SideEffect` for two reasons:

- **Schema backward compatibility**: `Gaps []SideEffect` and
  `DiscardedReturns []SideEffect` are already serialized and consumed.
  Changing the element type would break existing JSON consumers.
- **Separation of concerns**: `SideEffect` is a core analysis type used
  across Specs 001–004. Embedding quality-specific hint data into it
  would pollute the analysis output for all consumers.

The parallel-array invariant (`len(GapHints) == len(Gaps)` and
`len(DiscardedReturnHints) == len(DiscardedReturns)`) is enforced as
a postcondition of `ComputeContractCoverage` and `Assess` respectively.

### 3. hintForEffect in a dedicated hints.go

Isolated in `internal/quality/hints.go` rather than inlined in
`coverage.go` or `report.go` for two reasons:
- The function is called from two different sites (coverage.go and
  quality.go) — a shared location avoids duplication
- Isolation enables targeted unit testing and future expansion (e.g.,
  per-language hints, configurable templates) without touching metric
  or formatter logic

The function remains unexported — it is an internal derivation helper.
Callers access results through `ContractCoverage.GapHints` and
`ContractCoverage.DiscardedReturnHints`.

### 4. classifyUnmappedReason in mapping.go

Co-located with `MapAssertionsToEffects` because it has direct access to
the three signals needed for classification: `site.Depth`, `objToEffectID`
(built by `traceTargetValues`), and the `effects` list. Extracting it to
a separate file would require passing these as parameters through an
additional layer.

### 5. v1 inline-call heuristic limitation

`classifyUnmappedReason` uses `len(objToEffectID) == 0` as a proxy for
"return values were not traced." This correctly identifies inline-call
patterns for functions with only return/error effects. For functions with
**both** return and mutation effects, mutation tracing populates
`objToEffectID`, causing an inline-called return to be classified as
`no_effect_match` instead of `inline_call`. This is documented as a known
v1 limitation in the spec Edge Cases section — `no_effect_match` is still
actionable for agents, and the full fix belongs with the broader
inline-call tracing work tracked in `report.md`.

## Data Pipeline

The existing pipeline is unchanged:

```
loader.Load → analysis.Analyze → classify.Classify → quality.Assess → report
```

Internal changes within `quality.Assess`:

```
MapAssertionsToEffects  ← now populates UnmappedReason per unmapped site
ComputeContractCoverage ← now populates GapHints alongside Gaps
Assess (post-coverage)  ← now populates DiscardedReturnHints alongside DiscardedReturns
WriteText               ← now renders 4 expanded detail sections + hints
WriteJSON               ← unchanged (taxonomy fields serialize automatically)
```

## Requirement Mapping

| FR | Requirement | Component | Status |
|----|-------------|-----------|--------|
| FR-001 | UnmappedReasonType type + constants | `taxonomy/types.go` | ✅ |
| FR-002 | UnmappedReason field on AssertionMapping | `taxonomy/types.go` | ✅ |
| FR-003 | Populate UnmappedReason in MapAssertionsToEffects | `quality/mapping.go` | ✅ |
| FR-004 | GapHints field on ContractCoverage | `taxonomy/types.go` | ✅ |
| FR-005 | ComputeContractCoverage populates GapHints | `quality/coverage.go` | ✅ |
| FR-006 | hintForEffect for all P0/P1 types, generic for P2–P4 | `quality/hints.go` | ✅ |
| FR-007 | WriteText: unmapped per-item detail + reason | `quality/report.go` | ✅ |
| FR-008 | WriteText: hint line per gap | `quality/report.go` | ✅ |
| FR-009 | WriteText: Discarded returns section with hints | `quality/report.go` | ✅ |
| FR-009a | DiscardedReturnHints field + JSON serialization | `taxonomy/types.go`, `quality/quality.go` | ✅ |
| FR-010 | WriteText: ambiguous effects per-item | `quality/report.go` | ✅ |
| FR-011 | QualitySchema updated | `report/schema.go` | ✅ |

## Hint Templates by Effect Type

| Effect Type | Tier | Hint |
|---|---|---|
| `ErrorReturn` | P0 | `if err != nil { t.Fatal(err) }` |
| `ReturnValue` | P0 | `got := target(); // assert got == expected` |
| `SentinelError` | P0 | `if !errors.Is(err, ExpectedErr) { t.Errorf(...) }` |
| `ReceiverMutation` | P0 | `// assert receiver.{Target} after calling target()` |
| `PointerArgMutation` | P0 | `// assert *{Target} after calling target()` |
| `SliceMutation` | P1 | `// assert slice contents after calling target()` |
| `MapMutation` | P1 | `// assert map contents after calling target()` |
| `GlobalMutation` | P1 | `// assert {Target} (global) after calling target()` |
| `WriterOutput` | P1 | `// assert bytes written to {Target} after calling target()` |
| `HTTPResponseWrite` | P1 | `// assert HTTP response status and body after calling target()` |
| `ChannelSend` | P1 | `// assert value sent on {Target} after calling target()` |
| `ChannelClose` | P1 | `// assert {Target} is closed after calling target()` |
| `DeferredReturnMutation` | P1 | `// assert named return value after calling target()` |
| All P2–P4 | P2–P4 | `// assert {Type} side effect of target()` |
