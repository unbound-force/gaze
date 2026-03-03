# Feature Specification: Agent Context Reduction

**Feature Branch**: `016-agent-context-reduction`
**Created**: 2026-03-02
**Status**: Complete
**Input**: User description: "Reduce gaze-reporter agent prompt context window usage by externalizing canonical example output and document-enhanced classification scoring model into on-demand local files, and deduplicating repeated quadrant label patterns"

## Context

The gaze-reporter agent prompt consumes approximately 17,775 bytes (~4,443 tokens) every time the `/gaze` command is invoked. This is loaded into the conversation context regardless of which mode (crap, quality, full) the user requests. Analysis shows that 63.9% of the prompt consists of three sections that could be loaded on-demand or deduplicated:

1. **Canonical Example Output** (3,367 bytes, 18.9%) — a 65-line fictional report used as a formatting reference. Always loaded but only needed as a formatting guide.
2. **Document-Enhanced Classification scoring model** (2,395 bytes, 13.5%) — signal tables and thresholds used only in full mode when docscan returns documents. Never needed for crap or quality modes.
3. **Repeated quadrant labels** (~200 bytes) — Q1-Q4 emoji labels appear in 3 separate locations.

The prompt also has a scaffold copy that must stay in sync.

## User Scenarios & Testing

### User Story 1 - On-Demand Canonical Example (Priority: P1)

A developer runs `/gaze crap ./...` to check CRAP scores. The gaze-reporter agent loads its prompt into the context window. The canonical example output (3,367 bytes) is not embedded in the prompt — instead, the agent reads it from a local file only when it needs formatting reference. This saves ~3,100 bytes from the always-on prompt cost.

**Why this priority**: The canonical example is the single largest contiguous block in the prompt (18.9%). Moving it to an on-demand file produces the biggest context savings with minimal complexity. Every invocation of `/gaze` benefits.

**Independent Test**: Run `/gaze crap ./...` and verify the report output still follows the correct formatting pattern (emoji markers, table structure, metadata line, quadrant distribution). Compare against the canonical example to confirm formatting fidelity is preserved.

**Acceptance Scenarios**:

1. **Given** the gaze-reporter agent prompt without the inline canonical example, **When** the agent is invoked via `/gaze crap ./...`, **Then** the agent reads the canonical example from a local file and produces correctly formatted output matching the expected emoji markers, table structure, and metadata format.
2. **Given** the canonical example file exists at the expected path, **When** the agent reads it during report generation, **Then** the file contents match the original canonical example that was previously inline in the prompt.
3. **Given** a new project where `gaze init` is run, **When** the scaffolding completes, **Then** the canonical example file is created alongside the agent prompt and command files.

---

### User Story 2 - On-Demand Scoring Model (Priority: P2)

A developer runs `/gaze quality ./...` to check test quality metrics. The document-enhanced classification scoring model (2,395 bytes) is not needed for this mode. By externalizing it to a local file, the prompt is smaller for crap and quality modes. When the developer later runs `/gaze` (full mode) and docscan returns documents, the agent reads the scoring model file on demand.

**Why this priority**: The scoring model is only relevant in full mode with documentation present. Externalizing it saves ~2,100 bytes for the two most common modes (crap and quality) with zero formatting risk, since the content is loaded identically when needed.

**Independent Test**: Run `/gaze quality ./...` and confirm the scoring model is not loaded. Then run `/gaze` (full mode) and verify classification output correctly applies document-enhanced scoring with proper signal weights and thresholds.

**Acceptance Scenarios**:

1. **Given** the gaze-reporter agent prompt without the inline scoring model, **When** the agent runs in crap mode, **Then** the scoring model file is not read (no unnecessary tool calls), and the CRAP report is produced correctly.
2. **Given** the agent runs in full mode and docscan returns documentation files, **When** the agent needs to apply document-enhanced classification, **Then** it reads the scoring model from the local file and applies signal weights, AI inference signals, contradiction penalties, and classification thresholds correctly.
3. **Given** a new project where `gaze init` is run, **When** the scaffolding completes, **Then** the scoring model file is created alongside the other scaffolded files.

---

### User Story 3 - Deduplicated Quadrant Labels (Priority: P3)

The quadrant emoji labels (Q1-Q4) currently appear in three separate locations within the prompt: the Quick Reference Example, the CRAP Mode section, and the Emoji Vocabulary table. One of these three occurrences is replaced with a cross-reference, reducing redundancy by ~200 bytes.

