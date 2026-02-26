# Feature Specification: Agent-Oriented Quality Report Enhancements

**Feature Branch**: `006-agent-quality-report-enhancements`
**Created**: 2026-02-26
**Status**: Complete

## Clarifications

### Session 2026-02-26

- Q: Should `Discarded returns:` entries also show an assertion hint (same `hint:` format as gaps)? → A: Yes — show hints under discarded returns, same `hint:` format as gaps. Also expose `discarded_return_hints` in JSON parallel to `discarded_returns`.
- Q: Is the `inline_call` heuristic (`len(objToEffectID) == 0`) acceptable for v1 when a function has mixed return + mutation effects? → A: Yes — accept for v1; document the known edge case. Full fix deferred to the inline-call tracing improvement tracked in `report.md`.
- Q: Does adding new JSON output fields (`unmapped_reason`, `gap_hints`, `discarded_return_hints`) require a semantic version bump? → A: No explicit version gate — the project has not released yet; versioning handled at release time.

---

## User Scenarios & Testing *(mandatory)*

### User Story 1 — Unmapped Assertion Reasons (Priority: P1)

An agent running `gaze quality` receives a report that contains
unmapped assertions. Currently the output says only
`Unmapped assertions: 3` with no further detail. The agent cannot
tell whether the unmapped assertions represent a mapping bug, a test
pattern Gaze cannot trace, or a legitimate cross-target attribution.
As a result the agent either ignores all unmapped assertions or acts
on them all — both are wrong.

After this feature, each unmapped assertion in both text and JSON
output carries a `reason` that classifies the root cause. The agent
can act precisely: skip helper-body assertions, rewrite inline-call
assertions, and investigate no-match assertions.

**Why this priority**: Without knowing *why* an assertion is unmapped,
the agent's downstream actions are arbitrary. This is the highest-value
improvement because it turns opaque data into a decision tree.

**Independent Test**: Run `gaze quality` on the `helpers` and
`welltested` test fixtures and verify each unmapped assertion's text
and JSON output carries a non-empty `unmapped_reason` that matches
the known cause.

**Acceptance Scenarios**:

1. **Given** a test with assertions inside a helper body (depth > 0),
   **When** `gaze quality` reports them as unmapped, **Then** each
   carries `unmapped_reason: "helper_param"` in JSON and `[helper_param]`
   in text.
2. **Given** a test with an inline call assertion (`if c.Value() != 5`),
   **When** `gaze quality` reports it as unmapped, **Then** each carries
   `unmapped_reason: "inline_call"` and the text output includes a hint
   to assign the return value first.
3. **Given** a test with an assertion that genuinely does not match any
   detected side effect, **When** `gaze quality` reports it as unmapped,
   **Then** each carries `unmapped_reason: "no_effect_match"` in JSON
   and text.
4. **Given** `--format=json` output, **When** parsed, **Then** every
   element of `unmapped_assertions` with confidence 0 has a non-empty
   `unmapped_reason` field.

---

### User Story 2 — Gap Assertion Hints (Priority: P2)

A developer or agent receives a Contract Coverage gap report. The gaps
list shows *which* side effects are unasserted but not *how* to assert
them. An agent must understand the side effect taxonomy (ErrorReturn,
ReturnValue, ReceiverMutation, etc.) to know what assertion code to
write. That is an unnecessary burden.

After this feature, each coverage gap carries a Go code snippet hint
that tells the agent exactly what assertion pattern to add. Hints appear
in both the text output (indented under the gap) and as a parallel
`gap_hints` array in JSON output.

**Why this priority**: Closes the "what do I write?" gap that currently
forces agents to re-implement taxonomy knowledge. Directly satisfies
Constitution Principle III (every output must guide toward a concrete
improvement).

**Independent Test**: Run `gaze quality` on the `undertested` fixture
and verify that every gap item in text output has an indented `hint:`
line and that `contract_coverage.gap_hints` in JSON output is non-empty
and has the same length as `contract_coverage.gaps`.

**Acceptance Scenarios**:

