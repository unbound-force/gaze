# Feature Specification: Consolidate /classify-docs into /gaze and Fix Formatting Fidelity

**Feature Branch**: `012-consolidate-classify-docs`  
**Created**: 2026-03-01  
**Status**: Draft  
**Input**: User description: "Consolidate /classify-docs into /gaze and fix formatting fidelity"

## Context

The gaze OpenCode integration currently distributes two commands (`/gaze` and `/classify-docs`) and two agents (`gaze-reporter` and `doc-classifier`). The `/classify-docs` command exists as a standalone entry point that delegates to the `doc-classifier` agent. This creates three problems:

1. **Redundant command**: The gaze-reporter agent already attempts to invoke `/classify-docs` during full mode (lines 134-136 of `gaze-reporter.md`), but this inter-command delegation is unreliable. The `/classify-docs` command is never run standalone by users in practice — it only exists as a full-mode sub-step.

2. **Portability failure**: The `/classify-docs` command hardcodes `go build -o "${TMPDIR:-/tmp}/gaze-classify-docs" ./cmd/gaze` (line 36 of `classify-docs.md`), which fails in any project that is not the gaze repository itself. The gaze-reporter agent already has a robust 3-step binary resolution strategy (build from source, check PATH, go install) that does not have this problem.

3. **Formatting fidelity**: The gaze-reporter agent's prompt defines a mandatory emoji vocabulary (10 emojis), section markers, severity indicators, and a canonical example output. However, the agent's actual output does not reliably match this specification — emojis are missing, structure differs, and tone is plain/unstyled. The root cause is likely conflict with OpenCode's system-level instruction "Only use emojis if the user explicitly requests it." The agent prompt must be updated to explicitly assert that emoji usage is a required part of its output contract, not a user preference.

### Current Architecture

```
.opencode/
  command/
    gaze.md              → delegates to gaze-reporter agent
    classify-docs.md     → delegates to doc-classifier agent (TO BE REMOVED)
  agents/
    gaze-reporter.md     → runs CLI, formats report, attempts /classify-docs delegation
    doc-classifier.md    → enriches mechanical classification with doc signals (TO BE REMOVED)

internal/scaffold/
  assets/
    command/
      gaze.md            → embedded copy
      classify-docs.md   → embedded copy (TO BE REMOVED)
    agents/
      gaze-reporter.md   → embedded copy
      doc-classifier.md  → embedded copy
  scaffold.go            → embed.FS walker, distributes 4 files
  scaffold_test.go       → expects exactly 4 files, drift detection
```

### Target Architecture

```
.opencode/
  command/
    gaze.md              → delegates to gaze-reporter agent (UNCHANGED)
  agents/
    gaze-reporter.md     → runs CLI, formats report, includes inlined doc-signal scoring
    (doc-classifier.md REMOVED — scoring logic absorbed into gaze-reporter)

internal/scaffold/
  assets/
    command/
      gaze.md            → embedded copy (UNCHANGED)
    agents/
      gaze-reporter.md   → embedded copy (UPDATED)
      (doc-classifier.md REMOVED)
  scaffold.go            → embed.FS walker, distributes 2 files
  scaffold_test.go       → expects exactly 2 files, drift detection
```

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Full Mode Produces Classification Without /classify-docs (Priority: P1)

A developer runs `/gaze` (full mode) in a project that has gaze installed via PATH or `go install`. The full report includes a Classification Summary section with contractual/ambiguous/incidental distribution. The doc-classifier enrichment happens inline without requiring the `/classify-docs` command to exist.

**Why this priority**: This is the core consolidation. If the classification workflow doesn't work without `/classify-docs`, the command cannot be removed.

**Independent Test**: Can be tested by deleting `.opencode/command/classify-docs.md`, running `/gaze` in full mode, and verifying the classification section appears in the report.

**Acceptance Scenarios**:

1. **Given** a project with gaze on PATH and no `/classify-docs` command, **When** the user runs `/gaze ./...`, **Then** the full report includes a Classification Summary section.
2. **Given** a project with documentation files present, **When** the gaze-reporter runs full mode, **Then** it runs `gaze analyze --classify --format=json` and `gaze docscan`, applies document-signal scoring inline, and includes the enriched classification in the report.
3. **Given** a project with no documentation files (docscan returns empty), **When** the gaze-reporter runs full mode, **Then** it uses mechanical-only classification results and includes the Classification Summary section with a note about running docscan.

---

### User Story 2 - gaze init Distributes 2 Files (Priority: P1)

A developer runs `gaze init` in a new project. It creates exactly 2 files: `.opencode/agents/gaze-reporter.md` and `.opencode/command/gaze.md`. Neither `classify-docs.md` nor `doc-classifier.md` is created.

**Why this priority**: The scaffold system must match the new architecture. Distributing a removed command breaks the zero-waste mandate.

**Independent Test**: Can be tested by running `gaze init` in an empty directory and counting the created files.

