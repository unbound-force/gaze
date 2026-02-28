# Tasks: Report Voice Refinement

**Input**: Design documents from `/specs/010-report-voice-refinement/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, quickstart.md

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Setup

**Purpose**: Capture current state before any changes

- [ ] T001 Read the current `.opencode/agents/gaze-reporter.md` in full and confirm it matches the structure described in `data-model.md` (YAML frontmatter, Binary Resolution, Mode Parsing, CRAP Mode, Quality Mode, Full Mode, Output Format, Graceful Degradation, Error Handling)
- [ ] T002 Verify that `.opencode/agents/gaze-reporter.md` and `internal/scaffold/assets/agents/gaze-reporter.md` are currently byte-identical (baseline for SC-006)

**Checkpoint**: Baseline confirmed — prompt file structure understood, both copies verified identical.

---

## Phase 2: Foundational

**Purpose**: No foundational prerequisites beyond Phase 1. All user stories modify sections of the same file but are organized as sequential edits to avoid conflicts.

**Checkpoint**: Phase 1 complete — user story implementation can begin.

---

## Phase 3: User Story 1 + User Story 2 — Emoji-Free Clinical Tone and Flat Layout (Priority: P1) MVP

**Goal**: Rewrite the Output Format section to establish the clinical tone directive, emoji prohibition, title/metadata format, section omission rule, and terse interpretation rule. US1 and US2 are combined because they modify the same section and are both P1 — the tone and structure are inseparable.

**Independent Test**: Run `/gaze` on the gaze project itself and verify: zero emoji characters, plain-text title in format `Gaze Health Report — <project-name>`, single-line metadata, no multi-paragraph interpretations, omitted sections produce no placeholder headers.

- [ ] T003 [US1] Rewrite the `## Output Format` section (lines 158-166) in `.opencode/agents/gaze-reporter.md` with the comprehensive formatting specification: (a) tone directive — clinical, matter-of-fact voice; no emojis; no pedagogical explanations; every sentence conveys data or an actionable observation (FR-001, FR-002, FR-013); (b) title format — `Gaze Health Report — <project-name>` (FR-003); (c) metadata format — `Package: <pattern> | Date: <date>` (FR-004); (d) section omission rule — omit sections entirely when data is unavailable; no placeholder headers, blockquotes, or "N/A" content; append a single-line note at the end listing unavailable analyses (FR-005); (e) interpretation rule — single terse sentence after each data table, max 25 words (FR-013)
- [ ] T004 [US1] Update the intro sentence after `# Gaze Reporter Agent` heading in `.opencode/agents/gaze-reporter.md` to reinforce the clinical tone: replace "produce clear, human-readable summaries" with "produce concise, clinical diagnostic summaries — factual, terse, and emoji-free"
- [ ] T005 [US2] Update `## Graceful Degradation` section in `.opencode/agents/gaze-reporter.md`: replace "Note which sections are missing and why" with "Append a single-line note at the end of the report listing which analyses were unavailable." Remove any instruction that could produce blockquotes or warning banners for missing data (FR-005)
- [ ] T006 [US2] Update `## Quality Mode` section in `.opencode/agents/gaze-reporter.md`: remove the blockquote fallback instruction (lines 97-99) and replace with "If quality analysis is not available, omit this section entirely — do not render any header, blockquote, or placeholder" (FR-005)

**Checkpoint**: Output Format rewritten with tone directive. Quality Mode and Graceful Degradation aligned with omit-over-placeholder rule. Run `/gaze` to verify emoji-free output with flat structure.

---

## Phase 4: User Story 3 — Word-Based Grades and Right-Aligned Numerics (Priority: P2)

**Goal**: Add table formatting rules (right-aligned numerics, concise labels, CRAPload percentage) and word-based grade scale instructions to the prompt.

**Independent Test**: Run `/gaze crap` and verify numeric columns use right-alignment, metric labels are concise, CRAPload shows count and percentage, and grades use word-based ratings.

