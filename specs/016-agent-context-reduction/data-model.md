# Data Model: Agent Context Reduction

**Feature**: 016-agent-context-reduction
**Date**: 2026-03-02

## Overview

This feature has no application-level data model. The "data model" consists of the file organization for externalized agent prompt content and the scaffold system's file classification rules.

## File Organization

### Before (current state)

```text
.opencode/
├── agents/
│   └── gaze-reporter.md       # 17,775 bytes — full agent prompt
└── command/
    └── gaze.md                 # 1,308 bytes — command dispatcher

internal/scaffold/assets/       # Embedded copies (byte-identical)
├── agents/
│   └── gaze-reporter.md
└── command/
    └── gaze.md
```

### After (target state)

```text
.opencode/
├── agents/
│   └── gaze-reporter.md       # ~12,100 bytes — reduced prompt
├── command/
│   └── gaze.md                # 1,308 bytes — unchanged
└── references/
    ├── example-report.md      # ~3,400 bytes — canonical example (extracted)
    └── doc-scoring-model.md   # ~2,400 bytes — scoring model (extracted)

internal/scaffold/assets/       # Embedded copies (byte-identical)
├── agents/
│   └── gaze-reporter.md
├── command/
│   └── gaze.md
└── references/
    ├── example-report.md
    └── doc-scoring-model.md
```

## Scaffold File Classification

The scaffold system distinguishes two file categories based on subdirectory:

| Subdirectory | Category | On `gaze init` (file exists) | On `gaze init` (file absent) |
|-------------|----------|------------------------------|------------------------------|
| `agents/` | User-owned | Skip (preserve customizations) | Create |
| `command/` | User-owned | Skip (preserve customizations) | Create |
| `references/` | Tool-owned | Overwrite if content differs | Create |

### Overwrite-on-Diff Decision Flow (references/ only)

```text
[File exists?]
    │
    ├── No → Create file → result.Created
    │
    └── Yes
         │
         ├── [Force=true] → Overwrite → result.Overwritten
         │
         └── [Force=false]
              │
              ├── [Is tool-owned (references/)?]
              │    │
              │    ├── Yes → [Content differs?]
              │    │          │
              │    │          ├── Yes → Overwrite → result.Updated
              │    │          │
              │    │          └── No → Skip → result.Skipped
              │    │
              │    └── No → Skip → result.Skipped
              │
              └── (end)
```

### Result Type Extension

The scaffold `Result` struct gains a new `Updated` field to distinguish overwrite-on-diff from force-overwrite:

| Field | Meaning | When populated |
|-------|---------|---------------|
| `Created` | File did not exist, was written | Always (new files) |
| `Skipped` | File exists, not modified | User-owned files without `--force`; tool-owned files with same content |
| `Overwritten` | File exists, replaced via `--force` | Any file when `Force=true` |
| `Updated` | Tool-owned file exists, content differs, replaced | `references/` files when content changed |

## Reference File Content

### example-report.md

Content extracted from `## Example Output` section of the current agent prompt (lines 382-453). Contains:

- Prose framing: instructions for adapting the example to actual project data
- Markdown code block: 65-line fictional full-mode report demonstrating all 5 sections (CRAP Summary, Quality Summary, Classification Summary, Overall Health Assessment) with correct emoji markers, table structure, metadata format, and recommendation formatting

### doc-scoring-model.md

Content extracted from `### Document-Enhanced Classification` subsection of `## Full Mode` in the current agent prompt (lines 201-250). Contains:

- Document Signal Sources table (5 rows: readme, architecture_doc, specify_file, api_doc, other_md)
- AI Inference Signals table (3 rows: ai_pattern, ai_layer, ai_corroboration)
- Contradiction Penalty rule
- Classification Thresholds table (3 rows: ≥80 contractual, 50-79 ambiguous, <50 incidental)
