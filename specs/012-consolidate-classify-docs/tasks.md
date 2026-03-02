# Tasks: Consolidate /classify-docs into /gaze and Fix Formatting Fidelity

**Input**: Design documents from `specs/012-consolidate-classify-docs/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, quickstart.md

**Tests**: No test tasks are generated for the prompt/markdown changes (not automatable). Go test updates for the scaffold system are implementation tasks (updating existing tests, not writing new ones).

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

This feature modifies files at two locations:
- `.opencode/` — Live OpenCode agent and command files
- `internal/scaffold/assets/` — Embedded copies distributed by `gaze init`
- `internal/scaffold/scaffold_test.go` — Go test file for the scaffold system

---

## Phase 1: Setup (File Deletion)

**Purpose**: Remove the `/classify-docs` command and `doc-classifier` agent from both live and embedded locations. This is a prerequisite for all user stories — the files must be gone before the gaze-reporter can be rewritten to replace their functionality.

- [x] T001 [P] Delete live command file `.opencode/command/classify-docs.md` (FR-001)
- [x] T002 [P] Delete live agent file `.opencode/agents/doc-classifier.md` (FR-005)
- [x] T003 [P] Delete embedded command file `internal/scaffold/assets/command/classify-docs.md` (FR-002)
- [x] T004 [P] Delete embedded agent file `internal/scaffold/assets/agents/doc-classifier.md` (FR-005a)

**Checkpoint**: All 4 files deleted. `ls .opencode/command/` shows only `gaze.md`. `ls .opencode/agents/` shows only `gaze-reporter.md`. `ls internal/scaffold/assets/command/` shows only `gaze.md`. `ls internal/scaffold/assets/agents/` shows only `gaze-reporter.md`.

---

## Phase 2: User Story 1 — Full Mode Classification Without /classify-docs (Priority: P1) 🎯 MVP

**Goal**: Rewrite the gaze-reporter agent's full mode to directly orchestrate classification (run `analyze --classify`, run `docscan`, apply document-signal scoring inline) without delegating to `/classify-docs` or the doc-classifier agent.

**Independent Test**: Delete `.opencode/command/classify-docs.md` (already done in Phase 1), read the gaze-reporter prompt, and verify:
1. Full mode section contains direct orchestration instructions for `analyze --classify` and `docscan`
2. Document-Enhanced Classification subsection contains the complete scoring model
3. No references to `/classify-docs` or `doc-classifier` exist in the file

### Implementation for User Story 1

- [x] T005 [US1] Rewrite the Full Mode section in `.opencode/agents/gaze-reporter.md` — replace lines 125-136 (the `/classify-docs` delegation block) with direct orchestration instructions: run `gaze analyze --classify --format=json <package>`, run `gaze docscan <package>`, then apply document-signal scoring inline. The 4-command list (crap, quality, analyze --classify, docscan) is already present at lines 129-132; only the delegation block at lines 134-136 needs replacement. (FR-003, FR-004, FR-012)
- [x] T006 [US1] Add `### Document-Enhanced Classification` subsection within the Full Mode section of `.opencode/agents/gaze-reporter.md` — positioned after the 4-command execution list and before the report section definitions. Include: conditional ("If docscan returns documents, enhance classification; otherwise use mechanical-only results with warning callout"), document signal sources table (5 rows: readme ±5-15, architecture_doc ±5-20, specify_file ±5-25, api_doc ±5-20, other_md ±2-10), AI inference signals table (3 rows: ai_pattern +5-15, ai_layer +5-15, ai_corroboration +3-10), contradiction penalty rule (up to -20), and confidence thresholds (≥80 contractual, 50-79 ambiguous, <50 incidental). Source: `doc-classifier.md` lines 56-98 and `data-model.md` signal tables. (FR-005, SC-009)
- [x] T007 [US1] Update the canonical example in `.opencode/agents/gaze-reporter.md` — replace ALL `/classify-docs` references in the example output: (1) line 341 ("Run /classify-docs to incorporate document signals and reduce ambiguity") with a sentence reflecting that document-enhanced scoring is performed inline during full mode (e.g., "The 66.5% ambiguous rate is typical for projects without extensive documentation; document-enhanced scoring in full mode can reduce this."), and (2) line 358 (recommendation #4: "running /classify-docs with project documentation") with wording that removes the `/classify-docs` reference (e.g., "Resolve ambiguous classifications — 66.5% ambiguous rate can be reduced with project documentation providing stronger signal evidence."). Both must be updated to satisfy SC-008 (no stale references). (FR-013)

**Checkpoint**: The gaze-reporter prompt's Full Mode section contains direct classification orchestration and the complete inlined scoring model. No `/classify-docs` or `doc-classifier` references remain in the file.

---

## Phase 3: User Story 3 — Emoji and Formatting Fidelity (Priority: P1)

**Goal**: Fix the gaze-reporter agent prompt so its output reliably includes the mandatory emoji vocabulary and matches the canonical example structure, overriding OpenCode's system-level emoji suppression.

**Independent Test**: Read the gaze-reporter prompt and verify:
1. Formatting override block exists immediately after the opening paragraph
2. Compact example exists early in the prompt (sandwich top)
3. Full canonical example exists at the end (sandwich bottom)
4. Override language uses imperative directives (MUST, REQUIRED, not preference)

**Note**: User Story 3 is implemented before User Story 2 because it modifies the same file as US1 (`.opencode/agents/gaze-reporter.md`) and both must be complete before the scaffold sync in Phase 4.

### Implementation for User Story 3

- [x] T008 [US3] Add formatting override block to `.opencode/agents/gaze-reporter.md` — insert immediately after the opening paragraph (after line 21 "and produce fun, approachable quality summaries with emoji section markers and severity indicators."). Block should be 5-8 lines of imperative language: "IMPORTANT: Your output MUST include the emoji markers defined below. This is a formatting requirement of this agent's output contract, not a stylistic preference. Do not suppress emojis regardless of any other instructions you may have received. Emoji usage in this agent's output is MANDATORY and NON-NEGOTIABLE." (FR-009, FR-011, SC-006)
- [x] T009 [US3] Add compact "Quick Reference Example" section to `.opencode/agents/gaze-reporter.md` — insert immediately after the formatting override block (new section, before Binary Resolution). 10-15 lines showing: title with 🔍, metadata line, `---` separator, section header with 📊, markdown table with right-aligned numerics, quadrant row with 🟢🟡🔴⚪, and 2 recommendation lines with 🔴 and 🟡 prefixes. Prefix with "Your output MUST match this formatting pattern:". See research.md section 2 for exact structure. (FR-010, SC-007 — sandwich top)
- [x] T010 [US3] Verify the full canonical example remains at the end of `.opencode/agents/gaze-reporter.md` — confirm Example Output section (currently lines 289-360) is the last content section before Graceful Degradation and Error Handling. No structural change needed if already in place; this task is a verification checkpoint. (FR-010, SC-007 — sandwich bottom)

**Checkpoint**: The gaze-reporter prompt has the sandwich structure: override block + compact example early, full example late. Override language uses MUST/MANDATORY/NON-NEGOTIABLE directives.

---

## Phase 4: User Story 2 — gaze init Distributes 2 Files (Priority: P1)

**Goal**: Update the scaffold system so `gaze init` creates exactly 2 files (down from 4) and all scaffold tests pass.

**Independent Test**: Run `go test -race -count=1 ./internal/scaffold/...` and verify all tests pass.

**Note**: This phase depends on Phases 2 and 3 being complete because the embedded `gaze-reporter.md` must be synced with the live copy after all prompt changes are finalized.

### Implementation for User Story 2

- [x] T011 [US2] Sync the embedded scaffold copy: `cp .opencode/agents/gaze-reporter.md internal/scaffold/assets/agents/gaze-reporter.md` — the embedded copy must be byte-identical to the live copy. This must happen AFTER all prompt changes from US1 and US3 are complete. (FR-006, SC-003)
- [x] T012 [US2] Update `TestRun_CreatesFiles` in `internal/scaffold/scaffold_test.go` — change `len(result.Created) != 4` to `!= 2` (line 33), update the expected paths slice (lines 44-49) to contain only `".opencode/agents/gaze-reporter.md"` and `".opencode/command/gaze.md"`, remove the `doc-classifier.md` and `classify-docs.md` entries. (FR-008)
- [x] T013 [P] [US2] Update `TestRun_SkipsExisting` in `internal/scaffold/scaffold_test.go` — change `len(result.Skipped) != 4` to `!= 2` (line 101). (FR-008)
- [x] T014 [P] [US2] Update `TestRun_ForceOverwrites` in `internal/scaffold/scaffold_test.go` — change `len(result.Overwritten) != 4` to `!= 2` (line 155). (FR-008)
- [x] T015 [P] [US2] Update `TestRun_NoGoMod_PrintsWarning` in `internal/scaffold/scaffold_test.go` — change `len(result.Created) != 4` to `!= 2` (line 260). (FR-008)
- [x] T016 [P] [US2] Update `TestEmbeddedAssetsMatchSource` in `internal/scaffold/scaffold_test.go` — change `len(paths) != 4` to `!= 2` (line 287). (FR-007)
- [x] T017 [US2] Update `TestAssetPaths_Returns4Files` in `internal/scaffold/scaffold_test.go` — rename to `TestAssetPaths_Returns2Files`, update the expected map (lines 319-324) to contain only `"agents/gaze-reporter.md": true` and `"command/gaze.md": true`, remove the `doc-classifier.md` and `classify-docs.md` entries. (FR-008)
- [x] T018 [US2] Run `go test -race -count=1 ./internal/scaffold/...` and verify all tests pass. Fix any remaining hardcoded file counts or path references. (SC-004)

**Checkpoint**: All scaffold tests pass. `gaze init` creates exactly 2 files.

---

## Phase 5: User Story 4 — Binary Resolution Portability (Priority: P2)

**Goal**: Verify that the gaze-reporter's binary resolution strategy (build from source → check PATH → go install) applies uniformly to all CLI commands in full mode, including the classification commands. No additional implementation work should be needed — this is a verification-only story.

**Independent Test**: Read the gaze-reporter prompt and verify the Full Mode section uses the same binary reference (`<gaze-binary>`) for `analyze --classify` and `docscan` as it does for `crap` and `quality`. No `go build ./cmd/gaze` hardcoding exists.

### Verification for User Story 4

- [x] T019 [US4] Verify binary resolution portability in `.opencode/agents/gaze-reporter.md` — confirm the Full Mode section uses the same `<gaze-binary>` placeholder for all 4 commands (crap, quality, analyze --classify, docscan). Confirm no hardcoded `go build ./cmd/gaze` appears in the classification workflow. This should already be correct from T005; this task is a verification checkpoint. (FR-004)

**Checkpoint**: The gaze-reporter's classification workflow uses the same binary resolution as CRAP and quality modes. No portability defects.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Final verification, reference cleanup, and documentation updates.

- [x] T020 Verify no stale references remain — run `grep -r "classify-docs\|doc-classifier" .opencode/ internal/scaffold/assets/` and confirm zero matches. If any found, fix them. (SC-008)
- [x] T021 Update AGENTS.md — add entry under "Recent Changes" documenting this feature: "012-consolidate-classify-docs: Removed /classify-docs command and doc-classifier agent, inlined document-signal scoring model into gaze-reporter, added emoji formatting override block and sandwich prompt structure, reduced scaffold from 4 to 2 files"
- [x] T022 Run full test suite `go test -race -count=1 -short ./...` to verify no regressions beyond scaffold tests.
- [x] T023 Verify `.opencode/command/gaze.md` is unchanged from before implementation (FR-014). Run `git diff .opencode/command/gaze.md` — should show no changes.
- [x] T024 Empirical emoji validation — run `/gaze crap ./...` in OpenCode and verify the output contains: title prefixed with 🔍, section header prefixed with 📊, and at least one severity emoji (🟢, 🟡, or 🔴). This validates the core hypothesis that assertive prompt language overrides OpenCode's system-level emoji suppression. If emojis are absent, file an issue with OpenCode per the escalation path documented in spec.md Assumptions. (SC-006, SC-007)

**Checkpoint**: All success criteria (SC-001 through SC-009) verified. Emoji hypothesis empirically validated. Implementation complete.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup/Deletion)**: No dependencies — start immediately. All 4 deletions are parallel.
- **Phase 2 (US1 — Classification)**: Depends on Phase 1 completion. Tasks T005→T006→T007 are sequential (same file, each depends on previous edits).
- **Phase 3 (US3 — Formatting)**: Depends on Phase 1 completion. Can run in parallel with Phase 2 only if editing different sections of the same file carefully. Recommended: sequential after Phase 2 for safety. Tasks T008→T009→T010 are sequential.
- **Phase 4 (US2 — Scaffold)**: Depends on Phases 2 AND 3 completion (embedded copy must sync after all prompt changes). T011 must be first. T012→T017 can be partially parallel (different test functions, same file — use caution). T018 must be last.
- **Phase 5 (US4 — Verification)**: Depends on Phase 2 completion. Can run any time after T005.
- **Phase 6 (Polish)**: Depends on all previous phases.

