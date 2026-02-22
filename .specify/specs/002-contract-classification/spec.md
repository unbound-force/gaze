# Feature Specification: Contractual Classification & Confidence Scoring

**Feature Branch**: `002-contract-classification`
**Created**: 2026-02-20
**Status**: Draft
**Input**: User description: "Identify what observable side effects
the test target is responsible for in the design of the application
it is part of, with confidence scoring informed by project
documentation"

## Clarifications

### Session 2026-02-21

- Q: What is the LLM integration model for document-enhanced classification (US2)? → A: Gaze is exclusive to OpenCode. Document-enhanced classification is implemented as an OpenCode agent/command, delegating AI inference to OpenCode's infrastructure. Gaze never calls an LLM directly. This is inherently model-agnostic.
- Q: What is the scope of caller dependency analysis for mechanical classification? → A: Same-module callers only. Gaze loads sibling packages within the Go module to find callers, without whole-program analysis.
- Q: Should `.gaze.yaml` be a general Gaze configuration file or only for document scanning? → A: General Gaze config file covering thresholds, document excludes, and future settings. CLI flags override file-based settings.
- Q: What does "exported in signature" mean as a classification signal? → A: Side effect is observable through the exported API surface. Graduated scoring where return type, parameter type, and receiver type visibility each contribute to the +20 max.
- Q: Can US1 (mechanical classification) ship as a standalone deliverable before US2? → A: No, all four user stories (US1-US4) are planned and implemented together as a single delivery for Spec 002.
- Q: How should the OpenCode agent integration work end-to-end? → A: The agent does the scoring. The OpenCode command runs `gaze analyze --classify`, feeds the mechanical JSON + scanned docs to the doc-classifier agent, and the agent produces final enhanced classifications with merged scores. No additional `gaze` subcommand needed for merging.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Mechanical Classification (Priority: P1)

Given a function's side effects (from Spec 001), Gaze classifies
each as contractual, incidental, or ambiguous using deterministic
signals that require no LLM and no project documentation. This
provides a baseline classification that works without any external
dependencies.

**Why this priority**: This is the minimum viable classification.
It works offline, is deterministic and reproducible, and provides
immediate value even without AI or docs. Mechanical signals alone
can resolve many cases with high confidence.

**Independent Test**: Can be tested by providing functions with
known contractual/incidental side effects (e.g., a method
implementing an interface where the interface defines the
contract) and verifying classifications and scores.

**Acceptance Scenarios**:

1. **Given** a method that implements an interface, **When** Gaze
   classifies its side effects, **Then** side effects declared in
   the interface signature (return values, parameter mutations
   documented by interface contract) are classified as
   `contractual` with confidence >= 85.
2. **Given** a function whose return value is used by all callers,
   **When** Gaze classifies, **Then** the return value is
   classified as `contractual` with confidence >= 80.
3. **Given** a function that writes to a logger but no caller
   depends on the log output, **When** Gaze classifies, **Then**
   the log write is classified as `incidental` with
   confidence < 50 (below the incidental threshold per FR-003).
4. **Given** a function with a side effect that has contradicting
   signals (e.g., exported but no callers use it), **When** Gaze
   classifies, **Then** it is classified as `ambiguous` with
   confidence between 50-79.
5. **Given** two identical functions analyzed on separate runs,
   **When** only mechanical signals are used, **Then** the
   classifications and confidence scores are identical
   (deterministic).

---

### User Story 2 - Document-Enhanced Classification (Priority: P2)

Gaze scans the project repository for documentation files (README,
architecture docs, design docs, .specify files, all .md files) and
uses an OpenCode agent/command to extract design-intent signals that
refine the confidence scores from US1. Gaze delegates all AI
inference to OpenCode's infrastructure and never calls an LLM
directly, making this inherently model-agnostic.

**Why this priority**: Documents carry the richest design-intent
signal but require an LLM to interpret. This is the primary
differentiator of Gaze — understanding what a function is
*supposed* to do based on the project's own documentation.

**Independent Test**: Can be tested by providing a project with
explicit architectural documentation that declares a component's
responsibilities, analyzing a function in that component, and
verifying that document signals shift the confidence score in
the correct direction.

**Acceptance Scenarios**:

1. **Given** a repository with an `architecture.md` that states
   "The store layer is responsible for persistence", **When** Gaze
   classifies side effects of a store function, **Then** the
   database write side effect gains confidence from the document
   signal, and the confidence score increases.
