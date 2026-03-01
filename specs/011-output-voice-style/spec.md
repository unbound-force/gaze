# Feature Specification: Output Voice & Style Standardization

**Feature Branch**: `011-output-voice-style`  
**Created**: 2026-03-01  
**Status**: Draft  
**Supersedes**: Spec 010 (report-voice-refinement) -- replaces the clinical, emoji-free voice with a fun, emoji-rich voice  
**Input**: User description: "Standardize the tone (as fun), format, and use of emojis in the OpenCode gaze commands output"

## Context

Gaze reports are consumed by developers inside an OpenCode agent environment. The current report voice (established by spec 010) is clinical, terse, and emoji-free. User feedback indicates this voice feels sterile and makes reports less engaging to scan. The desired voice is **fun, approachable, and visually navigable** -- using emojis as section markers and severity indicators while keeping the data itself precise and actionable.

A reference output that captures the desired voice is provided below as the **canonical example** for this spec.

### Reference Output (Canonical Example)

```
🔍 Gaze Full Quality Report
Project: github.com/unbound-force/gaze · Branch: 009-crapload-reduction
Gaze Version: v1.0.0 (dev) · Go: 1.24.6 · Date: 2026-02-28
---
📊 CRAP Summary
| Metric | Value |
|--------|-------|
| Total functions analyzed | 216 |
| Average complexity | 6.2 |
| Average line coverage | 79.0% |
| Average CRAP score | 7.7 |
| CRAPload | 24 (functions ≥ threshold 15) |

Top 5 Worst CRAP Scores
| Function | CRAP | Complexity | Coverage | File |
|----------|------|-----------|----------|------|
| (analyzeModel).Update | 42.0 | 6 | 0.0% | cmd/gaze/interactive.go:163 |
| ...

GazeCRAP Quadrant Distribution
| Quadrant | Count | Meaning |
|----------|-------|---------|
| 🟢 Q1 — Safe | 29 | Low complexity, high contract coverage |
| 🟡 Q2 — Complex But Tested | 1 | High complexity, contracts verified |
| 🔴 Q4 — Dangerous | 4 | Complex AND contracts not adequately verified |
| ⚪ Q3 — Needs Tests | 0 | Simple but underspecified |

GazeCRAPload: 4 — All 4 Q4 functions (...) have 100% contract coverage — their risk is purely from high cyclomatic complexity (15–18), meaning they need decomposition, not more tests.
---
🧪 Quality Summary
> ⚠️ Module-level quality analysis returned 0 tests ...
---
🏷️ Classification Summary
| Classification | Count | % |
|---------------|-------|---|
| Contractual | 73 | 31.3% |
| Ambiguous | 155 | 66.5% |
| Incidental | 8 | 3.4% |

The 66.5% ambiguous rate is typical for mechanical-only classification. Run /classify-docs to incorporate document signals and reduce ambiguity.
---
🏥 Overall Health Assessment
Summary Scorecard
| Dimension | Grade | Details |
|-----------|-------|---------|
| CRAPload | 🟡 C+ | 24/216 functions (11%) above threshold |
| GazeCRAPload | 🟢 A | Only 4 functions above threshold |
| Avg Line Coverage | 🟢 B+ | 79.0% |
| ...

Top 5 Prioritized Recommendations
1. 🔴 Add tests for zero-coverage functions — ...
2. 🔴 Increase coverage for isPointerArgStore — ...
3. 🟡 Decompose high-complexity functions — ...
4. 🟡 Resolve ambiguous classifications — ...
5. 🟢 Run per-package quality analysis — ...
```

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Scanning a Full Quality Report (Priority: P1)

A developer runs a full Gaze quality report through the OpenCode agent. They receive a report that uses emoji section markers, color-coded severity indicators, and a conversational-but-precise tone. They can visually scan the report and immediately locate the section they care about (CRAP scores, quality, classification, or health assessment) by recognizing the emoji markers.

**Why this priority**: The full quality report is the primary output users interact with. If the voice and formatting are right here, the rest follows naturally.

**Independent Test**: Can be fully tested by generating a full-mode report and verifying that every section header uses the correct emoji prefix, tables are properly formatted, and interpretive commentary uses the defined tone vocabulary.

