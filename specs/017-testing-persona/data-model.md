# Data Model: Testing Persona Integration

**Feature**: 017-testing-persona
**Date**: 2026-03-05

## Overview

This feature does not introduce persistent data structures, database schemas, or API contracts. All deliverables are static markdown files (agent prompts, command definitions) and Go source code modifications (scaffold system). The "data model" describes the file inventory, ownership classification, and the relationships between files.

## File Inventory

### New Files

| File | Type | Scaffold? | Ownership | Description |
|------|------|-----------|-----------|-------------|
| `.opencode/agents/reviewer-testing.md` | Agent | Yes | User-owned | The Tester: test quality and testability auditor |
| `.opencode/command/speckit.testreview.md` | Command | Yes | Tool-owned | Testability analysis command |
| `.opencode/command/review-council.md` | Command | Yes (new) | Tool-owned | 4-reviewer governance council |
| `internal/scaffold/assets/agents/reviewer-testing.md` | Embed | N/A | N/A | Byte-identical copy for embed.FS |
| `internal/scaffold/assets/command/speckit.testreview.md` | Embed | N/A | N/A | Byte-identical copy for embed.FS |
| `internal/scaffold/assets/command/review-council.md` | Embed | N/A | N/A | Byte-identical copy for embed.FS |

### Modified Files

| File | Change Summary |
|------|---------------|
| `.specify/memory/constitution.md` | Add Principle IV: Testability; version 1.0.0 → 1.1.0 |
| `internal/scaffold/scaffold.go` | `isToolOwned` → explicit file list; `printSummary` hint update |
| `internal/scaffold/scaffold_test.go` | File count 4 → 7; new `TestIsToolOwned`; expand overwrite-on-diff tests |
| `AGENTS.md` | Document new agent, command, principle, scaffold changes |

### Unchanged Files

| File | Why Unchanged |
|------|---------------|
| `.opencode/agents/gaze-reporter.md` | Not related to testing persona |
| `.opencode/agents/reviewer-adversary.md` | Existing reviewer, not modified |
| `.opencode/agents/reviewer-architect.md` | Existing reviewer, not modified |
| `.opencode/agents/reviewer-guard.md` | Existing reviewer, not modified |
| `.opencode/command/gaze.md` | Not related to testing persona |
| `.opencode/command/speckit.*.md` (8 files) | FR-017: no Speckit commands modified |
| `.opencode/references/*.md` (2 files) | Not related to testing persona |
| `internal/scaffold/assets/agents/gaze-reporter.md` | Existing embed, unchanged |
| `internal/scaffold/assets/command/gaze.md` | Existing embed, unchanged |
| `internal/scaffold/assets/references/*.md` (2 files) | Existing embeds, unchanged |

## Ownership Model

### Classification Rules

```
isToolOwned(relPath) → bool:
  1. If relPath starts with "references/" → true (directory-level ownership)
  2. If relPath is "command/speckit.testreview.md" → true (explicit file)
  3. If relPath is "command/review-council.md" → true (explicit file)
  4. Otherwise → false (user-owned)
```

### Ownership Behavior Matrix

| Ownership | File Exists? | --force? | Behavior | Result Category |
|-----------|-------------|----------|----------|-----------------|
| User-owned | No | N/A | Create file | Created |
| User-owned | Yes | No | Skip | Skipped |
| User-owned | Yes | Yes | Overwrite | Overwritten |
| Tool-owned | No | N/A | Create file | Created |
| Tool-owned | Yes (same content) | No | Skip | Skipped |
| Tool-owned | Yes (diff content) | No | Overwrite | Updated |
| Tool-owned | Yes | Yes | Overwrite | Overwritten |

### File Count Summary

| Category | Count | Files |
|----------|-------|-------|
| User-owned agents | 2 | gaze-reporter.md, reviewer-testing.md |
| User-owned commands | 1 | gaze.md |
| Tool-owned commands | 2 | speckit.testreview.md, review-council.md |
| Tool-owned references | 2 | doc-scoring-model.md, example-report.md |
| **Total scaffold files** | **7** | |

## Agent Relationships

```
/review-council (command)
  ├── delegates to: reviewer-adversary (agent) — security/resilience
  ├── delegates to: reviewer-architect (agent) — architecture/conventions
  ├── delegates to: reviewer-guard (agent) — intent drift/zero-waste
  └── delegates to: reviewer-testing (agent) — test quality/testability [NEW]

/speckit.testreview (command) [NEW]
  └── delegates to: reviewer-testing (agent) — spec review mode only

/gaze (command)
  └── delegates to: gaze-reporter (agent) — quality reports [UNCHANGED]
```

## Constitution Entity

### Principle IV: Testability

**Position**: After Principle III (Actionable Output), before Development Workflow
**Version impact**: 1.0.0 → 1.1.0 (MINOR)
**Scope**: Dual — Gaze's own code AND user codebase analysis accuracy

**MUST statements**:
1. Functions MUST be testable in isolation (no external services, no shared mutable state)
2. Test contracts MUST verify observable side effects, not implementation details
3. Coverage strategy MUST be specified in plans for new code
4. Coverage ratchets MUST be enforced; regression MUST be treated as test failure

**Severity mapping** (for The Tester agent):
- Missing coverage strategy → CRITICAL
- Vague acceptance criteria → HIGH
- Missing fixture specification → MEDIUM
- Minor convention deviation → LOW
