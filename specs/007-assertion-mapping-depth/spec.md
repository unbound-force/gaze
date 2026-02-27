# Feature Specification: Assertion Mapping Depth

**Feature Branch**: `007-assertion-mapping-depth`
**Created**: 2026-02-27
**Status**: Draft
**Input**: User description: "Spec 007: Assertion Mapping Depth — Reaching 80% Contract Coverage."

## User Scenarios & Testing *(mandatory)*

### User Story 1 — Selector Expression Matching (Priority: P1)

A developer runs `gaze quality` on a package where tests assert on
fields of a returned struct (e.g., `result.SideEffects`,
`cfg.Classification.Thresholds.Contractual`). Currently the mapping
engine only recognizes assertions that reference the exact variable
assigned from the target call. Any field access, method call, or
indexing on that variable is invisible to the mapper, causing the
assertion to go unmapped and the return value's contract coverage
to report 0%.

After this feature, the mapping engine recognizes that
`result.SideEffects` is an assertion on `result` — the traced return
value — and maps it to the corresponding `ReturnValue` side effect.
This single improvement addresses the largest source of unmapped
assertions across the project.

**Why this priority**: Selector expressions are the dominant assertion
pattern in the project. The `analysis` package (108 test-target pairs,
51% of all pairs) scores 6.9% contract coverage solely because every
test asserts on struct fields of the return value. Fixing this one
pattern has the highest impact on the weighted average.

**Independent Test**: Run `gaze quality` on the `analysis` package
before and after the change. Before: ~6.9% average contract coverage.
After: the majority of test-target pairs that assert on return value
fields report coverage for their `ReturnValue` effect.

**Acceptance Scenarios**:

1. **Given** a test that assigns `result := target()` and asserts on
   `result.FieldName`, **When** `gaze quality` runs, **Then** the
   `ReturnValue` effect for `target` is marked as covered.
2. **Given** a test that assigns `result := target()` and asserts on
   a nested field `result.A.B.C`, **When** `gaze quality` runs,
   **Then** the `ReturnValue` effect for `target` is marked as covered.
3. **Given** a test that assigns `got, err := target()` and asserts
   on `got.Field`, **When** `gaze quality` runs, **Then** the
   `ReturnValue` effect is covered and the `ErrorReturn` effect
   remains unaffected (it is covered or uncovered based on `err`
   assertions independently).
4. **Given** a test that assigns `result := target()` and passes
   `result.Field` as an argument to a helper (e.g.,
   `countEffects(result.SideEffects, ...)`), **When** `gaze quality`
   runs, **Then** the `ReturnValue` effect for `target` is marked as
   covered.

---

### User Story 2 — Built-in Call Unwinding (Priority: P2)

A developer runs `gaze quality` on a package where tests use `len()`,
`cap()`, or other built-in functions on a return value (e.g.,
`len(docs)` where `docs` was returned by the target function).
Currently the `len(docs)` call wraps the traced identifier in a
call expression, and the mapping engine cannot see through it.

After this feature, when a built-in function call wraps a traced
object, the mapper unwraps the call and matches the inner argument
to the corresponding side effect.

**Why this priority**: `len()` assertions on slices and maps are the
second most common pattern causing unmapped assertions. The `docscan`
and `analysis` packages both use `len()` checks extensively. This is
lower priority than selector expressions because it affects fewer
test-target pairs.

**Independent Test**: Run `gaze quality` on the `docscan` package
and verify that tests using `len(docs)` have their `ReturnValue`
effect marked as covered.

**Acceptance Scenarios**:

1. **Given** a test that assigns `docs := target()` and asserts on
   `len(docs)`, **When** `gaze quality` runs, **Then** the
   `ReturnValue` effect for `target` is marked as covered.
2. **Given** a test that assigns `m := target()` and asserts on
   `len(m)` where `m` is a map, **When** `gaze quality` runs,
   **Then** the `ReturnValue` effect is marked as covered.
3. **Given** a test that calls `len()` on a variable that is NOT a
   traced return value (e.g., a local slice literal), **When**
   `gaze quality` runs, **Then** no false positive mapping is created.

---

### User Story 3 — Index Expression Resolution (Priority: P3)

A developer runs `gaze quality` on a package where tests index into
a returned slice or map (e.g., `results[0].Function`,
`docs[i].Path`). Currently the indexing expression breaks the object
identity chain — the mapping engine cannot trace `results[0]` back
to the `results` variable assigned from the target call.

After this feature, when an index expression is encountered in an
assertion, the mapper resolves the collection being indexed back to
its traced return value.

**Why this priority**: Index expressions often co-occur with selector
expressions (e.g., `results[0].Function`). US1 (selector matching)
handles the `.Function` part, but without index resolution the root
`results[0]` access may still fail to trace. This story ensures the
full chain resolves correctly.

**Independent Test**: Run `gaze quality` on a test fixture containing
`results[0].Field` assertions and verify the `ReturnValue` effect
is covered.

**Acceptance Scenarios**:

1. **Given** a test that assigns `results := target()` and asserts on
   `results[0]`, **When** `gaze quality` runs, **Then** the
   `ReturnValue` effect is marked as covered.
2. **Given** a test that assigns `results := target()` and asserts on
   `results[0].Field.SubField`, **When** `gaze quality` runs, **Then**
   the `ReturnValue` effect is marked as covered (combining index +
   selector resolution).
3. **Given** an index expression on a non-traced variable (e.g., a
   local array literal), **When** `gaze quality` runs, **Then** no
   false positive mapping is created.

---

### User Story 4 — Confidence Differentiation (Priority: P4)