**Acceptance Scenarios**:

1. **Given** a project with CRAP, quality, classification, and side-effect data available, **When** the gaze-reporter agent produces a "full" mode report, **Then** each major section header is prefixed with its designated emoji marker (e.g., "📊 CRAP Summary", "🧪 Quality Summary", "🏷️ Classification Summary", "🏥 Overall Health Assessment").
2. **Given** a full report is generated, **When** the user scans the output, **Then** severity-graded items use colored circle emojis (🟢 for good/safe, 🟡 for warning/moderate, 🔴 for danger/critical, ⚪ for neutral/missing).
3. **Given** a full report is generated, **When** the report includes interpretive sentences, **Then** those sentences are concise (one sentence per data table), factual, and use an approachable tone without filler or pedagogy.

---

### User Story 2 - Reading a CRAP-Only Report (Priority: P2)

A developer runs a CRAP-only report. They see the same emoji-prefixed headers, severity indicators, and tone as the full report, but scoped to CRAP data only.

**Why this priority**: CRAP-only mode is the second most common usage pattern. Ensuring voice consistency across modes prevents a disjointed user experience.

**Independent Test**: Can be tested by generating a "crap" mode report and verifying emoji markers, table format, and tone match the full-mode voice standard.

**Acceptance Scenarios**:

1. **Given** a project with CRAP data, **When** the gaze-reporter agent produces a "crap" mode report, **Then** the report title is "🔍 Gaze CRAP Report" and the CRAP summary section uses "📊" as its section marker.
2. **Given** a CRAP-only report, **When** quadrant distribution is included, **Then** each quadrant row uses the designated color circle emoji (🟢, 🟡, 🔴, ⚪).

---

### User Story 3 - Quality-Only Report Consistency (Priority: P3)

A developer runs a quality-only report. The voice, emoji usage, and formatting are consistent with other report modes.

**Why this priority**: Ensures the voice standard is applied uniformly, not just to the most common mode.

**Independent Test**: Can be tested by generating a "quality" mode report and verifying it follows the same voice and format conventions.

**Acceptance Scenarios**:

1. **Given** a project with quality data, **When** the gaze-reporter agent produces a "quality" mode report, **Then** the report uses "🧪 Quality Summary" as its section header and maintains the same tone and formatting conventions as the full report.

---

### User Story 4 - Recommendations with Severity Markers (Priority: P2)

A developer reads the prioritized recommendations section. Each recommendation is prefixed with a colored circle emoji indicating its severity/priority, making it visually obvious which items are most urgent.

**Why this priority**: Recommendations are the most actionable part of the report. Visual severity markers accelerate triage.

**Independent Test**: Can be tested by generating a report with recommendations and verifying each recommendation line starts with the correct severity emoji.

**Acceptance Scenarios**:

1. **Given** a report with prioritized recommendations, **When** a recommendation addresses a critical issue (e.g., zero-coverage functions), **Then** it is prefixed with "🔴".
2. **Given** a report with recommendations, **When** a recommendation addresses a moderate issue (e.g., decomposition opportunities), **Then** it is prefixed with "🟡".
3. **Given** a report with recommendations, **When** a recommendation addresses an improvement opportunity (e.g., running additional analysis), **Then** it is prefixed with "🟢".

---

### Edge Cases