2. **Given** a repository with a `.specify/specs/*/spec.md` that
   states "FR-003: System MUST return validation errors for
   invalid input", **When** Gaze classifies a validation
   function's error return, **Then** the error return is
   classified as `contractual` with boosted confidence.
3. **Given** a repository with a README that describes the
   project's purpose, **When** Gaze classifies a function whose
   side effects align with that purpose, **Then** the alignment
   is reflected in the confidence score.
4. **Given** a document exclude configuration that excludes
   `vendor/` and `CHANGELOG.md`, **When** Gaze scans for
   documents, **Then** files matching the exclude patterns are
   not read or processed.
5. **Given** no documentation exists in the repository, **When**
   Gaze classifies with document-enhanced mode, **Then** Gaze
   gracefully falls back to mechanical-only classification and
   reports that no document signals were found.

---

### User Story 3 - Document Exclude Configuration (Priority: P3)

Users can configure which files and directories are excluded from
document scanning via a configuration file, preventing noise from
irrelevant markdown files (changelogs, vendor docs, generated docs).

**Why this priority**: Practical usability for large repos. Without
excludes, document scanning could process hundreds of irrelevant
files, increasing latency and potentially introducing noise into
confidence scores.

**Independent Test**: Can be tested by creating a config with
excludes, placing documentation in both included and excluded paths,
and verifying only the included paths influence classification.

**Acceptance Scenarios**:

1. **Given** a `.gaze.yaml` config with
   `exclude_docs: ["vendor/**", "CHANGELOG.md"]`, **When** Gaze
   scans for documents, **Then** files in `vendor/` and
   `CHANGELOG.md` are skipped.
2. **Given** no `.gaze.yaml` exists, **When** Gaze scans for
   documents, **Then** a built-in default exclude list is used
   (vendor/, node_modules/, .git/, CHANGELOG.md,
   CONTRIBUTING.md, CODE_OF_CONDUCT.md).
3. **Given** a `.gaze.yaml` with `include_docs: ["docs/**",
   "README.md"]`, **When** Gaze scans, **Then** only matching
   files are processed, overriding the default full-repo scan.

---

### User Story 4 - Confidence Score Breakdown (Priority: P4)

Each classification includes a detailed breakdown showing which
signals contributed to the confidence score and by how much,
enabling users to understand and audit the classification.

**Why this priority**: Transparency and trust. If a user disagrees
with a classification, they need to understand why Gaze made
that decision. Aligns with Constitution Principle I (Accuracy).

**Independent Test**: Can be tested by analyzing a function with
multiple contributing signals and verifying the breakdown sums
to the reported confidence score.

**Acceptance Scenarios**:

1. **Given** a classified side effect, **When** the user requests
   verbose output, **Then** each contributing signal is listed
   with its source, weight, and reasoning.
2. **Given** a side effect with both mechanical and document
   signals, **When** breakdown is displayed, **Then** both
   signal categories are clearly separated.

---

### Edge Cases

- What happens when mechanical signals contradict document signals?
  The contradiction MUST be reported in the breakdown. Confidence
  score MUST reflect the conflict (lower overall confidence). The
  classification MUST default to `ambiguous` if the contradiction
  is strong.
- What happens when document-enhanced classification is unavailable
  (OpenCode not running, user skips the command, agent error)?
  Gaze MUST fall back to mechanical-only classification and report
  a warning.
- How does Gaze handle very large documentation sets (hundreds of
  .md files)? Gaze MUST prioritize documents by proximity to the
  target function: same package > same module > project root.
  Document scanning MUST complete within a configurable timeout.
- What happens when a function has no detectable signals at all
  (no interfaces, no callers, no docs)? The base confidence
  score is 50 (neutral). Signals adjust from this base: positive
  signals push toward contractual, negative/incidental signals
  push toward incidental. With zero signals, all side effects
  MUST be classified as `ambiguous` with confidence exactly 50
  and a note explaining insufficient signal.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Gaze MUST classify each side effect (from Spec 001
  output) as `contractual`, `incidental`, or `ambiguous`.
- **FR-002**: Each classification MUST include a confidence score
  from 0-100.
- **FR-003**: The default classification thresholds MUST be:
  >= 80 = contractual, 50-79 = ambiguous, < 50 = incidental.
  Thresholds MUST be configurable via `.gaze.yaml` and
  overridable via CLI flags.