**Why this priority**: Small savings with very low risk. The labels remain in two locations (Quick Reference Example and Emoji Vocabulary), providing sufficient coverage for the agent to produce correct output.

**Independent Test**: Run `/gaze crap ./...` and verify the quadrant distribution table uses the correct emoji-prefixed labels (Q1 Safe, Q2 Complex But Tested, Q4 Dangerous, Q3 Needs Tests) with proper emoji markers.

**Acceptance Scenarios**:

1. **Given** the deduplicated prompt with quadrant labels removed from the CRAP Mode section, **When** the agent produces a CRAP report with quadrant distribution, **Then** all four quadrant rows use the correct emoji-prefixed labels matching the Quick Reference Example.

---

### Edge Cases

- What happens when the canonical example file is missing or unreadable? The agent should produce output using the Quick Reference Example in the prompt as its formatting guide and include a warning that the full example could not be loaded.
- What happens when the scoring model file is missing during full mode? The agent should skip document-enhanced classification and use mechanical-only results, with a warning callout.
- What happens when `gaze init` is run in a project that already has older scaffold files without the new reference files? The new files should be created. Reference files whose content differs from the embedded scaffold version should be overwritten with the latest version. Existing agent and command files follow their current skip-if-present behavior.

## Requirements

### Functional Requirements

- **FR-001**: The gaze-reporter agent prompt MUST NOT contain the canonical example output inline. It MUST instead contain an instruction directing the agent to read the example from a local file.
- **FR-002**: The canonical example MUST be stored in a local file accessible to the agent via the Read tool, at a path under `.opencode/`.
- **FR-003**: The gaze-reporter agent prompt MUST NOT contain the Document-Enhanced Classification scoring model inline. It MUST instead contain an instruction directing the agent to read the scoring model from a local file when running in full mode with docscan results.
- **FR-004**: The scoring model MUST be stored in a local file accessible to the agent via the Read tool, at a path under `.opencode/`.
- **FR-005**: The quadrant emoji labels in the CRAP Mode section of the prompt MUST be replaced with a cross-reference to the Quick Reference Example, removing one of the three redundant listings.
- **FR-006**: The `gaze init` scaffolding MUST create the new reference files (canonical example and scoring model) alongside the existing agent and command files.
- **FR-007**: The scaffold copies MUST remain byte-identical to the live copies under `.opencode/`.
- **FR-008**: The agent prompt instructions for reading external files MUST be explicit and mandatory (e.g., "Read this file before producing your first report").
- **FR-009**: The agent MUST gracefully handle missing reference files — using inline fallbacks (Quick Reference Example) and warning callouts rather than failing silently.
- **FR-010**: When `gaze init` runs and a reference file already exists with content that differs from the embedded scaffold version, it MUST overwrite the file with the latest version. Agent and command files retain their existing skip-if-present behavior.

### Key Entities

- **Agent Prompt**: The gaze-reporter system prompt loaded into the context window on every `/gaze` invocation. The primary artifact being optimized.
- **Reference File**: A local file containing content externalized from the prompt, read on-demand by the agent via tool calls. Two instances: canonical example and scoring model.
- **Scaffold Asset**: An embedded copy of a reference file distributed via `gaze init` for new projects.

## Clarifications

### Session 2026-03-02

- Q: When `gaze init` runs and reference files already exist, should it skip them (like agent/command files) or overwrite them? → A: Overwrite reference files when content differs from the embedded scaffold version. Agent and command files keep their existing skip-if-present behavior.

## Success Criteria

### Measurable Outcomes

- **SC-001**: The always-on agent prompt size is reduced by at least 25% (from ~17,775 bytes to no more than 13,300 bytes).
- **SC-002**: CRAP-mode and quality-mode invocations do not load the scoring model file (zero unnecessary tool calls for mode-irrelevant content).
- **SC-003**: Full-mode invocations with docscan results load the scoring model and produce output with correct document-enhanced classification matching pre-change behavior.
- **SC-004**: Report formatting fidelity is preserved — emoji markers, table structure, metadata format, quadrant labels, and grade-to-emoji mapping match the pre-change output for identical input data.
- **SC-005**: The `gaze init` command scaffolds the new reference files into the correct directories without requiring user intervention.
- **SC-006**: The scaffold copies and live `.opencode/` copies remain byte-identical after changes.
