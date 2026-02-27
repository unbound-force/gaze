# Feature Specification: Assertion Mapping Depth

**Feature Branch**: `007-assertion-mapping-depth`
**Created**: 2026-02-27
**Status**: Ready for Implementation
**Input**: User description: "Spec 007: Assertion Mapping Depth — Reaching 80% Contract Coverage."

## Clarifications

### Session 2026-02-27

- Q: How should US5 (Helper Return Value Tracing) be characterized relative to the original scope? → A: US5 is a justified scope expansion — a deliberate decision driven by research findings (research.md R3), not scope creep. The research phase proved expression resolution alone (US1-US3) reaches only ~45-55% weighted average contract coverage because the `analysis` package (108 pairs, 51% of all pairs) uses helper indirection. US5 was added to achieve the 80% SC-001 target, making it load-bearing for the primary success criterion.
- Q: Is SC-001's 80% weighted average contract coverage a hard gate or best-effort target? → A: 80% is a best-effort target. If not met after full implementation of US1-US5, SC-001 should be revised to the actual achieved value and the remaining gap documented as a known limitation with follow-on tracking, rather than blocking the release.
- Q: Should the variable shadowing edge case be promoted to a formal FR? → A: Yes. Add FR-016 for variable shadowing to ensure full traceability from requirement to task to test. The edge case already uses MUST language and describes a distinct failure mode (traced variable reassigned in same scope).
- Q: For US5 helper verification (FR-014), does "calls the target" mean structural presence in SSA or data-flow tracing of the return value? → A: Structural presence only. The helper must contain a `*ssa.Call` instruction whose callee is the target function (verified by iterating the helper's SSA blocks). No data-flow tracing of the return value path is required. This keeps implementation simple and avoids the complexity of full SSA value chain analysis for marginal accuracy gain. Confidence 65 already reflects this reduced precision.

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

**Independent Test**: Run `gaze quality` on packages where tests use
selector assertion patterns — `multilib` (testify-style `user.Name`),
`crap` (struct field assertions on Report), or `classify` (nested
slice/struct access). Before: these packages report 58-69% contract
coverage. After: test-target pairs that assert on return value fields
report coverage for their `ReturnValue` effect. (Note: the `analysis`
package requires US5 helper tracing in addition to selector matching.)

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

### User Story 5 — Helper Return Value Tracing (Priority: P2)

A developer writes tests that delegate target function calls to helper
functions (e.g., `result := analyzeFunc(t, "returns", "SingleReturn")`
where `analyzeFunc` internally calls the target function
`analysis.AnalyzeFunctionWithSSA`). Currently `traceReturnValues` only
traces when the target call appears directly in an assignment statement
in the test function's AST. When the target call is inside a helper,
`findAssignLHS` fails because the helper's AST is in a different
function body, and `objToEffectID` remains empty.

After this feature, when the direct target call assignment search
fails, the mapping engine searches the test function's AST for
assignments whose RHS is a call to the helper function that
(transitively) returns the target's result. The helper's return
variable is then traced as if it were the target's return value.

**Why this priority**: The `analysis` package has 108 test-target pairs
(51% of all pairs) that use helper indirection exclusively. Without
this fix, the weighted average contract coverage cannot reach 80%.
This is equal priority with US2 because it addresses a distinct
root cause that affects the largest package.

**Independent Test**: Run `gaze quality` on the `analysis` package
before and after. Before: ~6.9% average contract coverage. After:
test-target pairs using `result := analyzeFunc(t, ...)` followed by
assertions on `result.SideEffects` (combining this with US1 selector
resolution) report coverage for the `ReturnValue` effect.

**Acceptance Scenarios**:

1. **Given** a test that calls a helper `result := helper(t, args...)`
   where the helper internally calls the target and returns its
   result, **When** `gaze quality` runs, **Then** `result` is traced
   to the target's `ReturnValue` effect and assertions on `result`
   (or `result.Field` via US1) are mapped.
2. **Given** a test that calls a helper returning a value unrelated
   to the target function, **When** `gaze quality` runs, **Then** no
   false positive mapping is created.
3. **Given** a helper that wraps the target call with additional
   processing (e.g., `result := target(); validate(result); return result`),
   **When** `gaze quality` runs, **Then** the mapping still succeeds
   because the helper's return value originates from the target call.

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
- Helper return tracing MUST only activate when direct target call
  assignment search fails — it is a fallback, not the primary path.
- A helper that calls multiple functions (including the target) MUST
  still correctly trace the return value if the helper's return
  originates from the target call.
- A helper that does NOT call the target function MUST NOT cause
  false positive tracing — the return value tracing must verify the
  helper transitively invokes the target.

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
  TestSC003_MappingAccuracy MUST be raised to reflect the actual
  achieved accuracy upon successful implementation. *Amended*: The
  original target of 85% was not achievable with US1-US5 alone.
  Measured accuracy reached 78.8% (52/66); the ratchet floor is set
  to 76.0% (~3-point margin). The remaining gap to 85% requires
  helper parameter tracing and testify field access resolution,
  tracked as follow-on work.
- **FR-012**: All existing assertion mappings that succeed at
  confidence 75 MUST continue to succeed at the same confidence.
  No regressions in existing mapping behavior.
- **FR-013**: When `findAssignLHS` fails to find a direct assignment
  for the target call, the mapping engine MUST search the test
  function's AST for assignments whose RHS calls a function that
  transitively invokes the target, and trace the LHS variable as
  the return value receiver at indirect confidence (65).
- **FR-014**: Helper return tracing MUST verify that the helper
  function actually calls the target function before tracing.
  Verification uses structural presence: the helper's SSA must
  contain a `*ssa.Call` instruction whose callee is the target
  function, verified by iterating the helper's SSA blocks and
  instructions. No data-flow tracing of the return value path is
  required. A helper that does not call the target MUST NOT produce
  a mapping.
- **FR-015**: Helper return tracing MUST be limited to depth 1 —
  only direct callers of the target, not transitive chains of
  helpers calling helpers. This bounds complexity and prevents
  false positive traces through long call chains.
- **FR-016**: When a traced return value variable is reassigned in
  the same scope (e.g., `result = localValue` after
  `result := target()`), assertions on the reassigned variable MUST
  NOT be mapped to the original return value's side effect. The
  mapping engine must respect the most recent assignment.
  *Known limitation*: Plain `=` reassignment reuses the same
  `types.Object` in Go's type system, making it indistinguishable
  from the original assignment via object identity alone. FR-016 is
  fully enforceable only for `:=` scope shadowing (which creates a
  new `types.Object`). Enforcing FR-016 for plain reassignment
  requires SSA value-flow analysis, tracked as follow-on work.

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
  packages with tests reaches >= 80% (best-effort target), measured
  by running `gaze quality --format=json` on each package and
  computing the test-target-pair-weighted average of
  `contract_coverage.percentage`. This requires both expression
  resolution (US1-US3) and helper return tracing (US5) working
  together. Expression resolution alone is projected to reach only
  ~45-55% (see research.md R3) because the `analysis` package (51%
  of pairs) uses helper indirection. US5 was added to scope to close
  this gap. If the 80% target is not met after full implementation,
  the criterion should be revised to the actual achieved value and
  the remaining gap documented as a known limitation.
- **SC-002**: The mapping accuracy ratchet (TestSC003_MappingAccuracy)
  floor is raised to reflect the actual achieved accuracy across the
  standard test fixtures. *Amended*: The original target of >= 85%
  was not achievable with expression resolution and helper return
  tracing alone. Measured accuracy: 78.8% (52/66); ratchet floor:
  76.0%. The remaining 6.2pp gap to 85% requires helper parameter
  tracing and testify argument resolution (follow-on work).
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
