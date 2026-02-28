# Feature Specification: Report Voice Refinement

**Feature Branch**: `010-report-voice-refinement`
**Created**: 2026-02-28
**Status**: Draft
**Input**: User description: "Refine gaze-reporter agent output to use clinical tone, no emojis, concise formatting with right-aligned numerics, word-based grades, flat structure, prominent risk matrix, and bottom-line summary"

## User Scenarios & Testing *(mandatory)*

### User Story 1 — Emoji-Free, Clinical Report Output (Priority: P1)

A developer runs `/gaze` on any Go project and receives a report that uses plain text throughout — no emojis in section headers, quadrant labels, grades, recommendations, or inline annotations. The report reads like a clinical diagnostic: factual, terse, matter-of-fact. Every sentence conveys data or an actionable observation; there are no pedagogical explanations or filler paragraphs.

**Why this priority**: Emojis and verbose interpretive paragraphs are the most visible deviation from the preferred output style. Eliminating them produces the largest perceived improvement with the smallest change surface.

**Independent Test**: Run `/gaze` on any Go project and verify the output contains zero emoji characters and no multi-sentence interpretive paragraphs.

**Acceptance Scenarios**:

1. **Given** a Go project with CRAP data, **When** the user runs `/gaze`, **Then** the report contains zero emoji characters (no Unicode in ranges U+1F300–U+1FAFF, U+2600–U+27BF, U+FE00–U+FE0F, or colored circle characters).
2. **Given** a report with a "Top 5 Worst CRAP Scores" section, **When** the user reads the interpretation line below the table, **Then** it is a single terse sentence (no more than 25 words) stating the key pattern, not a multi-paragraph explanation.
3. **Given** a report with quadrant distribution, **When** the user reads the quadrant labels, **Then** they use plain text descriptors (e.g., "Q1 — Safe") without any colored circle or emoji prefix.

---

### User Story 2 — Concise Structure and Flat Layout (Priority: P1)

A developer receives a report with a flat, minimal structure. Empty or N/A sections are omitted entirely rather than rendered with placeholder warnings. The report title is a single plain-text line. Metadata appears on one line. Section headers are plain text without decorative prefixes. The report uses the fewest sections necessary to convey the data.

**Why this priority**: Structural conciseness is tightly coupled with the tone goal — a clinical report does not have empty sections or decorative headers. This must ship alongside US1 to achieve the desired voice.

**Independent Test**: Run `/gaze` on a project where quality analysis returns no data. Verify the Quality Summary section is omitted entirely rather than rendered with a warning blockquote.

**Acceptance Scenarios**:

1. **Given** a report title, **When** the user reads the first line, **Then** it reads as plain text in the format `Gaze Health Report — <project-name>`.
2. **Given** report metadata, **When** the user reads the metadata line, **Then** it appears as a single line: `Package: <pattern> | Date: <date>`.
3. **Given** a quality analysis that returns no data, **When** the report is generated, **Then** the Quality Summary section is omitted entirely — no header, no blockquote, no placeholder text.
4. **Given** a classification analysis that succeeds, **When** the report is generated, **Then** the Classification Summary section appears with data but without a separate sub-header for "Interpretation" — the interpretation is a single sentence appended after the table.

---

### User Story 3 — Word-Based Grades and Right-Aligned Numerics (Priority: P2)

The report uses word-based ratings (Poor, Fair, Good, Strong, Excellent) instead of letter grades (C+, A-, B+). Numeric columns in all tables use right-aligned formatting. The CRAPload metric always includes both count and percentage of total functions.

**Why this priority**: These are formatting refinements that improve readability and consistency. They depend on the structural decisions made in US1/US2 but are independently testable.

**Independent Test**: Run `/gaze crap` on any Go project and verify the Overall Grade table uses word-based ratings and numeric columns are right-aligned.

**Acceptance Scenarios**:

1. **Given** a report with an Overall Grade table, **When** the user reads the Rating column, **Then** all values are one of: Poor, Fair, Good, Strong, Excellent — no letter grades, no emoji prefixes.
2. **Given** a report with a CRAP Summary table, **When** the user reads numeric columns (CRAP, Complexity, Coverage), **Then** the markdown table separator uses right-alignment syntax (e.g., `|------:|`).
3. **Given** a CRAPload metric, **When** the user reads the value, **Then** it includes both count and percentage: e.g., "40 functions (29.2%)".
4. **Given** a report with metric labels, **When** the user reads them, **Then** they use concise forms: "Functions analyzed" (not "Total functions analyzed"), "Avg complexity" (not "Average complexity").

---

### User Story 4 — Risk Matrix and Bottom Line (Priority: P2)

The Overall Health Assessment section features a prominent risk matrix table as its centerpiece, with columns: Priority, Function, Risk, Why. Each "Why" cell is a terse, data-packed clause. The report ends with a plain-text "Bottom line:" paragraph (1-3 sentences) summarizing the project's key takeaway and recommended next action.

**Why this priority**: The risk matrix and bottom line are the most actionable parts of the report. They transform raw data into a decision-support artifact. This is independently valuable even without the other formatting changes.

**Independent Test**: Run `/gaze` (full mode) on any Go project and verify the health assessment contains a risk matrix table and the report ends with a "Bottom line:" paragraph.

**Acceptance Scenarios**:

1. **Given** a full-mode report, **When** the user reads the Overall Health Assessment, **Then** the first table is a risk matrix with columns: Priority (centered, numeric), Function, Risk (Critical/High/Medium/Low), Why.
2. **Given** a risk matrix "Why" cell, **When** the user reads it, **Then** it is a single clause containing key metrics and a brief rationale — no more than 20 words.
3. **Given** a full-mode report, **When** the user reads the last paragraph, **Then** it begins with "Bottom line:" and contains 1-3 sentences summarizing the project health and the single most important next action.
4. **Given** a CRAP-only report, **When** the user reads the output, **Then** it still ends with a terse summary sentence (but not labeled "Bottom line:" since there is no cross-referenced health assessment).

