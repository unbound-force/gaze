# Feature Specification: Contractual Classification & Confidence Scoring

**Feature Branch**: `002-contract-classification`
**Created**: 2026-02-20
**Status**: Draft
**Input**: User description: "Identify what observable side effects
the test target is responsible for in the design of the application
it is part of, with confidence scoring informed by project
documentation"

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
   confidence >= 60.
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
uses an AI agent to extract design-intent signals that refine the
confidence scores from US1.

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
- What happens when the AI agent is unavailable (no API key, rate
  limited, network error)? Gaze MUST fall back to mechanical-only
  classification and report a warning.
- How does Gaze handle very large documentation sets (hundreds of
  .md files)? Gaze MUST prioritize documents by proximity to the
  target function: same package > same module > project root.
  Document scanning MUST complete within a configurable timeout.
- What happens when a function has no detectable signals at all
  (no interfaces, no callers, no docs)? All side effects MUST
  be classified as `ambiguous` with confidence ~50 and a note
  explaining insufficient signal.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Gaze MUST classify each side effect (from Spec 001
  output) as `contractual`, `incidental`, or `ambiguous`.
- **FR-002**: Each classification MUST include a confidence score
  from 0-100.
- **FR-003**: The default classification thresholds MUST be:
  >= 80 = contractual, 50-79 = ambiguous, < 50 = incidental.
  Thresholds MUST be configurable.
- **FR-004**: Mechanical signal sources MUST include:
  - Interface satisfaction (+30 max)
  - Exported in signature (+20 max)
  - Caller dependency analysis (+15 max)
  - Naming convention match (+10 max)
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
  for document exclude/include patterns.
- **FR-009**: The default exclude list MUST include at minimum:
  `vendor/**`, `node_modules/**`, `.git/**`, `testdata/**`,
  `CHANGELOG.md`, `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`,
  `LICENSE`, `LICENSE.md`.
- **FR-010**: Gaze MUST function without an LLM by using
  mechanical signals only. AI-enhanced classification MUST be
  opt-in or automatic-with-fallback.
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
  for a single side effect. Attributes: label (enum), confidence
  (0-100), signals ([]Signal), reasoning (string, AI-generated
  or template-based).
- **ConfidenceScore**: Numeric 0-100 value. Composed of weighted
  signals. Attributes: total (int), breakdown
  ([]SignalContribution).
- **Signal**: A single piece of evidence contributing to the score.
  Attributes: source (enum: interface, caller, naming, godoc,
  readme, architecture_doc, specify_file, api_doc, other_md,
  ai_pattern, ai_layer, ai_corroboration, contradiction),
  weight (int, can be negative), source_file (string, path),
  excerpt (string, relevant text), reasoning (string).
- **ExcludeConfig**: Document scanning exclusion rules. Attributes:
  exclude_patterns ([]glob), include_patterns ([]glob, optional
  override), timeout (duration).
- **ClassifiedAnalysisResult**: Extends AnalysisResult from Spec 001
  with classifications per side effect.

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
