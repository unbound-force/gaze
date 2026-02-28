# Research: Report Voice Refinement

**Date**: 2026-02-28
**Feature**: `010-report-voice-refinement`

## R1: Reference Output Analysis

**Decision**: Use the gcal-organizer report output as the canonical reference for formatting, tone, and structure.

**Rationale**: The user explicitly identified the gcal-organizer output as the preferred style. Side-by-side comparison reveals concrete, extractable patterns.

**Key patterns extracted from reference output**:

### Title and Metadata
```
Gaze Health Report — gcal-organizer
Package: ./... | Date: Sat Feb 28, 2026
```
- Plain text, no emoji, em dash separator
- Single-line metadata with pipe separator

### Table Formatting
```markdown
| Function | CRAP | Complexity | Coverage | Location |
|----------|-----:|----------:|---------:|----------|
```
- Right-aligned numeric columns (`:` in separator)
- "Location" not "File" for the source reference column

### Interpretation Style
```
All five have 0% test coverage with high cyclomatic complexity.
```
- One sentence. Factual. References the data. No explanation of what CRAP means.

### Quadrant Labels
```
Q1 — Safe
Q3 — Simple But Underspecified
Q4 — Dangerous
```
- Plain text, em dash, descriptive label. No colored circles.

### Risk Matrix
```markdown
| Priority | Function | Risk | Why |
|:--------:|----------|------|-----|
| 1 | SyncCalendarAttachments | Critical | Complexity 38, GazeCRAP 1482, Q4. Most branching logic in the codebase. |
```
- Priority column center-aligned, numeric
- "Why" is a data-packed clause: metrics first, then a terse rationale

### Overall Grade
```markdown
| Aspect | Rating | Notes |
|--------|--------|-------|
| Library code (internal/) | Fair | Core packages well-tested; orchestration layer is not |
```
- Word-based ratings: Fair, Poor, Good, etc.
- "Notes" column provides context in a single clause

### Bottom Line
```
Bottom line: Solid foundation in utility packages (retry, config, secrets) but the orchestration and CLI layers are significantly under-tested. The two most critical business logic functions are in Q4 "Dangerous". Prioritize refactoring and test-contract work on these before adding new features.
```
- Starts with "Bottom line:"
- 3 sentences: positive acknowledgment, key risk, recommended action

### Recommendations
```
1. Refactor SyncCalendarAttachments (complexity 38 → target <15). Extract sub-responsibilities into separate methods, then add contract-asserting tests.
2. Add contract assertions to retry.Do and config.Load. Both have good line coverage but 0% contract coverage — tests invoke but never assert on outcomes.
```
- Plain numbered list
- Action verb → specific target → metric → brief rationale
- No emoji prefixes

## R2: LLM Prompt Engineering for Output Format Control

**Decision**: Use a combination of explicit rules AND a concrete example output in the agent prompt.

**Rationale**: LLMs follow examples more reliably than abstract formatting rules. Research on prompt engineering consistently shows that few-shot examples produce more consistent formatting compliance than instruction-only approaches. The combination of rules (for the agent to reason about edge cases) plus an example (for the agent to pattern-match against) provides the highest formatting compliance rate.

**Alternatives considered**:
- **Rules only** (current approach): The current prompt uses only abstract rules ("Clear section headers", "Tables for numerical data", "Bold for key metrics"). This produced inconsistent output — the gaze report used emojis and verbose explanations despite no instruction to do so. Rejected because it demonstrably fails to control tone.
- **Example only**: Providing only an example without rules risks the agent copying the example's specific data rather than adapting the format to new data. Also fails on edge cases not covered by the example. Rejected as insufficient alone.
- **Template with placeholders**: Providing a template with `{variable}` placeholders. Too rigid — doesn't allow the agent to adapt section count based on available data. Rejected because it conflicts with the omit-empty-sections requirement.

## R3: Grade Scale Mapping

**Decision**: Use a 5-point word scale: Poor, Fair, Good, Strong, Excellent.

**Rationale**: These words map naturally to quality assessments without requiring the reader to decode letter grades. They are unambiguous, culturally neutral, and sort intuitively from worst to best.

| Word | Equivalent Letter Range | When to Use |
|------|------------------------|-------------|
| Poor | D, F | Metric is critically below acceptable levels |
| Fair | C, C+ | Metric has significant room for improvement |
| Good | B, B+ | Metric meets baseline expectations |
| Strong | A-, A | Metric exceeds expectations |
| Excellent | A+ | Metric is exemplary with minimal room for improvement |

**Alternatives considered**:
- Letter grades (A+, B-, C): Rejected — this is what the current output uses. Ambiguous (is C "average" or "bad"?), and the emoji-prefixed colored versions are what the user dislikes.
- Numeric scores (1-5, 1-10): Rejected — adds another numeric scale on top of CRAP scores, complexity, and coverage percentages. Cognitive overload.
- Traffic light (Red/Yellow/Green): Rejected — maps poorly to a 5-point scale and would tempt the agent to use colored emoji circles.

## R4: Section Ordering in Full Mode

**Decision**: The following section order for full-mode reports:

1. Title + Metadata
2. CRAP Summary (table + one-sentence interpretation)
3. GazeCRAP Quadrant Distribution (if data available; omit if not)
4. Quality Summary (if data available; omit if not)
5. Classification Summary (if data available; omit if not)
6. Overall Health Assessment
   a. Risk Matrix (first table — centerpiece)
   b. Prioritized Recommendations (numbered action sentences)
   c. Overall Grade (Aspect/Rating/Notes table)
   d. Bottom line (closing paragraph)

**Rationale**: This order follows a diagnostic report pattern: raw data first (CRAP), then enriched data (quality, classification), then synthesis (health assessment). The risk matrix leads the assessment section because it's the most actionable artifact. The bottom line closes the report because it's the takeaway the developer remembers.

**Alternatives considered**:
- Health assessment first, data second: Rejected — the assessment loses credibility without the supporting data above it.
- Overall Grade before Risk Matrix: Rejected — the grade is a summary; the risk matrix is the actionable detail. Leading with the detail is more useful.
