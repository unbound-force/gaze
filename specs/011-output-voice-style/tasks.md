# Tasks: Output Voice & Style Standardization

**Input**: Design documents from `/specs/011-output-voice-style/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, quickstart.md

**Tests**: No automated tests requested. Verification is manual via `/gaze` command in OpenCode (per plan.md).

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Phase 1: Setup (Rebase and Branch Preparation)

**Purpose**: Rebase branch onto current `main` to pick up spec 010 artifacts that need to be deleted

- [ ] T001 Rebase branch onto current `main` by running `git fetch origin && git rebase origin/main` to pick up spec 010 artifacts

---

## Phase 2: Foundational (Spec 010 Cleanup + Output Format Core)

**Purpose**: Delete spec 010 artifacts and rewrite the Output Format section of the gaze-reporter prompt. MUST complete before any mode-specific work.

- [ ] T002 Delete the entire `specs/010-report-voice-refinement/` directory and all 7 files within it (spec.md, plan.md, tasks.md, research.md, data-model.md, quickstart.md, checklists/requirements.md) per FR-014
- [ ] T003 Rewrite the Output Format section (lines 192-235) of `.opencode/agents/gaze-reporter.md` to define the complete voice standard: emoji vocabulary table (10 emojis with semantic roles from data-model.md), section marker assignments (🔍 title, 📊 CRAP, 🧪 quality, 🏷️ classification, 🏥 health), grade-to-emoji severity mapping (🟢 B+ and above, 🟡 B through C, 🔴 C- and below, ⚪ neutral), tone anti-pattern bans (exclamation marks, slang, puns, first-person pronouns per FR-004), no-pedagogy rule (do not explain what CRAP scores, quadrants, or coverage metrics mean per FR-005), title format (`🔍 Gaze Full Quality Report`), metadata format (`Project: <path> · Branch: <branch>` + `Gaze Version: <ver> · Go: <ver> · Date: <date>` per FR-010), warning callout format (`> ⚠️ ...` per FR-013), section separator rules (`---` per FR-011), section omission rule (omit silently per FR-006), and interpretation limit (one sentence per table, max 25 words)
- [ ] T004 Rewrite the agent description (lines 20-21) of `.opencode/agents/gaze-reporter.md` to replace "clinical diagnostic summaries — factual, terse, and emoji-free" with "fun, approachable quality summaries with emoji section markers and severity indicators"

**Checkpoint**: Output Format section and agent description now define the complete voice standard. Mode-specific work can begin.

---

## Phase 3: User Story 1 - Scanning a Full Quality Report (Priority: P1) MVP

**Goal**: Full-mode report output uses emoji section markers, severity indicators, letter grades, and conversational tone matching the canonical example.

**Independent Test**: Run `/gaze` in OpenCode on a Go project and verify the full report has emoji-prefixed section headers, colored circle severity emojis, letter grades in the scorecard, and conversational interpretive sentences.

### Implementation for User Story 1

- [ ] T005 [US1] Rewrite the Full Mode section (lines 127-191) of `.opencode/agents/gaze-reporter.md`: replace the Overall Health Assessment structure (remove Risk Matrix table and word-based Overall Grade table), replace with Summary Scorecard using letter grades (A through F) paired with colored circle emojis per the grade-to-emoji mapping in FR-002/FR-003, and replace "Bottom line:" closing paragraph with a conversational closing sentence
- [ ] T006 [US1] Rewrite the CRAP Mode section quadrant labels (lines 81-91) of `.opencode/agents/gaze-reporter.md`: replace plain-text quadrant labels with emoji-prefixed labels (🟢 Q1 — Safe, 🟡 Q2 — Complex But Tested, ⚪ Q3 — Needs Tests, 🔴 Q4 — Dangerous) and add the GazeCRAPload conversational interpretation line per FR-012
- [ ] T007 [US1] Rewrite the Example Output section (lines 239-306) of `.opencode/agents/gaze-reporter.md`: replace the clinical example with the canonical example from the spec (spec.md lines 17-73), adapting function names and numbers to be clearly fictional while preserving the exact structural format (emoji-prefixed headers, metadata line with centered dots, markdown tables, severity-colored grades, numbered recommendations with emoji prefixes, warning callouts with ⚠️ blockquote)

**Checkpoint**: Full-mode report should now produce output matching the canonical example structure. SC-001, SC-002, SC-005, SC-006, SC-007 verifiable.

---

## Phase 4: User Story 2 - CRAP-Only Report Consistency (Priority: P2)

**Goal**: CRAP-only mode uses the same emoji markers, severity indicators, and tone as the full report.

**Independent Test**: Run `/gaze crap ./...` in OpenCode and verify the report title is "🔍 Gaze CRAP Report", CRAP summary uses "📊", and quadrant rows use colored circle emojis.

### Implementation for User Story 2

- [ ] T008 [US2] Update the CRAP Mode section of `.opencode/agents/gaze-reporter.md` to specify that the CRAP-only report title MUST be "🔍 Gaze CRAP Report" (not "🔍 Gaze Full Quality Report"), the metadata line MUST follow the same centered-dot format as the full report per FR-010, and the section header MUST be "📊 CRAP Summary" per FR-001

**Checkpoint**: CRAP-only mode produces voice-consistent output. SC-003 partially verifiable.

---

## Phase 5: User Story 4 - Recommendations with Severity Markers (Priority: P2)

**Goal**: Prioritized recommendations are numbered and prefixed with severity emojis (🔴/🟡/🟢).

**Independent Test**: Run `/gaze` in OpenCode and verify that each recommendation line starts with a numbered index followed by a colored circle emoji matching its severity.

### Implementation for User Story 4

- [ ] T009 [US4] Update the Prioritized Recommendations subsection within the Full Mode section of `.opencode/agents/gaze-reporter.md` to specify that each recommendation MUST be numbered (1., 2., 3...) and prefixed with a severity emoji: 🔴 for critical issues (zero-coverage functions, Q4 Dangerous items), 🟡 for moderate issues (decomposition opportunities, coverage gaps), 🟢 for improvement opportunities (optional analysis runs). Default to 🟡 when severity is unclear per FR-007 and edge case specification

**Checkpoint**: Recommendations display severity emojis. SC-002 fully verifiable for recommendations.

---

## Phase 6: User Story 3 - Quality-Only Report Consistency (Priority: P3)

**Goal**: Quality-only mode uses the same voice standard as the full report.

**Independent Test**: Run `/gaze quality ./...` in OpenCode and verify the report uses "🧪 Quality Summary" as its section header with consistent tone.

### Implementation for User Story 3

- [ ] T010 [US3] Update the Quality Mode section of `.opencode/agents/gaze-reporter.md` to specify that the quality-only report title MUST be "🔍 Gaze Quality Report", the metadata line MUST follow the same centered-dot format per FR-010, and the section header MUST be "🧪 Quality Summary" per FR-001. Warning callouts MUST use the `> ⚠️ ...` blockquote format per FR-013

**Checkpoint**: Quality-only mode produces voice-consistent output. SC-003 fully verifiable.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Sync files, update documentation, and verify end-to-end consistency

- [ ] T011 Copy the completed `.opencode/agents/gaze-reporter.md` to `internal/scaffold/assets/agents/gaze-reporter.md` to maintain byte-identity (both files MUST be identical per plan.md constraints)
- [ ] T012 [P] Update `AGENTS.md` spec listing (around line 90): replace `010-report-voice-refinement/` entry with `011-output-voice-style/   # spec.md, plan.md, tasks.md, research.md, data-model.md, quickstart.md, checklists/`
- [ ] T013 [P] Update `AGENTS.md` Recent Changes section (around line 240): replace the spec 010 entry with `011-output-voice-style: Rewrote gaze-reporter agent prompt for fun, emoji-rich output — emoji section markers (🔍📊🧪🏷️🏥), colored circle severity indicators (🟢🟡🔴⚪), letter grades with emoji, severity-prefixed recommendations, tone anti-pattern bans, canonical example output`
- [ ] T014 Verify the final `.opencode/agents/gaze-reporter.md` prompt contains no references to "clinical", "emoji-free", "Poor/Fair/Good/Strong/Excellent" word-based grades, or "Bottom line:" closing format from spec 010
- [ ] T015 Run quickstart.md verification checklist: confirm the prompt satisfies all 6 verification items (emoji-prefixed headers, severity emojis, letter grades, severity-prefixed recommendations, no banned anti-patterns, silent section omission)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Setup (rebase must complete before spec 010 files exist on branch to delete)
- **User Stories (Phases 3-6)**: All depend on Foundational phase completion (Output Format section must be written first)
  - US1 (Phase 3) should be completed first as the MVP — other stories depend on the full-mode structure as a reference
  - US2 (Phase 4), US4 (Phase 5) can proceed after US1
  - US3 (Phase 6) can proceed after US1
