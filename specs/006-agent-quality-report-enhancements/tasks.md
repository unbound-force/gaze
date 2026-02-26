# Tasks: Agent-Oriented Quality Report Enhancements

**Input**: `specs/006-agent-quality-report-enhancements/plan.md` and `spec.md`
**Branch**: `006-agent-quality-report-enhancements`

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: Which user story this task belongs to (US1–US4)
- All tasks include exact file paths

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirm the module compiles cleanly on the feature branch before any changes.

- [x] T001 Verify `go build ./...` passes on branch `006-agent-quality-report-enhancements`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Add new taxonomy types and fields to `internal/taxonomy/types.go`. All logic and
formatter phases depend on these definitions existing before they can compile.

**⚠️ CRITICAL**: Phases 3–6 cannot begin until this phase is complete.

- [x] T002 Add `UnmappedReasonType` typed string constant, `UnmappedReasonHelperParam`, `UnmappedReasonInlineCall`, and `UnmappedReasonNoEffectMatch` constants, and `UnmappedReason UnmappedReasonType` field (`json:"unmapped_reason,omitempty"`) on `AssertionMapping` in `internal/taxonomy/types.go`
- [x] T003 Add `GapHints []string` (`json:"gap_hints,omitempty"`) and `DiscardedReturnHints []string` (`json:"discarded_return_hints,omitempty"`) fields to `ContractCoverage` in `internal/taxonomy/types.go`

**Checkpoint**: `go build ./...` passes. Existing tests still pass.

---

## Phase 3: User Story 1 — Unmapped Assertion Reasons (Priority: P1) 🎯 MVP

**Goal**: Every unmapped `AssertionMapping` carries a typed `UnmappedReason` in both JSON and
text output so agents can triage mapping failures (helper body, inline call, or no match)
without guessing.

**Independent Test**: Run `gaze quality` on the `helpers` and `welltested` fixtures; every
unmapped assertion carries a non-empty `unmapped_reason` value matching the known cause.

### Implementation for User Story 1

- [x] T004 [US1] Add `classifyUnmappedReason()` helper and `hasReturnEffects()` predicate to `internal/quality/mapping.go`; populate `UnmappedReason` on each unmapped `AssertionMapping` inside `MapAssertionsToEffects`
- [x] T005 [US1] Expand the unmapped-assertions block in `WriteText` from count-only to per-item list showing `file:line`, assertion type, and `[reason: <value>]` in `internal/quality/report.go`

### Tests for User Story 1

- [x] T006 [US1] Add unit tests for `UnmappedReason` population (depth > 0 → helper_param, depth 0 + empty objToEffectID + return effects → inline_call, otherwise → no_effect_match) in `internal/quality/quality_test.go`
- [x] T007 [US1] Add integration tests asserting text and JSON output for `helpers`/`welltested` fixtures carry correct `unmapped_reason` values in `internal/quality/quality_test.go`

**Checkpoint**: `gaze quality` on `helpers`/`welltested` fixtures shows per-item unmapped list
with reason labels; JSON `unmapped_assertions` elements have `unmapped_reason` set.

---

## Phase 4: User Story 2 — Gap Assertion Hints (Priority: P2)

**Goal**: Every coverage gap in text and JSON output carries a Go assertion code snippet that
tells agents exactly what assertion code to write for the effect type.

**Independent Test**: Run `gaze quality` on the `undertested` fixture; every gap in text output
has an indented `hint:` line; `gap_hints` in JSON has the same length as `gaps`.

### Implementation for User Story 2

- [x] T008 [P] [US2] Create `internal/quality/hints.go` with unexported `hintForEffect(e taxonomy.SideEffect) string` implementing all P0/P1 type-specific hint templates from `plan.md` and a `// assert {Type} side effect of target()` generic fallback for P2–P4
- [x] T009 [US2] Update `ComputeContractCoverage` in `internal/quality/coverage.go` to call `hintForEffect` for each gap and populate `ContractCoverage.GapHints`; enforce `len(GapHints) == len(Gaps)` as postcondition
- [x] T010 [US2] Add indented `        hint: <snippet>` line under each gap item in the gaps section of `WriteText` in `internal/quality/report.go`

### Tests for User Story 2

