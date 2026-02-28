# Data Model: Report Voice Refinement

**Date**: 2026-02-28
**Feature**: `010-report-voice-refinement`

## Overview

This feature modifies a prompt file, not a data store. The "data model" describes the structure of the agent prompt and the report output format it governs.

## Entity: Agent Prompt File

**Locations** (must be byte-identical):
- `.opencode/agents/gaze-reporter.md`
- `internal/scaffold/assets/agents/gaze-reporter.md`

### Structure

```
┌─────────────────────────────────┐
│ YAML Frontmatter                │  ← UNCHANGED
│ (description, tool permissions) │
├─────────────────────────────────┤
│ # Gaze Reporter Agent           │  ← UNCHANGED (intro sentence updated for tone)
│ ## Binary Resolution            │  ← UNCHANGED
│ ## Mode Parsing                 │  ← UNCHANGED
│ ## CRAP Mode                    │  ← MODIFIED (formatting instructions)
│ ## Quality Mode                 │  ← MODIFIED (remove blockquote fallback)
│ ## Full Mode                    │  ← MODIFIED (add risk matrix, grades, bottom line)
│ ## Output Format                │  ← REWRITTEN (tone directive, formatting rules)
│ ## Example Output               │  ← NEW SECTION
│ ## Graceful Degradation         │  ← MODIFIED (omit-over-placeholder rule)
│ ## Error Handling               │  ← UNCHANGED
└─────────────────────────────────┘
```

## Entity: Report Output Sections

### Full Mode Section Order

```
Title                          "Gaze Health Report — <project-name>"
Metadata                       "Package: <pattern> | Date: <date>"
───────────────────────────────
CRAP Summary                   Table + one-sentence interpretation
GazeCRAP Quadrant Distribution Table (omit if no data)
Quality Summary                Table (omit if no data)
Classification Summary         Table + one-sentence interpretation (omit if no data)
───────────────────────────────
Overall Health Assessment
├── Risk Matrix                Priority | Function | Risk | Why
├── Recommendations            Numbered action sentences (3-5)
├── Overall Grade              Aspect | Rating | Notes
└── Bottom line                1-3 sentence closing paragraph
───────────────────────────────
[Unavailable analyses note]    Single line, only if sections omitted
```

### CRAP Mode Section Order

```
Title + Metadata
CRAP Summary (table + interpretation)
GazeCRAP Quadrant Distribution (if available)
Summary sentence
```

### Quality Mode Section Order

```
Title + Metadata
Quality Summary (table + interpretation)
Summary sentence
```

## Grade Scale

| Rating | Criteria |
|--------|----------|
| Poor | Metric critically below acceptable levels |
| Fair | Significant room for improvement |
| Good | Meets baseline expectations |
| Strong | Exceeds expectations |
| Excellent | Exemplary, minimal room for improvement |

## Risk Level Scale

| Level | Criteria |
|-------|----------|
| Critical | Zero coverage + high complexity, or Q4 Dangerous with GazeCRAP > 100 |
| High | CRAP > threshold with < 50% coverage, or 0% coverage with moderate complexity |
| Medium | CRAP near threshold, or good coverage but 0% contract coverage |
| Low | CRAP below threshold with minor coverage gaps |