- **FR-004**: Mechanical signal sources MUST include:
  - Interface satisfaction (+30 max)
  - Exported API surface visibility (+20 max) — graduated: the
    side effect is observable through exported return types,
    exported parameter types, or exported receiver types; each
    visibility dimension contributes independently
  - Caller dependency analysis (+15 max) — scoped to same-module
    callers only (sibling packages within the Go module); does NOT
    require whole-program or cross-module analysis
  - Naming convention match (+10 max for standard prefixes such as
    Get*, Fetch*, Save*, Delete*, Handle*, etc.); **exception**: `Err*`
    sentinel error variables receive +30 because they are unambiguously
    contractual by Go convention (exported, matched by callers, no other
    signals available for package-level `var` declarations). The +30
    weight is chosen so that a bare sentinel with no other signals reaches
    the default contractual threshold (50 + 30 = 80).
  - Godoc comment declares behavior (+15 max)
- **FR-005**: Document signal sources MUST include:
  - README.md (+15 max)
  - Architecture/design documents (+20 max)
  - .specify files (specs, plans, contracts) (+20 max)
  - API/user-facing documentation (+15 max)
  - Any other .md files in repo (+10 max)
- **FR-006**: AI architectural inference signals MUST include:
  - Design pattern recognition (+15 max)
  - Layer/boundary analysis (+15 max)
  - Cross-document corroboration (+10 max)
- **FR-007**: Contradicting signals MUST apply a penalty
  (up to -20).
- **FR-008**: Gaze MUST support a `.gaze.yaml` configuration file
  as the general Gaze configuration file. It MUST support document
  exclude/include patterns, classification thresholds, and future
  settings. CLI flags MUST override file-based settings.
- **FR-009**: The default exclude list MUST include at minimum:
  `vendor/**`, `node_modules/**`, `.git/**`, `testdata/**`,
  `CHANGELOG.md`, `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`,
  `LICENSE`, `LICENSE.md`.
- **FR-010**: Gaze MUST function without an LLM by using
  mechanical signals only. Document-enhanced classification is
  invoked via an OpenCode agent/command and MUST be opt-in or
  automatic-with-fallback. Gaze MUST NOT embed any LLM SDK or
  call any LLM provider directly.
- **FR-011**: Mechanical-only classification MUST be fully
  deterministic — identical inputs produce identical outputs.
- **FR-012**: Document scanning MUST follow a priority order:
  1. Target function's godoc comment
  2. Package-level doc.go
  3. .specify/ files
  4. docs/ directory
  5. README.md
  6. Other .md files in repo
- **FR-013**: Classification output MUST extend the Spec 001
  AnalysisResult JSON schema (additive, not breaking).
- **FR-014**: Each classification MUST include a signal breakdown
  accessible via verbose/detailed output mode.
- **FR-015**: Gaze MUST support Go 1.24+ (consistent with
  Spec 001).

### Key Entities

- **Classification**: The contractual/incidental/ambiguous label
  for a single side effect. Attached as an optional field on the
  existing `SideEffect` struct (pointer, omitted when nil).
  Attributes: label (ClassificationLabel enum), confidence
  (int, 0-100), signals ([]Signal), reasoning (string).
- **Signal**: A single piece of evidence contributing to the score.
  Attributes: source (string: interface, caller, naming, godoc,
  readme, architecture_doc, specify_file, api_doc, other_md,
  ai_pattern, ai_layer, ai_corroboration, contradiction),
  weight (int, can be negative), source_file (string, omitempty),
  excerpt (string, omitempty), reasoning (string, omitempty).
  Detail fields are omitted from JSON when empty (non-verbose).
- **GazeConfig**: General configuration loaded from `.gaze.yaml`.
  Attributes: classification_thresholds (contractual, ambiguous,
  incidental boundaries), exclude_patterns ([]glob), include_patterns
  ([]glob, optional override), doc_scan_timeout (duration). CLI
  flags override all fields.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Mechanical-only classification correctly identifies
  >= 90% of contractual side effects as contractual (not
  incidental) on a benchmark suite of 30+ Go functions with
  known contracts.
- **SC-002**: Document-enhanced classification improves accuracy
  by >= 10 percentage points over mechanical-only on the same
  benchmark suite when relevant documentation exists.
- **SC-003**: False contractual rate (incidental effects classified
  as contractual) is < 15% on the benchmark suite.
- **SC-004**: Mechanical-only classification completes within the
  Spec 001 analysis time budget (no additional latency beyond
  static analysis).
- **SC-005**: Document scanning and AI classification adds < 10s
  for a typical project (< 50 .md files, < 100KB total).
- **SC-006**: When no LLM is available, Gaze produces a complete
  classification with zero errors (graceful degradation verified).
- **SC-007**: Confidence score breakdown sums correctly and each
  signal weight is within documented bounds.