---

### User Story 5 — Recommendations as Action Sentences (Priority: P3)

Prioritized recommendations appear as numbered sentences with an action verb, specific target, and measurable goal. No emoji prefixes on recommendation items. Each recommendation references concrete metrics from the report data.

**Why this priority**: This is a refinement of the existing recommendation format. The structural and tonal changes in US1-US4 already improve recommendations significantly; this story polishes the final format.

**Independent Test**: Run `/gaze` on a project with CRAPload > 0 and verify each recommendation is a single action sentence without emoji.

**Acceptance Scenarios**:

1. **Given** a report with prioritized recommendations, **When** the user reads recommendation #1, **Then** it begins with an action verb (e.g., "Refactor", "Add", "Break up", "Write", "Consider") followed by a specific function or package name.
2. **Given** a recommendation, **When** the user reads it, **Then** it includes at least one concrete metric from the report (e.g., "complexity 38 → target <15", "0% coverage", "CRAP 650").
3. **Given** a list of recommendations, **When** the user reads the prefixes, **Then** they are plain numbers (1., 2., 3.) without emoji or colored indicators.

---

### Edge Cases

- What happens when the project has zero functions above the CRAP threshold? The report MUST still produce a valid health assessment with a positive "Bottom line:" acknowledging the clean state.
- What happens when GazeCRAP data is unavailable (no contract coverage)? The report MUST omit the GazeCRAP sections entirely rather than showing "N/A" or warning blockquotes.
- What happens when only one gaze command succeeds (e.g., `gaze crap` works but `gaze quality` fails)? The report MUST produce a partial report covering only the successful sections, with a single-line note at the end listing which analyses were unavailable.
- What happens when the project name is very long (>50 characters)? The title line MUST truncate or wrap gracefully without breaking the single-line format.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The gaze-reporter agent prompt MUST instruct the agent to produce output with zero emoji characters in all modes (crap, quality, full).
- **FR-002**: The gaze-reporter agent prompt MUST specify plain-text section headers without decorative prefixes (no emoji, no Unicode symbols).
- **FR-003**: The gaze-reporter agent prompt MUST specify a single-line report title in the format `Gaze Health Report — <project-name>`.
- **FR-004**: The gaze-reporter agent prompt MUST specify single-line metadata: `Package: <pattern> | Date: <date>`.
- **FR-005**: The gaze-reporter agent prompt MUST instruct the agent to omit sections entirely when data is unavailable, rather than rendering placeholder warnings or N/A content.
- **FR-006**: The gaze-reporter agent prompt MUST specify word-based ratings (Poor, Fair, Good, Strong, Excellent) for the Overall Grade table.
- **FR-007**: The gaze-reporter agent prompt MUST specify right-aligned markdown column syntax for all numeric data columns.
- **FR-008**: The gaze-reporter agent prompt MUST specify CRAPload format as count with percentage: e.g., "40 functions (29.2%)".
- **FR-009**: The gaze-reporter agent prompt MUST specify concise metric labels (e.g., "Functions analyzed" not "Total functions analyzed").
- **FR-010**: The gaze-reporter agent prompt MUST specify a Risk Matrix table in the Overall Health Assessment with columns: Priority (centered), Function, Risk, Why.
- **FR-011**: The gaze-reporter agent prompt MUST specify that every full-mode report ends with a "Bottom line:" paragraph of 1-3 sentences.
- **FR-012**: The gaze-reporter agent prompt MUST specify that recommendations use numbered action sentences without emoji prefixes.
- **FR-013**: The gaze-reporter agent prompt MUST specify terse, single-sentence interpretations after data tables — no multi-paragraph explanations.
- **FR-014**: The gaze-reporter agent prompt MUST specify that the "Why" column in the risk matrix contains data-packed clauses of no more than 20 words.
- **FR-015**: Changes to the agent prompt MUST be applied to both locations: `.opencode/agents/gaze-reporter.md` and `internal/scaffold/assets/agents/gaze-reporter.md`.
- **FR-016**: The agent prompt MUST include an example output snippet showing the expected formatting for at least the title, metadata, CRAP summary table, and risk matrix.

### Key Entities

- **gaze-reporter agent prompt**: The markdown file (with YAML frontmatter) that defines the agent's behavior, output format, and tone. Exists in two locations that must stay synchronized.
- **Report sections**: Title, Metadata, CRAP Summary, GazeCRAP Quadrants, Quality Summary, Classification Summary, Overall Health Assessment (Risk Matrix, Overall Grade, Recommendations, Bottom Line).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Running `/gaze` on any Go project produces output containing zero emoji characters (verified by scanning the output for Unicode emoji ranges).
- **SC-002**: The report uses word-based ratings exclusively — no letter grades appear anywhere in the output.
- **SC-003**: Every full-mode report ends with a "Bottom line:" paragraph.
- **SC-004**: Sections with unavailable data are omitted entirely — no "N/A", no warning blockquotes, no placeholder headers.
- **SC-005**: The risk matrix table appears as the first table in the Overall Health Assessment section in every full-mode report.
- **SC-006**: Both agent prompt files (`.opencode/agents/gaze-reporter.md` and `internal/scaffold/assets/agents/gaze-reporter.md`) are byte-identical after the change.
- **SC-007**: The agent prompt includes at least one example output snippet demonstrating the expected formatting.
- **SC-008**: Numeric columns in all example tables use right-alignment markdown syntax.