- [ ] T007 [US3] Add table formatting rules to the `## Output Format` section in `.opencode/agents/gaze-reporter.md`: (a) right-aligned numeric columns using `|------:|` syntax for CRAP, Complexity, Coverage, and all other numeric data (FR-007); (b) concise metric labels — "Functions analyzed" not "Total functions analyzed", "Avg complexity" not "Average complexity" (FR-009); (c) CRAPload format — always include count and percentage: e.g., "40 functions (29.2%)" (FR-008); (d) use "Location" not "File" for the source reference column header (per research.md R1)
- [ ] T008 [US3] Update `## CRAP Mode` section in `.opencode/agents/gaze-reporter.md`: (a) replace "Total functions analyzed" with "Functions analyzed" (FR-009); (b) add CRAPload percentage format instruction (FR-008); (c) replace quadrant label instructions to use plain-text descriptors — "Q1 — Safe", "Q3 — Simple But Underspecified", "Q4 — Dangerous" — no colored circles or emoji (FR-002); (d) add instruction for terse summary sentence at end of CRAP-only reports
- [ ] T009 [US3] Add word-based grade scale to `## Full Mode` section in `.opencode/agents/gaze-reporter.md`: instruct the agent to use an Overall Grade table with columns Aspect, Rating, Notes where Rating values are exclusively from the set {Poor, Fair, Good, Strong, Excellent} (FR-006). Include the grade scale criteria from data-model.md so the agent can assign grades consistently

**Checkpoint**: Tables use right-aligned numerics, concise labels, CRAPload percentage, and word-based grades. Run `/gaze crap` to verify.

---

## Phase 5: User Story 4 — Risk Matrix and Bottom Line (Priority: P2)

**Goal**: Add risk matrix table specification and "Bottom line:" closing paragraph to the Full Mode section.

**Independent Test**: Run `/gaze` (full mode) and verify the Overall Health Assessment contains a Risk Matrix table as its first element and the report ends with a "Bottom line:" paragraph.

- [ ] T010 [US4] Add Risk Matrix specification to the `## Full Mode` Overall Health Assessment subsection in `.opencode/agents/gaze-reporter.md`: instruct the agent to produce a risk matrix table as the FIRST table in the health assessment with columns Priority (centered, numeric), Function, Risk (Critical/High/Medium/Low), Why (data-packed clause, max 20 words) (FR-010, FR-014). Include the risk level criteria from data-model.md so the agent can assign risk levels consistently
- [ ] T011 [US4] Add "Bottom line:" specification to `## Full Mode` in `.opencode/agents/gaze-reporter.md`: instruct the agent to end every full-mode report with a plain-text paragraph beginning "Bottom line:" containing 1-3 sentences — a positive acknowledgment of strengths, the key risk, and the single most important next action (FR-011)
- [ ] T012 [US4] Update the section ordering in `## Full Mode` in `.opencode/agents/gaze-reporter.md` to match research.md R4: CRAP Summary → GazeCRAP Quadrants (if available) → Quality Summary (if available) → Classification Summary (if available) → Overall Health Assessment (Risk Matrix → Recommendations → Overall Grade → Bottom line)

**Checkpoint**: Full-mode report has risk matrix as first assessment table and ends with "Bottom line:" paragraph. Run `/gaze` to verify.

---

## Phase 6: User Story 5 — Recommendations as Action Sentences (Priority: P3)

**Goal**: Add recommendation formatting rules to the prompt.

**Independent Test**: Run `/gaze` on a project with CRAPload > 0 and verify recommendations are numbered action sentences without emoji prefixes, each referencing concrete metrics.

- [ ] T013 [US5] Add recommendation formatting rules to the `## Full Mode` Overall Health Assessment subsection in `.opencode/agents/gaze-reporter.md`: (a) recommendations as numbered action sentences starting with an action verb (Refactor, Add, Break up, Write, Consider) followed by a specific function or package name (FR-012); (b) each recommendation includes at least one concrete metric from the report data (e.g., "complexity 38 → target <15", "0% coverage") (FR-012); (c) plain numbers only (1., 2., 3.) — no emoji prefixes or colored indicators (FR-012)