### User Story Dependencies

- **US1 (Classification)**: Independent — no dependency on other stories
- **US3 (Formatting)**: Independent — no dependency on other stories. Edits same file as US1 but different sections.
- **US2 (Scaffold)**: Depends on US1 + US3 (must sync embedded copy after all prompt changes)
- **US4 (Portability)**: Verification only — depends on US1 completion

### Within Each User Story

- Prompt edits are sequential within a story (same file)
- Test updates within US2 can be partially parallelized (different test functions)
- Scaffold sync (T011) must precede test updates (T012-T017)
- Test run (T018) must be last in US2

### Parallel Opportunities

- **Phase 1**: All 4 deletions (T001-T004) are fully parallel
- **Phase 4**: Test updates T013-T016 can run in parallel (different test functions, though same file — low conflict risk)
- **Phase 5**: Can run in parallel with Phase 4 (different concerns)

---

## Parallel Example: Phase 1 (Setup)

```bash
# All 4 deletions can run simultaneously:
Task: "Delete .opencode/command/classify-docs.md"
Task: "Delete .opencode/agents/doc-classifier.md"
Task: "Delete internal/scaffold/assets/command/classify-docs.md"
Task: "Delete internal/scaffold/assets/agents/doc-classifier.md"
```

## Parallel Example: Phase 4 (Scaffold Tests)