1. **Given** a gap with type `ErrorReturn`, **When** output is rendered,
   **Then** hint is `if err != nil { t.Fatal(err) }`.
2. **Given** a gap with type `ReturnValue`, **When** output is rendered,
   **Then** hint is `got := target(); // assert got == expected`.
3. **Given** a gap with type `ReceiverMutation` and target `"Count"`,
   **When** output is rendered, **Then** hint references the target
   field name.
4. **Given** `gap_hints` in JSON output, **When** parsed, **Then**
   `len(gap_hints) == len(gaps)` and each hint is a non-empty string.
5. **Given** a test with 100% contract coverage (no gaps), **When**
   output is rendered, **Then** `gap_hints` is omitted from JSON
   (omitempty) and no hint section appears in text.

---

### User Story 3 — Discarded Returns in Text Output (Priority: P3)

The `discarded_returns` field is present in JSON output but completely
absent from text output. When a test does `_ = target()` or ignores a
return value entirely, Gaze detects this as a definitive unassertion
(stronger signal than a gap) but this information is invisible in text.
Agents reading text output cannot see definitively unasserted effects.

**Why this priority**: `discarded_returns` are the most unambiguous
coverage gaps (the test *explicitly* threw away the value). They are
more actionable than regular gaps. Text output should surface them.

**Independent Test**: Run `gaze quality` on the `undertested` fixture
(which contains `_ = store.Set(...)` patterns) and verify the text
output includes a `Discarded returns:` section listing each discarded
effect with type, description, and location.

**Acceptance Scenarios**:

1. **Given** a test that discards a return value (`_ = target()`),
   **When** text output is rendered, **Then** a `Discarded returns:`
   section appears listing the effect type, description, location, and
   a `hint:` line with the assertion code snippet.
2. **Given** no discarded returns, **When** text output is rendered,
   **Then** no `Discarded returns:` section appears.
3. **Given** `--format=json` output, **When** parsed, **Then**
   `contract_coverage.discarded_return_hints` is present and
   `len(discarded_return_hints) == len(discarded_returns)`.
4. **Given** no discarded returns, **When** `--format=json` output is
   parsed, **Then** `discarded_return_hints` is absent from the JSON
   output (omitempty — the key must not appear, not merely be empty).

---

### User Story 4 — Ambiguous Effects Detail in Text Output (Priority: P4)

Ambiguous effects are currently shown in text output only as a count:
`Ambiguous effects (excluded): 4`. An agent trying to reduce the
ambiguous count (by adding GoDoc comments to fix classification) cannot
see which specific effects are ambiguous or where they are located.

**Why this priority**: Ambiguous effects are excluded from coverage
metrics and inflate the apparent coverage percentage. Surfacing them
with detail enables targeted GoDoc improvements. This is a formatter-
only change (data already in JSON).

**Independent Test**: Run `gaze quality` on a package with known
ambiguous effects and verify each ambiguous effect appears in text
output with type, description, and location.

**Acceptance Scenarios**:

1. **Given** a report with 2 ambiguous effects, **When** text output
   is rendered, **Then** both appear with type, description, and source
   location (not just a count).
2. **Given** no ambiguous effects, **When** text output is rendered,
   **Then** no ambiguous section appears.

---

### Edge Cases

- A report with zero unmapped assertions must not render an unmapped
  section (existing behavior preserved).
- A gap with an exotic effect type (P4 tier: ReflectionMutation, CgoCall)
  MUST produce a generic hint, not an empty string.
- The `gap_hints` JSON array MUST have exactly the same length as `gaps`;
  the parallel invariant must hold even when some hints are generic.
- When `site.Depth == 0` but `objToEffectID` is empty for reasons other
  than an inline call (e.g., mutations only), the reason MUST default to
  `no_effect_match` not `inline_call`.
