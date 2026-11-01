# Specification Quality Checklist: Deterministic AI Report Rendering

**Purpose**: Validate specification completeness and quality before proceeding to planning
**Created**: 2026-11-01
**Feature**: [spec.md](../spec.md)

## Content Quality

- [x] No implementation details (languages, frameworks, APIs)
- [x] Focused on user value and business needs
- [x] Written for non-technical stakeholders
- [x] All mandatory sections completed

## Requirement Completeness

- [x] No [NEEDS CLARIFICATION] markers remain
- [x] Requirements are testable and unambiguous
- [x] Success criteria are measurable
- [x] Success criteria are technology-agnostic (no implementation details)
- [x] All acceptance scenarios are defined
- [x] Edge cases are identified
- [x] Scope is clearly bounded
- [x] Dependencies and assumptions identified

## Feature Readiness

- [x] All functional requirements have clear acceptance criteria
- [x] User scenarios cover primary flows
- [x] Feature meets measurable outcomes defined in Success Criteria
- [x] No implementation details leak into specification

## Notes

- Validation run 1 (2026-11-01) left one item open: FR-034, whether
  the scaffolded `.opencode/agents/gaze-reporter.md` becomes
  tool-owned. Validation run 2 (2026-11-01) closed it: the owner
  chose option A (tool-owned) during /uf.unleash clarify, as
  recorded in spec.md ## Clarifications.
- **Implementation-detail judgment**: the spec names CLI flags
  (`--format=markdown`, `--sections`, `--ai`), file paths
  (`.opencode/agents/gaze-reporter.md`), adapter names, and the
  structured answer's field names. For a CLI developer tool these
  are the user-facing contract, not internal implementation, so
  the item passes. Internal function names, package layout, and
  event-parsing mechanics are deliberately left to planning.
- **Success criteria**: reference observable outcomes (exit codes,
  byte-identical output, golden-file match, prompt length deltas,
  CI job behavior). Determinism-pack rule IDs (DE-002, DE-004,
  DE-007) are cited as governance traceability, not technology.
- **Settled decisions**: the owner's pre-made decisions are recorded
  under Background → "Decisions already made" and were not re-opened
  as clarifications.
- **Items to verify in planning** (recorded as assumptions, not
  ambiguities): opencode 1.19.0 event format (FR-016) and
  tool-permission keys (FR-014); concrete grade-rubric boundaries
  (FR-007, Assumptions); final validation bounds (FR-018,
  Assumptions).
- **Constitution alignment** (preliminary, formal check at plan):
  I Accuracy — no metric substitution, invalid output fails (FR-005,
  FR-019); II Minimal Assumptions — markdown output needs no AI
  provider (FR-025); III Actionable Output — byte-stable, comparable
  reports with machine-readable grades (FR-003, FR-010);
  IV Testability — seam, golden, round-trip, and incident regression
  tests required (FR-035–FR-038).