- **Polish (Phase 7)**: Depends on ALL user story phases being complete (the prompt must be fully written before syncing the scaffolded copy)

### User Story Dependencies

- **User Story 1 (P1)**: Depends on Foundational (Phase 2). This is the MVP — the full-mode rewrite establishes the voice patterns that other stories reference
- **User Story 2 (P2)**: Depends on US1 completion (CRAP-only mode references the quadrant labels and structure established in the full-mode rewrite)
- **User Story 4 (P2)**: Can start after US1 completion (recommendations are within the Full Mode section)
- **User Story 3 (P3)**: Depends on Foundational (Phase 2). Independently testable but references voice conventions from US1

### Within Each User Story

- Each task modifies a specific section of `.opencode/agents/gaze-reporter.md`
- Tasks within a story are sequential (they modify the same file)
- Tasks across stories CAN be parallelized if they modify non-overlapping sections, but sequential execution is recommended to avoid merge conflicts in a single file

### Parallel Opportunities

- T012 and T013 can run in parallel (different sections of AGENTS.md)
- T002 can run in parallel with T003/T004 (different files: spec 010 deletion vs prompt rewrite)
- User stories target different sections of the same file, so parallelism is limited

---

## Parallel Example: Foundational Phase

```bash
# T002 can run in parallel with T003/T004 (different files):
Task: "T002 Delete specs/010-report-voice-refinement/ directory"

# T003 and T004 modify the same file — run sequentially:
Task: "T003 Rewrite Output Format section of .opencode/agents/gaze-reporter.md"
Task: "T004 Rewrite agent description of .opencode/agents/gaze-reporter.md"
```

