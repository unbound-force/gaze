# Tasks: Agent-Oriented Quality Report Enhancements

**Input**: `specs/006-agent-quality-report-enhancements/plan.md` and `spec.md`
**Branch**: `006-agent-quality-report-enhancements`

## Phase 1: Data Model (Blocking Prerequisites)

**Purpose**: Add `UnmappedReasonType` and `GapHints` to the taxonomy.
No logic or formatting depends on these until they exist.

**⚠️ CRITICAL**: Phases 2–4 cannot begin until Phase 1 is complete.

- [ ] T001 [P] [US1] Add `UnmappedReasonType` typed constant, three
  reason constants (`helper_param`, `inline_call`, `no_effect_match`),
  and optional `UnmappedReason UnmappedReasonType` field to
  `AssertionMapping` in `internal/taxonomy/types.go`.

- [ ] T002 [P] [US2] Add `GapHints []string` field (`json:"gap_hints,omitempty"`)
  to `ContractCoverage` in `internal/taxonomy/types.go`.

**Checkpoint**: `go build ./...` passes. Existing tests still pass.

---

## Phase 2: Logic (US1 + US2 Core Logic)

**Purpose**: Populate the new fields in the analysis pipeline.
Depends on Phase 1.

- [ ] T003 [US1] Populate `UnmappedReason` for each unmapped
  `AssertionMapping` in `internal/quality/mapping.go`:
  — `site.Depth > 0` → `UnmappedReasonHelperParam`
  — depth 0, `len(objToEffectID) == 0`, target has return/error
    effects → `UnmappedReasonInlineCall`
  — all other cases → `UnmappedReasonNoEffectMatch`
  Pass `effects` (the full effect list) and `objToEffectID` to the
  reason-determination logic so the inline-call heuristic can check
  for return/error effects.

- [ ] T004 [P] [US2] Create `internal/quality/hints.go` with
  `hintForEffect(e taxonomy.SideEffect) string` implementing the
  hint table from `plan.md`. Cover all P0/P1 types explicitly;
  use `// assert {Type} side effect of target()` as the P2-P4
  generic fallback. Include package doc comment.

- [ ] T005 [US2] Update `ComputeContractCoverage` in
  `internal/quality/coverage.go` to call `hintForEffect` for each
  gap and populate `GapHints`. Enforce `len(GapHints) == len(Gaps)`
  as a postcondition.

**Checkpoint**: `go build ./...` passes. Existing tests still pass.

---

## Phase 3: Text Formatter (US1–US4 Text Output)

**Purpose**: Expand `WriteText` in `internal/quality/report.go` to
surface all four improvements. Depends on Phase 2.

- [ ] T006 [US1] Expand unmapped assertions in `WriteText` from
  count-only to per-item list. Each line: `file:line  assertion_type
  [reason: reason_text]`. Emit count header then indented list.

- [ ] T007 [US2] Add hint line indented under each gap in the Gaps
  section of `WriteText`. Format: `        hint: <snippet>`.

- [ ] T008 [P] [US3] Add `Discarded returns:` section to `WriteText`
  when `r.ContractCoverage.DiscardedReturns` is non-empty. Format
  matches Gaps: `      - {Type}: {Description} ({Location})`.

- [ ] T009 [P] [US4] Expand ambiguous effects in `WriteText` from
  count-only to per-item list. Each line:
  `      - {Type}: {Description} ({Location})`.

**Checkpoint**: Running `gaze quality` on the `undertested` and
`helpers` fixtures produces the expected new sections. Human review.

---

## Phase 4: JSON Schema + Tests

**Purpose**: Keep the schema honest and protect against regression.
Depends on Phase 1 (schema changes) and Phase 2 (logic tests).

- [ ] T010 [P] [US1+US2] Update `QualitySchema` in
  `internal/report/schema.go`:
  — Add `"unmapped_reason"` (string, enum of three values, optional)
    to `AssertionMapping` definition.
  — Add `"gap_hints"` (array of string, optional) to
    `ContractCoverage` definition.

- [ ] T011 [P] [US2] Add table-driven unit tests for `hintForEffect`
  in `internal/quality/quality_test.go` (or a new `hints_test.go`):
  verify every `SideEffectType` constant returns a non-empty string.
  Verify specific hints for all P0 and P1 types match expected values.

- [ ] T012 [P] [US1] Add unit tests for `UnmappedReason` population
  in `internal/quality/quality_test.go`: construct `AssertionSite`
  instances at depth 0 and depth 1, call `MapAssertionsToEffects`,
  assert `UnmappedReason` values are as expected.

- [ ] T013 [US1-US4] Update `TestWriteText_Output` in
  `internal/quality/quality_test.go` to cover all four new text
  sections. Add report fixtures with unmapped assertions (with
  reasons), gaps (with hints), discarded returns, and ambiguous
  effects. Assert expected substrings appear in output.

- [ ] T014 [P] [US1+US2] Update JSON schema validation tests in
  `internal/quality/quality_test.go` (`TestWriteJSON_Structure` and
  related) to confirm new fields serialize correctly and the output
  validates against the updated `QualitySchema`.

**Checkpoint**: `go test -race -count=1 -short ./...` passes.

---

## Phase 5: Documentation

- [ ] T015 [P] Update GoDoc on modified exported types and functions:
  — `AssertionMapping`: document `UnmappedReason` field
  — `ContractCoverage`: document `GapHints` field
  — `UnmappedReasonType`: package-level doc + each constant
  — `hintForEffect` in hints.go: full doc comment
  — `ComputeContractCoverage`: mention GapHints postcondition

---

## Dependencies & Execution Order

- **Phase 1** (T001, T002): No dependencies — run in parallel.
- **Phase 2** (T003, T004, T005): All depend on Phase 1.
  - T004 depends only on taxonomy types; can start as soon as T001+T002 compile.
  - T003 depends on T001 (needs UnmappedReasonType constants).
  - T005 depends on T002 + T004.
- **Phase 3** (T006–T009): All depend on Phase 2 completing.
  - T006 depends on T003 (needs UnmappedReason in mappings).
  - T007 depends on T005 (needs GapHints in ContractCoverage).
  - T008, T009: formatter-only, depend on Phase 1 (types) but not Phase 2 logic.
- **Phase 4** (T010–T014): Depends on Phase 3 completing.
- **Phase 5** (T015): Can run in parallel with Phase 4.

## Parallel Opportunities

```
Phase 1: T001 ‖ T002
Phase 2: T004 can start when T001+T002 compile; T003 after T001; T005 after T002+T004
Phase 3: T008, T009 can start when Phase 1 complete; T006 after T003; T007 after T005
Phase 4: T010, T011, T012, T014 can run in parallel after their dependencies
Phase 5: T015 can run alongside Phase 4
```