An agent or CI system consumes `gaze quality` JSON output and uses
the `confidence` field on assertion mappings to decide how much to
trust a mapping. Currently all SSA-traced matches report confidence
75. After US1-US3, mappings made through indirect expression
resolution (selectors, built-ins, indexing) should report a slightly
lower confidence than direct identity matches to reflect the
additional inference step.

**Why this priority**: Lowest priority because this is a refinement,
not a correctness issue. All indirect matches are still valid
mappings; the confidence signal is advisory.

**Independent Test**: Run `gaze quality --format=json` on a test
with both direct return value assertions and field-access assertions.
Verify that direct matches have confidence 75 and indirect matches
have a distinct, lower confidence value.

**Acceptance Scenarios**:

1. **Given** a test with a direct assertion on a return value (`got`),
   **When** mapped, **Then** the mapping has confidence 75.
2. **Given** a test with a selector assertion on a return value
   (`got.Field`), **When** mapped, **Then** the mapping has a
   confidence value lower than 75 but above 50.
3. **Given** JSON output, **When** parsed, **Then** the `confidence`
   field is present on all assertion mappings and values are within
   the range [50, 100].

---

### Edge Cases

- A selector expression on a variable that is NOT a traced return
  value (e.g., a test-local struct) MUST NOT produce a false positive
  mapping. Only variables traced to the target function's return or
  mutation effects are eligible.
- A deeply nested selector chain (`a.B.C.D.E`) MUST resolve correctly
  by recursively unwinding to the root identifier `a`.
- A `len()` call with multiple arguments (e.g., `copy(dst, src)`)
  MUST NOT be treated as an unwinding candidate — only single-argument
  built-in calls that inspect a value (`len`, `cap`) qualify.
- An index expression with a complex index (e.g., `results[i+1]`)
  MUST still resolve the collection root (`results`); the index value
  itself is irrelevant to the mapping.
- When a traced return value is shadowed by a later assignment in the
  same scope (e.g., `result = localValue`), assertions on the
  shadowed `result` MUST NOT be mapped to the original return value's
  side effect. The mapping engine must respect the most recent
  assignment.
- The mapping accuracy ratchet test (TestSC003_MappingAccuracy) MUST
  have its baseline floor raised to reflect the new accuracy, not
  lowered.
- All existing mapped assertions MUST remain mapped — no regressions
  in the set of assertions that currently map successfully.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The assertion mapping engine MUST resolve selector
  expressions (`x.Field`) by walking the expression tree to find the
  root identifier `x` and checking it against the traced object map.
- **FR-002**: The selector resolution MUST handle arbitrarily deep
  chains (`x.A.B.C`) by recursively unwinding `SelectorExpr` nodes
  until a root `Ident` is reached.
- **FR-003**: The assertion mapping engine MUST resolve built-in
  function calls (`len(x)`, `cap(x)`) by extracting the first
  argument and checking it against the traced object map.
- **FR-004**: Built-in call unwinding MUST be limited to value-
  inspecting built-ins (`len`, `cap`). Side-effecting built-ins
  (`append`, `delete`, `close`, `copy`, `clear`) MUST NOT be unwound.
- **FR-005**: The assertion mapping engine MUST resolve index
  expressions (`x[i]`, `x[key]`) by extracting the collection
  expression `x` and checking it against the traced object map.
- **FR-006**: Index expression resolution MUST combine with selector
  resolution so that `x[0].Field` resolves through both indexing and
  selector unwinding to the root `x`.
- **FR-007**: Mappings made through indirect resolution (selector,
  built-in, or index) MUST carry a lower confidence value than direct
  identity matches to reflect the additional inference.
- **FR-008**: The direct identity match confidence MUST remain at 75.
  Indirect matches MUST use a confidence value of 65.
- **FR-009**: The mapping engine MUST NOT produce false positive
  matches — a selector, built-in, or index expression on a variable
  that is not in the traced object map MUST be skipped without
  creating a mapping.
- **FR-010**: When a traced variable is passed as an argument to a
  function call (e.g., `helper(result.Field)`), the mapper MUST
  resolve the argument's root identifier and attempt to match it
  against the traced object map, producing a mapping at indirect
  confidence if matched.
- **FR-011**: The mapping accuracy ratchet floor in
  TestSC003_MappingAccuracy MUST be raised to at least 85% upon
  successful implementation.
- **FR-012**: All existing assertion mappings that succeed at
  confidence 75 MUST continue to succeed at the same confidence.
  No regressions in existing mapping behavior.

### Key Entities

- **Indirect Match**: An assertion mapping where the assertion
  expression does not directly reference a traced object but reaches
  it through selector access, built-in wrapping, or index access.
  Distinguished from a direct match by a lower confidence value (65
  vs 75).
- **Expression Root**: The innermost identifier reached by
  recursively unwinding selector expressions (`x.A.B` -> `x`), index
  expressions (`x[0]` -> `x`), and built-in calls (`len(x)` -> `x`).
  The root is tested against the traced object map.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: The weighted average contract coverage across all
  packages with tests reaches >= 80%, measured by running
  `gaze quality --format=json` on each package and computing the
  test-target-pair-weighted average of `contract_coverage.percentage`.
- **SC-002**: The mapping accuracy ratchet (TestSC003_MappingAccuracy)
  reaches >= 85% across the standard test fixtures, with the baseline
  floor raised accordingly.
- **SC-003**: Zero regressions — every assertion mapping that succeeds
  at confidence 75 before this change MUST continue to succeed at
  confidence 75 after. The total count of direct-confidence (75)
  mappings MUST NOT decrease.
- **SC-004**: No false positive mappings — running `gaze quality` on
  a fixture containing assertions on non-traced local variables MUST
  produce zero incorrect mappings, verified by automated test.
- **SC-005**: Analysis throughput does not regress by more than 2x —
  benchmark tests for the mapping function MUST show per-pair
  processing time within 2x of the pre-change baseline.