- [x] T011 [P] [US2] Add table-driven unit tests for `hintForEffect` covering every `SideEffectType` constant in `internal/quality/hints_test.go`; assert non-empty return for every type and exact expected strings for all P0 and P1 types
- [x] T012 [US2] Add tests asserting (1) `len(gap_hints) == len(gaps)` on the `undertested` fixture JSON output and (2) the `gap_hints` key is absent from JSON output for a fixture with zero coverage gaps (verifying `omitempty` behaviour), both in `internal/quality/quality_test.go`

**Checkpoint**: `gaze quality` on `undertested` fixture shows `hint:` line under each gap;
JSON `gap_hints` length equals `gaps` length.

---

## Phase 5: User Story 3 — Discarded Returns in Text Output (Priority: P3)

**Goal**: Text output surfaces a `Discarded returns:` section listing each discarded effect
with type, description, location, and a `hint:` line. JSON carries a parallel
`discarded_return_hints` array with the same length as `discarded_returns`.

**Independent Test**: Run `gaze quality` on `undertested` fixture (contains `_ = store.Set(...)`
patterns); text output includes `Discarded returns:` section with at least one entry and a
`hint:` per entry; JSON `discarded_return_hints` has the same length as `discarded_returns`.

### Implementation for User Story 3

- [x] T013 [US3] Update `quality.Assess` in `internal/quality/quality.go` to call `hintForEffect` for each discarded return and populate `ContractCoverage.DiscardedReturnHints`; enforce `len(DiscardedReturnHints) == len(DiscardedReturns)` as postcondition
- [x] T014 [US3] Add `Discarded returns:` section to `WriteText` in `internal/quality/report.go` rendering `      - {Type}: {Description} ({Location})` and an indented `        hint: <snippet>` line per entry; section omitted when `DiscardedReturns` is empty

### Tests for User Story 3

- [x] T015 [US3] Add test asserting text output for `undertested` fixture includes `Discarded returns:` section with `hint:` lines in `internal/quality/quality_test.go`
- [x] T016 [US3] Add test asserting `len(discarded_return_hints) == len(discarded_returns)` on `undertested` fixture JSON output, and that `discarded_return_hints` is absent from JSON for a fixture with no discarded returns (omitempty), in `internal/quality/quality_test.go`

**Checkpoint**: `gaze quality` on `undertested` shows `Discarded returns:` section; JSON
`discarded_return_hints` is populated and the length-invariant holds.

---

## Phase 6: User Story 4 — Ambiguous Effects Detail in Text Output (Priority: P4)

**Goal**: Ambiguous effects in text output expand from a count (`Ambiguous effects (excluded): 4`)
to a per-item list with type, description, and source location so agents can target GoDoc fixes.

**Independent Test**: Run `gaze quality` on a package with known ambiguous effects; each
ambiguous effect appears individually with type, description, and location — not just a count.

### Implementation for User Story 4

- [x] T017 [US4] Expand the ambiguous-effects block in `WriteText` from count-only to per-item list showing `      - {Type}: {Description} ({Location})` in `internal/quality/report.go`; section omitted when `AmbiguousEffects` is empty

### Tests for User Story 4

- [x] T018 [US4] Add test asserting text output lists each ambiguous effect individually with type and location in `internal/quality/quality_test.go`

**Checkpoint**: Text output for packages with ambiguous effects lists each one individually;
packages with zero ambiguous effects render no ambiguous section (empty-set behavior preserved).

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Keep the JSON schema honest, validate serialization, and complete documentation.

- [x] T019 [P] Update `QualitySchema` in `internal/report/schema.go` to add `"unmapped_reason"` (string, enum of three values, optional) to `AssertionMapping` and `"gap_hints"` and `"discarded_return_hints"` (arrays of string, optional) to `ContractCoverage`
- [x] T020 Update `TestWriteJSON_Structure` and related schema-validation tests in `internal/quality/quality_test.go` to confirm new fields serialize correctly and output validates against the updated `QualitySchema`
- [x] T021 [P] Update GoDoc comments on `UnmappedReasonType` and its three constants, `AssertionMapping.UnmappedReason`, `ContractCoverage.GapHints`, `ContractCoverage.DiscardedReturnHints` in `internal/taxonomy/types.go`; add full doc comment for `hintForEffect` covering the hint table structure and P2–P4 fallback in `internal/quality/hints.go`; update the `ComputeContractCoverage` postcondition note in `internal/quality/coverage.go`
- [x] T022 Run `go test -race -count=1 -short ./...` and confirm all tests pass

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (T001)**: No dependencies — start immediately.
- **Phase 2 (T002, T003)**: Depends on Phase 1. T002 and T003 edit the same file; write them
  in one pass.