**Checkpoint**: Recommendations follow action-sentence format with metrics. Run `/gaze` to verify.

---

## Phase 7: Example Output and Synchronization

**Purpose**: Add the concrete example output and synchronize the scaffold copy.

- [ ] T014 Add a new `## Example Output` section in `.opencode/agents/gaze-reporter.md` after the `## Output Format` section and before `## Graceful Degradation`. The example MUST show: (a) title line: `Gaze Health Report — example-project`; (b) metadata line: `Package: ./... | Date: Sat Feb 28, 2026`; (c) CRAP Summary table with right-aligned numerics and concise labels; (d) one-sentence interpretation; (e) GazeCRAP quadrant table with plain-text labels; (f) Risk Matrix table with Priority/Function/Risk/Why columns; (g) Overall Grade table with word-based ratings (Aspect/Rating/Notes); (h) one numbered recommendation as action sentence; (i) "Bottom line:" closing paragraph. Use realistic but fictional data (FR-016, SC-007, SC-008)
- [ ] T015 Copy the completed `.opencode/agents/gaze-reporter.md` to `internal/scaffold/assets/agents/gaze-reporter.md` byte-for-byte. Verify byte-identity using diff or checksum (FR-015, SC-006)
- [ ] T016 Assess documentation impact: check if `AGENTS.md` needs updates for spec 010 in Active Technologies and Recent Changes sections. Update if needed per AGENTS.md Documentation Validation Gate requirements

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — MUST run first to confirm baseline
- **Foundational (Phase 2)**: Depends on Phase 1 — verification gate only
- **US1+US2 (Phase 3)**: Depends on Phase 2 completion — establishes tone and structure
- **US3 (Phase 4)**: Depends on Phase 3 — builds on the Output Format section written in US1
- **US4 (Phase 5)**: Depends on Phase 3 — builds on the Full Mode section structure
- **US5 (Phase 6)**: Depends on Phase 5 — recommendations live inside the Full Mode health assessment
- **Example + Sync (Phase 7)**: Depends on ALL previous phases — example must reflect final format; sync is always last

### User Story Dependencies

- **US1 + US2 (P1)**: Combined — both modify Output Format and mode sections. MUST complete before US3/US4/US5
- **US3 (P2)**: Depends on US1+US2 (needs the Output Format section to exist before adding table rules)
- **US4 (P2)**: Depends on US1+US2 (needs the Full Mode section structure). Can run in parallel with US3 since they modify different subsections
- **US5 (P3)**: Depends on US4 (recommendations live inside the health assessment section added by US4)

### Within Each User Story

- All tasks within a user story are sequential (same file, same or adjacent sections)

### Parallel Opportunities

- **US3 (Phase 4) + US4 (Phase 5)**: Could run in parallel if implemented carefully since US3 modifies Output Format + CRAP Mode while US4 modifies Full Mode. However, both touch Full Mode, so sequential is safer.
- **T007 + T010**: These modify different subsections of the prompt and could be done in parallel by different workers.

---

## Implementation Strategy

### MVP First (User Stories 1 + 2 Only)

1. Complete Phase 1: Setup (verify baseline)
2. Complete Phase 2: Foundational (gate only)
3. Complete Phase 3: US1 + US2 (tone + structure)
4. **STOP and VALIDATE**: Run `/gaze` — verify emoji-free, flat structure, terse interpretations
5. This alone delivers the most impactful change

### Incremental Delivery

1. Complete US1+US2 → Test → The report has the right voice
2. Add US3 → Test → Tables are properly formatted with word grades
3. Add US4 → Test → Risk matrix and bottom line appear
4. Add US5 → Test → Recommendations are polished
5. Add Example + Sync → Final gate
6. Each increment improves the report without breaking previous changes