- What happens when a section has no data? The section is omitted entirely -- no header, no emoji, no placeholder text.
- What happens when the report contains only one section (e.g., only CRAP data)? The report still uses the full title line, metadata line, and the single section with its emoji marker.
- What happens when a grade falls on the boundary between two severity levels? The voice standard defines explicit grade-to-emoji mappings so boundary behavior is deterministic.
- What happens when a recommendation has no clear severity? Default to 🟡 (moderate) to avoid both under-alarming and over-alarming.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Report output MUST use designated emoji prefixes for all major section headers. The mapping is: 🔍 (report title), 📊 (CRAP summary), 🧪 (quality summary), 🏷️ (classification summary), 🏥 (overall health assessment).
- **FR-002**: Report output MUST use colored circle emojis as severity indicators throughout: 🟢 (good/safe/A-range), 🟡 (moderate/warning/B-C range), 🔴 (critical/danger/D-F range), ⚪ (neutral/no data/N/A).
- **FR-003**: Report output MUST use letter grades (A, A-, B+, B, B-, C+, C, C-, D, F) paired with their corresponding colored circle emoji in the health assessment scorecard.
- **FR-004**: Report tone MUST be approachable and conversational while remaining precise. Interpretive sentences are allowed (and encouraged) after data tables, limited to one concise sentence per table.
- **FR-005**: Report output MUST NOT include pedagogical explanations of what CRAP scores, quadrants, or coverage metrics mean. The reader is assumed to be a developer who already understands the concepts.
- **FR-006**: Report sections with no available data MUST be silently omitted. No empty sections, no "N/A" placeholders, no headers without content.
- **FR-007**: Recommendations MUST be numbered and prefixed with a severity emoji (🔴, 🟡, or 🟢) that corresponds to the urgency of the recommended action.
- **FR-008**: The voice standard MUST apply uniformly across all report modes (full, crap, quality). No mode produces output that contradicts the voice conventions.
- **FR-009**: Data tables MUST use markdown table format with consistent column alignment. Numeric columns are right-aligned where the rendering context supports it.
- **FR-010**: The report MUST include a metadata line immediately below the title showing project path, branch, version, and date, separated by centered dots (·).
- **FR-011**: Major sections MUST be separated by horizontal rules (---).
- **FR-012**: The GazeCRAPload summary line MUST provide a brief, conversational interpretation of what the Q4 function count means in practical terms (e.g., whether the issue is coverage or complexity).
- **FR-013**: Warning callouts MUST use the ⚠️ emoji prefix inside a blockquote (> ⚠️ ...) to visually distinguish advisory notices from data sections.

### Key Entities

- **Voice Standard**: The complete set of rules governing tone, emoji usage, and formatting for all Gaze agent-generated output. This is the central artifact produced by this feature.
- **Section Marker**: An emoji character that prefixes a major section header, serving as a visual anchor for scanning.
- **Severity Indicator**: A colored circle emoji (🟢🟡🔴⚪) that communicates the urgency or health status of a data point.
- **Grade**: A letter grade (A through F, with +/- modifiers) paired with a severity indicator, used in health assessment scorecards.

## Assumptions

- The voice standard applies to the **gaze-reporter agent prompt** output only, not to the Go CLI's direct text/JSON formatters. The CLI formatters use lipgloss styling and remain unchanged.
- Emojis render correctly in the user's terminal/editor environment. OpenCode agent output is displayed in contexts that support Unicode emoji rendering.
- The reference output provided by the user is treated as the **canonical example** -- the voice standard is derived from it, not the other way around.
- This spec supersedes the voice decisions made in spec 010 (report-voice-refinement). Where spec 010 prohibited emojis and mandated clinical tone, this spec replaces those directives with the fun, emoji-rich voice described here.
- The "fun" tone means approachable and conversational, not silly or unprofessional. The report remains a diagnostic tool -- it just doesn't read like a medical chart.
- The GazeCRAPload interpretive line (FR-012) is a brief sentence, not a paragraph. It distills the practical takeaway from the Q4 data.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of major section headers in agent-generated reports use the designated emoji prefix from the voice standard.
- **SC-002**: 100% of severity-graded items (quadrant rows, scorecard grades, recommendations) use the correct colored circle emoji for their severity level.
- **SC-003**: All three report modes (full, crap, quality) produce output that conforms to the same voice standard -- no mode deviates from the emoji, tone, or formatting conventions.
- **SC-004**: Zero sections appear in the output with no data -- empty sections are silently omitted in 100% of cases.
- **SC-005**: Every interpretive sentence in the report conveys a specific, actionable observation. Zero filler sentences ("This section shows..." or "The data indicates that...") appear in the output.
- **SC-006**: The gaze-reporter agent prompt explicitly defines the complete emoji vocabulary, tone guidelines, and formatting rules such that any agent model produces consistent output from the same data.
- **SC-007**: The report output matches the canonical example's structure: title with emoji, metadata line, horizontal-rule-separated sections, emoji-prefixed headers, markdown tables, severity-colored grades, numbered recommendations with emoji prefixes.