- **Phase 3 (T004–T007)**: Depends on T002. T004 (logic) before T005 (formatter). Tests
  T006/T007 depend on T004.
- **Phase 4 (T008–T012)**: Depends on T003. T008 (hints.go) before T009 (coverage.go) before
  T010 (report.go). Tests T011/T012 depend on T008/T009.
- **Phase 5 (T013–T016)**: Depends on T003 and T008. T013 (quality.go) before T014 (report.go).
  Tests T015/T016 depend on T013/T014.
- **Phase 6 (T017–T018)**: Depends on T002 (types must exist). Formatter-only — no new
  pipeline logic.
- **Phase 7 (T019–T022)**: Depends on Phases 3–6 completing.

### User Story Dependencies

- **US1 (P1)**: Unblocked after T002.
- **US2 (P2)**: Unblocked after T003. T008–T010 are independent of US1.
- **US3 (P3)**: Unblocked after T003 AND T008 (needs `hintForEffect` from US2's `hints.go`).
- **US4 (P4)**: Unblocked after T002. Formatter-only; no dependency on US1–US3 logic.

### Parallel Opportunities

```
Phase 2:   T002 → T003   (same file — write sequentially in one edit pass; no [P])
After P2:  T004 (US1) ‖ T008 (US2) ‖ T017 (US4)  — three stories start in parallel
After T008: T013 (US3) can start
Phase 7:   T019 ‖ T021  (different files: schema.go vs types.go+hints.go+coverage.go)
           T020 follows sequentially (quality_test.go — same file as earlier test tasks)
```

---

## Parallel Example: User Story 2

```bash
# After Phase 2 completes, implement US2 in sequence:
Task 1: "Create hintForEffect() in internal/quality/hints.go"          # T008
Task 2: "Populate GapHints in internal/quality/coverage.go"            # T009
Task 3: "Add hint lines in WriteText in internal/quality/report.go"    # T010

# T011 can run in parallel (different file: hints_test.go):
Task [P]: "Table-driven unit tests in internal/quality/hints_test.go"  # T011
# T012 writes to quality_test.go — run after other quality_test.go tasks complete:
Task:      "Length-invariant + zero-gaps omitempty test in internal/quality/quality_test.go"  # T012
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001)
2. Complete Phase 2: Foundational (T002, T003)
3. Complete Phase 3: User Story 1 (T004–T007)
4. **STOP and VALIDATE**: Run `gaze quality` on `helpers`/`welltested`; confirm per-item
   unmapped list with reason labels appears in text and JSON.
5. Schema update for US1 fields (T019 partial) and GoDoc (T021 partial).

### Incremental Delivery

1. Phase 1 + Phase 2 → Foundation ready
2. Phase 3 (US1) → Unmapped reasons visible → Validate independently
3. Phase 4 (US2) → Gap hints visible → Validate independently
4. Phase 5 (US3) → Discarded returns section → Validate independently
5. Phase 6 (US4) → Ambiguous effects list → Validate independently
6. Phase 7 → Schema + tests + docs → Full test suite pass

### Parallel Team Strategy

With multiple contributors after Phase 2 completes:

- Contributor A: User Story 1 (`mapping.go` + unmapped section in `report.go`)
- Contributor B: User Story 2 (`hints.go` + `coverage.go` + hint lines in `report.go`)
- Contributor C: User Story 4 (ambiguous per-item section in `report.go` — formatter only)
- User Story 3 starts after Contributor B finishes `hints.go`

---

## Notes

- [P] tasks operate on different files with no dependency on incomplete tasks
- [Story] label maps each task to its user story for traceability
- US3 depends on US2's `hintForEffect` — plan US3 after T008 is done
- T002 and T003 both modify `internal/taxonomy/types.go` — write in a single edit pass
- All new JSON fields use `omitempty` — no breaking changes for existing consumers
- Verify `go build ./...` passes after each phase before starting the next
