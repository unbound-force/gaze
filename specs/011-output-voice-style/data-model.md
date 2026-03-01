# Data Model: Output Voice & Style Standardization

**Branch**: `011-output-voice-style` | **Date**: 2026-03-01

## Overview

This feature modifies the gaze-reporter agent prompt, not Go data structures. The "data model" here is the **voice standard** — a structured set of rules that govern how the agent formats its output. These rules are encoded directly in the prompt markdown, not in Go types.

## Entities

### Voice Standard

The complete rule set governing agent output. Encoded in the `## Output Format` section of `gaze-reporter.md`.

| Attribute | Type | Description |
|-----------|------|-------------|
| Emoji vocabulary | Fixed set of 10 | Closed list of allowed emojis with semantic roles |
| Section marker map | Emoji → section name | Maps each section header to its emoji prefix |
| Severity indicator map | Grade → emoji | Maps letter grades to colored circle emojis |
| Tone anti-patterns | List of 4 bans | Exclamation marks, slang, puns, first-person |
| Interpretation limit | Constraint | One sentence per table, max 25 words |
| Section omission rule | Behavior | Omit sections with no data silently |

### Section Marker Map

| Section | Emoji | Header Text |
|---------|-------|-------------|
| Report title | 🔍 | `🔍 Gaze Full Quality Report` (or mode variant) |
| CRAP summary | 📊 | `📊 CRAP Summary` |
| Quality summary | 🧪 | `🧪 Quality Summary` |
| Classification summary | 🏷️ | `🏷️ Classification Summary` |
| Health assessment | 🏥 | `🏥 Overall Health Assessment` |

### Severity Indicator Map

| Grade Range | Emoji | Semantic |
|-------------|-------|----------|
| A, A-, B+ | 🟢 | Good/safe |
| B, B-, C+, C | 🟡 | Moderate/warning |
| C-, D, F | 🔴 | Critical/danger |
| N/A, no data | ⚪ | Neutral |

### Quadrant-to-Emoji Map

| Quadrant | Emoji | Label |
|----------|-------|-------|
| Q1 | 🟢 | Safe |
| Q2 | 🟡 | Complex But Tested |
| Q3 | ⚪ | Needs Tests |
| Q4 | 🔴 | Dangerous |

### Recommendation Severity Map

| Priority Level | Emoji | Criteria |
|----------------|-------|----------|
| Critical | 🔴 | Zero-coverage functions, Q4 Dangerous items |
| Moderate | 🟡 | Decomposition opportunities, coverage gaps |
| Improvement | 🟢 | Optional analysis runs, minor enhancements |
| Default | 🟡 | When severity is unclear |

## Relationships

- The **Voice Standard** contains all four maps (Section Marker, Severity Indicator, Quadrant, Recommendation Severity).
- The **Section Marker Map** determines which emoji prefixes which section header.
- The **Severity Indicator Map** is used by both the health assessment scorecard (grades) and the quadrant distribution table.
- The **Recommendation Severity Map** is independent of the grade map — recommendations use action-based criteria, not letter grades.

## State Transitions

Not applicable. The voice standard is a static configuration embedded in the prompt. It does not have runtime state.

## Validation Rules

- Every emoji in the output MUST be a member of the 10-emoji vocabulary.
- Every section header MUST use exactly the emoji from the Section Marker Map.
- Every letter grade MUST be paired with the emoji from the Severity Indicator Map.
- No emoji may appear without a semantic role from the vocabulary.