## Parallel Example: Polish Phase

```bash
# These can run in parallel (different sections of AGENTS.md):
Task: "T012 Update AGENTS.md spec listing"
Task: "T013 Update AGENTS.md Recent Changes"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (rebase)
2. Complete Phase 2: Foundational (delete spec 010, write Output Format + agent description)
3. Complete Phase 3: User Story 1 (full-mode rewrite)
4. **STOP and VALIDATE**: Run `/gaze` and verify output matches canonical example
5. If satisfactory, proceed to remaining stories

### Incremental Delivery

1. Setup + Foundational → Voice standard established
2. Add US1 (full mode) → Test independently → MVP complete
3. Add US2 (CRAP mode) + US4 (recommendations) → Test independently
4. Add US3 (quality mode) → Test independently → All modes covered
5. Polish → Sync scaffolded copy, update AGENTS.md → Feature complete

---

## Notes

- Phase numbering in this file is finer-grained than plan.md (7 phases vs 4). Plan.md phases map as: Plan Phase 1 → Tasks Phases 1-2, Plan Phase 2 → Tasks Phases 3-6, Plan Phase 3 → Tasks T011, Plan Phase 4 → Tasks T014-T015
- All tasks modify markdown files only — no Go code changes
- The prompt file `.opencode/agents/gaze-reporter.md` is the primary artifact — most tasks target different sections of this single file
- The scaffolded copy at `internal/scaffold/assets/agents/gaze-reporter.md` MUST be an exact byte-identical copy of the active prompt (T011)
- Commit after each phase or logical task group
- Stop at any checkpoint to validate the story independently