**Acceptance Scenarios**:

1. **Given** an empty project directory with a go.mod, **When** the user runs `gaze init`, **Then** exactly 2 files are created under `.opencode/`.
2. **Given** an existing project with 4 scaffolded files (including `classify-docs.md` and `doc-classifier.md`), **When** the user runs `gaze init --force`, **Then** 2 files are overwritten and neither `classify-docs.md` nor `doc-classifier.md` is recreated (but neither is deleted — removal of stale files is out of scope).

---

### User Story 3 - Emoji and Formatting Fidelity (Priority: P1)

A developer runs `/gaze` in any mode. The report output uses the mandatory emoji vocabulary: section markers (🔍📊🧪🏷️🏥), severity indicators (🟢🟡🔴⚪), warning callouts (⚠️), and letter grades paired with severity emojis. The output matches the canonical example structure.

**Why this priority**: The formatting fidelity fix is equal priority because it addresses a user-visible regression — the agent prompt's formatting spec is currently being silently overridden by OpenCode's system instructions.

**Independent Test**: Can be tested by running `/gaze` and checking the output for the presence of all mandatory emojis in their correct positions.

**Acceptance Scenarios**:

1. **Given** a project with CRAP data, **When** the gaze-reporter produces a report, **Then** the report title starts with "🔍", the CRAP section header starts with "📊", and quadrant rows use 🟢🟡🔴⚪.
2. **Given** a full report with a health assessment, **When** the scorecard is rendered, **Then** each grade is paired with its corresponding severity emoji per the grade-to-emoji mapping.
3. **Given** a full report with recommendations, **When** recommendations are rendered, **Then** each recommendation is prefixed with a severity emoji (🔴, 🟡, or 🟢).

---

### User Story 4 - Binary Resolution Portability in Full Mode (Priority: P2)

A developer runs `/gaze` in full mode in a project that is NOT the gaze repository. The classification step uses the same binary resolution strategy as the CRAP and quality steps — it does not attempt `go build ./cmd/gaze`.

**Why this priority**: This is a natural consequence of the consolidation (P1) and does not require additional implementation work if the consolidation is done correctly. It is a separate story because it validates a distinct user scenario.

**Independent Test**: Can be tested by running `/gaze` in a non-gaze project where gaze is installed via PATH and verifying no `go build ./cmd/gaze` error occurs.

**Acceptance Scenarios**:

1. **Given** a project that is not the gaze repository, **When** gaze is installed via PATH, **Then** `/gaze` full mode completes successfully including the classification step.
2. **Given** a project that is not the gaze repository, **When** gaze is NOT installed, **Then** the gaze-reporter's standard binary resolution failure message appears (not a `go build ./cmd/gaze` error).

---

### Edge Cases

- What happens if `gaze docscan` fails or returns no documents? The gaze-reporter uses mechanical-only classification results and includes a warning callout (`> ⚠️ ...`) noting that document-enhanced scoring was skipped.
- What happens if a user still has a stale `classify-docs.md` or `doc-classifier.md` from a previous `gaze init`? They remain on disk (gaze init does not delete files). The stale files are non-functional artifacts and are no longer distributed or supported.
- What happens if `/classify-docs` is typed manually? If the stale file exists, it runs (but may fail due to the hardcoded build and missing doc-classifier agent). If the file was removed, OpenCode reports an unknown command error.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The `/classify-docs` command file MUST be removed from `.opencode/command/classify-docs.md`.
- **FR-002**: The embedded copy of `classify-docs.md` MUST be removed from `internal/scaffold/assets/command/classify-docs.md`.
- **FR-003**: The gaze-reporter agent's full mode MUST directly orchestrate the classification workflow: run `gaze analyze --classify --format=json`, run `gaze docscan`, and apply document-signal scoring inline using the absorbed doc-classifier scoring model.
- **FR-004**: The gaze-reporter agent MUST use its existing binary resolution strategy (build from source → check PATH → go install) for all CLI commands in full mode, including the classification commands.
- **FR-005**: The doc-classifier agent file MUST be removed from `.opencode/agents/doc-classifier.md`. Its scoring logic (document signal weights, AI inference signals, contradiction penalties, confidence recalculation) MUST be inlined into the gaze-reporter agent prompt.
- **FR-005a**: The embedded copy of `doc-classifier.md` MUST be removed from `internal/scaffold/assets/agents/doc-classifier.md`.
- **FR-006**: The scaffold system (`scaffold.go`) MUST distribute exactly 2 files after the removal: `agents/gaze-reporter.md` and `command/gaze.md`.
- **FR-007**: The scaffold drift detection test (`TestEmbeddedAssetsMatchSource`) MUST be updated to expect 2 files instead of 4.
- **FR-008**: All scaffold tests that hardcode file counts (4 → 2) MUST be updated.
- **FR-009**: The gaze-reporter agent prompt MUST include explicit language asserting that emoji usage is a mandatory part of its output contract, not optional, and not subject to system-level suppression.
- **FR-010**: The gaze-reporter agent prompt MUST use a "sandwich" structure: a compact formatting example early in the prompt (immediately after the opening paragraph) to establish the pattern, AND the full canonical example retained at the end as final reinforcement. Both positions maximize LLM attention weight on the formatting contract.
- **FR-011**: The gaze-reporter agent prompt MUST use assertive override language (e.g., "THIS IS A FORMATTING REQUIREMENT, NOT A PREFERENCE") to resist system-level emoji suppression.
- **FR-012**: The gaze-reporter agent prompt's full mode section (lines 125-136) MUST be rewritten to replace the unreliable `/classify-docs` delegation with direct orchestration instructions.
- **FR-013**: The canonical example output in the gaze-reporter prompt MUST replace the `/classify-docs` reference (line 341 of gaze-reporter.md: "Run /classify-docs to incorporate document signals") since document-signal scoring is now performed inline during full mode.
- **FR-014**: The `/gaze` command file (`.opencode/command/gaze.md`) MUST remain unchanged.

