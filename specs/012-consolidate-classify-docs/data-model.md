# Data Model: Consolidate /classify-docs into /gaze

**Date**: 2026-03-01
**Branch**: `012-consolidate-classify-docs`

## Overview

This feature modifies markdown prompt files and Go test code. There are no new data structures, database schemas, or API contracts. The "data model" describes the entities being transformed and the scoring model being relocated.

## Entity: Scaffold Asset Manifest

The scaffold system's embedded asset manifest changes from 4 files to 2:

### Before (4 files)

| Relative Path | Type | Purpose |
|---------------|------|---------|
| `agents/gaze-reporter.md` | Agent | Quality reporting agent prompt |
| `agents/doc-classifier.md` | Agent | Document-enhanced classification agent prompt |
| `command/gaze.md` | Command | `/gaze` command dispatcher |
| `command/classify-docs.md` | Command | `/classify-docs` command dispatcher |

### After (2 files)

| Relative Path | Type | Purpose |
|---------------|------|---------|
| `agents/gaze-reporter.md` | Agent | Quality reporting agent prompt (now includes inlined scoring model) |
| `command/gaze.md` | Command | `/gaze` command dispatcher (unchanged) |

### Removed

| Relative Path | Reason |
|---------------|--------|
| `agents/doc-classifier.md` | Scoring logic absorbed into gaze-reporter |
| `command/classify-docs.md` | Command consolidated into `/gaze` full mode |

## Entity: Document-Signal Scoring Model

The scoring model migrates from `doc-classifier.md` into the gaze-reporter prompt's Full Mode section.

### Signal Sources (unchanged)

| Source | Weight Range | Direction | Evidence |
|--------|-------------|-----------|---------|
| `readme` | ±5 to ±15 | Positive: function named in README; Negative: described as internal | Module README |
| `architecture_doc` | ±5 to ±20 | Positive: contract declared; Negative: marked as implementation detail | Architecture/design docs |
| `specify_file` | ±5 to ±25 | Positive: spec names as required behavior; Negative: spec marks as optional | `specs/` files |
| `api_doc` | ±5 to ±20 | Positive: listed in API reference; Negative: marked as non-public | API reference docs |
| `other_md` | ±2 to ±10 | Positive: referenced in markdown; Negative: described as debug/internal | Other markdown files |

### AI Inference Signals (unchanged)

| Source | Weight Range | Evidence |
|--------|-------------|---------|
| `ai_pattern` | +5 to +15 | Recognizable design pattern whose contract implies the side effect |
| `ai_layer` | +5 to +15 | Architectural layer analysis (e.g., service layer mutations are usually contractual) |
| `ai_corroboration` | +3 to +10 | Multiple independent document signals agree |

### Contradiction Penalty (unchanged)

Up to -20 when document signals and mechanical signals disagree.

### Classification Thresholds (unchanged)

| Confidence | Label |
|-----------|-------|
| ≥ 80 | contractual |
| 50–79 | ambiguous |
| < 50 | incidental |

## Entity: Gaze-Reporter Prompt Structure

### Before

```
1. YAML frontmatter
2. Opening paragraph
3. Binary Resolution
4. Mode Parsing
5. CRAP Mode
6. Quality Mode
7. Full Mode (with /classify-docs delegation)
8. Output Format (emoji vocabulary, tone, etc.)
9. Example Output (canonical example)
10. Graceful Degradation
11. Error Handling
```

### After (sandwich structure)

```
1. YAML frontmatter
2. Opening paragraph
3. FORMATTING OVERRIDE BLOCK (new)
4. COMPACT EXAMPLE (new — sandwich top)
5. Binary Resolution
6. Mode Parsing
7. CRAP Mode
8. Quality Mode
9. Full Mode (rewritten — direct orchestration)
   9a. Document-Enhanced Classification (new — inlined scoring model)
10. Output Format (emoji vocabulary, tone, etc.)
11. FULL Example Output (retained — sandwich bottom)
12. Graceful Degradation
13. Error Handling
```

### New Sections Detail

**Section 3 — Formatting Override Block**: 5-8 lines of imperative language declaring emoji usage as mandatory contract, not optional preference. Positioned immediately after the opening paragraph to intercept system-level emoji suppression before any output is generated.

**Section 4 — Compact Example**: 10-15 line abbreviated example showing title with 🔍, section header with 📊, table formatting, quadrant emojis (🟢🟡🔴⚪), and recommendation severity prefixes (🔴🟡🟢). Serves as visual anchor.

**Section 9a — Document-Enhanced Classification**: ~60-70 lines containing the condensed scoring model (signal sources, AI inference, contradiction penalty, thresholds). Positioned within Full Mode after the 4-command execution list and before the report section definitions.

## State Transitions

Not applicable — this feature has no state machines or lifecycle transitions.

## Validation Rules

| Rule | Source | Enforcement |
|------|--------|-------------|
| Embedded assets = 2 files | FR-006 | `TestAssetPaths_Returns2Files` |
| Embedded files match live files | FR-007 | `TestEmbeddedAssetsMatchSource` |
| No `/classify-docs` references | SC-008 | Manual grep verification |
| Scoring model present in gaze-reporter | SC-009 | Manual content verification |