- **Known v1 limitation**: When a function has both `ReturnValue` and
  `ReceiverMutation` effects and the return value is called inline, the
  mutation tracing populates `objToEffectID` (so `len > 0`), causing an
  inline-called return assertion to be classified as `no_effect_match`
  instead of `inline_call`. This is acceptable for v1 — `no_effect_match`
  is still actionable. Full resolution deferred to the inline-call tracing
  improvement documented in `report.md`.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Gaze MUST add `UnmappedReasonType` as a typed string
  constant type in `internal/taxonomy` with three values:
  `helper_param`, `inline_call`, and `no_effect_match`.
- **FR-002**: The `AssertionMapping` struct MUST include an optional
  `unmapped_reason` field of type `UnmappedReasonType`, serialized as
  `json:"unmapped_reason,omitempty"`.
- **FR-003**: `MapAssertionsToEffects` MUST populate `UnmappedReason`
  for each unmapped `AssertionMapping` based on: `site.Depth > 0` →
  `helper_param`; depth 0 with `len(objToEffectID) == 0` and return/error
  effects present → `inline_call`; all other cases → `no_effect_match`.
  *v1 known limitation*: functions with mixed return + mutation effects
  may misclassify an inline-called return as `no_effect_match` because
  mutation tracing populates `objToEffectID` — acceptable for v1.
- **FR-004**: `ContractCoverage` MUST include a `gap_hints` field
  (`[]string`, `json:"gap_hints,omitempty"`) parallel to `gaps`.
- **FR-005**: `ComputeContractCoverage` MUST populate `GapHints` with
  one Go code snippet per gap, derived from the effect's `Type` and
  `Target`.
- **FR-006**: The `hintForEffect` function MUST produce specific hints
  for all P0 and P1 effect types and a generic template for P2–P4.
- **FR-007**: The text formatter `WriteText` MUST expand unmapped
  assertions from a count to a per-item list showing location,
  assertion type, and `unmapped_reason`.
- **FR-008**: The text formatter `WriteText` MUST show a hint line
  indented under each coverage gap.
- **FR-009**: The text formatter `WriteText` MUST show a `Discarded
  returns:` section listing each discarded effect when present, with a
  `hint:` line per entry (same format as gap hints).
- **FR-009a**: `ContractCoverage` MUST include a `discarded_return_hints`
  field (`[]string`, `json:"discarded_return_hints,omitempty"`) parallel
  to `discarded_returns`, populated by `hintForEffect`.
- **FR-010**: The text formatter `WriteText` MUST expand ambiguous
  effects from a count to a per-item list showing type, description,
  and location.
- **FR-011**: The `QualitySchema` JSON Schema constant MUST be updated
  to reflect the new `unmapped_reason`, `gap_hints`, and
  `discarded_return_hints` fields.

### Key Entities

- **UnmappedReasonType**: Typed string constant. Values: `helper_param`
  (assertion inside a helper body at depth > 0), `inline_call` (target
  called inline without assignment; return value never traced),
  `no_effect_match` (no side effect object matched the assertion).
- **GapHints**: Parallel `[]string` on `ContractCoverage`, same length
  as `Gaps`. Each element is a Go code snippet describing how to assert
  on the corresponding gap effect.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Every unmapped assertion in `helpers` and `welltested`
  fixture output carries the correct `unmapped_reason` value matching
  the known cause (helper_param or inline_call), verified by automated
  test.
- **SC-002**: Every gap in `undertested` fixture JSON output has a
  corresponding non-empty entry in `gap_hints`; `len(gap_hints) ==
  len(gaps)` verified by automated test.
- **SC-003**: Text output for the `undertested` fixture contains a
  `Discarded returns:` section with at least one entry and a `hint:`
  line per entry; `discarded_return_hints` in JSON has the same length
  as `discarded_returns`, verified by automated test.
- **SC-004**: Text output for a fixture with ambiguous effects lists
  each ambiguous effect individually (type + location), not just a
  count, verified by automated test.
- **SC-005**: All existing quality tests continue to pass without
  modification (backward compatibility of JSON schema via omitempty).
- **SC-006**: `hintForEffect` returns a non-empty string for every
  `SideEffectType` constant in the taxonomy, verified by table-driven
  unit test.