### Key Entities

- **gaze-reporter agent**: The primary agent that runs CLI commands and formats reports. Updated to include inlined document-signal scoring (formerly in doc-classifier) and direct classification orchestration.
- **scaffold system**: The `embed.FS`-based file distribution system in `internal/scaffold/`. Updated to distribute 2 files instead of 4.

## Clarifications

### Session 2026-03-01

- Q: Which agent delegation mechanism should the gaze-reporter use for classification enrichment? → A: Inline the doc-classifier logic into gaze-reporter — the gaze-reporter reads docscan output and applies document signal scoring itself, no separate agent needed. The doc-classifier agent is removed entirely.
- Q: What structural approach should the gaze-reporter prompt use for the canonical example to ensure formatting compliance? → A: Sandwich — place a compact example early (after the opening paragraph) AND retain the full example at the end as final reinforcement. This maximizes LLM attention at both prompt boundaries.
- Q: If empirical testing shows prompt-level overrides do NOT reliably suppress the system emoji rule, what is the fallback? → A: Escalate to OpenCode — file an issue requesting agent prompts be exempt from the system emoji rule. Document as a known limitation with a defined escalation path rather than requiring a guaranteed outcome from prompt engineering.

## Assumptions

- OpenCode's system-level emoji suppression ("Only use emojis if the user explicitly requests it") can be overridden by sufficiently assertive language in the agent prompt. This is a hypothesis that must be validated empirically. If assertive prompt language does not reliably work, the fallback is to file an issue with OpenCode requesting that agent prompts be exempt from the system-level emoji rule. This is documented as a known risk, not a hard blocker.
- The doc-classifier's scoring model (document signal weights, AI inference signals, contradiction penalties, confidence recalculation with base-50 model) can be fully expressed as inline prompt instructions within the gaze-reporter agent without exceeding practical prompt length limits.
- Removing `classify-docs.md` and `doc-classifier.md` from the scaffold assets does not require a migration path for existing users. Users who previously ran `gaze init` will retain their stale copies until they manually delete them or run `gaze init --force` (which does not delete extra files).
- Technologies involved: Markdown files (agent prompts, command definitions) and Go code (scaffold system: `scaffold.go`, `scaffold_test.go`). No changes to analysis, taxonomy, classify, or any other `internal/` packages.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Neither `/classify-docs` command file nor `doc-classifier.md` agent file exists in `.opencode/` or `internal/scaffold/assets/` after implementation.
- **SC-002**: `gaze init` creates exactly 2 files (verified by `TestRun_CreatesFiles` passing with updated expectations).
- **SC-003**: The drift detection test (`TestEmbeddedAssetsMatchSource`) passes with 2 embedded assets matching their `.opencode/` counterparts.
- **SC-004**: All scaffold tests pass: `go test -race -count=1 ./internal/scaffold/...` exits 0.
- **SC-005**: The gaze-reporter agent's full mode prompt contains direct orchestration instructions for classification (run analyze --classify, run docscan, apply document-signal scoring inline) with no reference to `/classify-docs` or the doc-classifier agent.
- **SC-006**: The gaze-reporter agent prompt contains explicit emoji override language that asserts emoji usage is mandatory and not subject to system-level suppression.
- **SC-007**: The gaze-reporter prompt uses a sandwich structure: a compact formatting example appears early (after the opening paragraph) AND the full canonical example appears at the end of the prompt.
- **SC-008**: No references to `/classify-docs` or `doc-classifier` remain in any file under `.opencode/` or `internal/scaffold/assets/` after implementation.
- **SC-009**: The gaze-reporter agent prompt contains the doc-classifier's scoring model: document signal sources (readme, architecture_doc, specify_file, api_doc, other_md), weight ranges, AI inference signals, contradiction penalties, and confidence recalculation rules.