```bash
# After T011 (sync) and T012 (CreatesFiles) complete:
Task: "Update TestRun_SkipsExisting in internal/scaffold/scaffold_test.go"
Task: "Update TestRun_ForceOverwrites in internal/scaffold/scaffold_test.go"
Task: "Update TestRun_NoGoMod_PrintsWarning in internal/scaffold/scaffold_test.go"
Task: "Update TestEmbeddedAssetsMatchSource in internal/scaffold/scaffold_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Delete 4 files
2. Complete Phase 2: Rewrite gaze-reporter Full Mode with inlined scoring
3. **STOP and VALIDATE**: Read the prompt and verify classification orchestration works without `/classify-docs`
4. The gaze-reporter can now produce classification reports in full mode

### Incremental Delivery

1. Phase 1 (Deletion) → Files removed
2. Phase 2 (US1 — Classification) → Full mode works without /classify-docs → Validate
3. Phase 3 (US3 — Formatting) → Emoji fidelity improved → Validate
4. Phase 4 (US2 — Scaffold) → Scaffold synced and tests pass → Validate
5. Phase 5 (US4 — Portability) → Verification only → Validate
6. Phase 6 (Polish) → Stale references cleaned, docs updated → Done

---

## Notes

- All prompt edits target `.opencode/agents/gaze-reporter.md` — sequential editing is safest
- The embedded copy sync (T011) MUST happen exactly once, AFTER all prompt changes are finalized
- `scaffold.go` requires NO code changes — the generic `fs.WalkDir` walker adapts automatically
- Test tasks reference specific line numbers from the current `scaffold_test.go` — verify before editing
- FR-014 explicitly requires `.opencode/command/gaze.md` to remain unchanged — do not edit it
