<!--
  SYNC IMPACT REPORT
  ==================
  Version change: (none) → 1.0.0 (initial ratification)

  Modified principles:
    - [PRINCIPLE_1_NAME] → I. Accuracy
    - [PRINCIPLE_2_NAME] → II. Minimal Assumptions
    - [PRINCIPLE_3_NAME] → III. Actionable Output
    - [PRINCIPLE_4_NAME] → (removed, not needed)
    - [PRINCIPLE_5_NAME] → (removed, not needed)

  Added sections:
    - Development Workflow (Section 3)
    - Governance (filled from template)

  Removed sections:
    - [SECTION_2_NAME] (user elected to skip)
    - Principles 4 and 5 (user chose 3 principles)

  Templates requiring updates:
    ✅ .specify/templates/plan-template.md — no changes needed;
       Constitution Check section is generic and will align at plan time.
    ✅ .specify/templates/spec-template.md — no changes needed;
       requirements format already uses MUST/SHOULD language.
    ✅ .specify/templates/tasks-template.md — no changes needed;
       task phases are feature-driven, not principle-specific.
    ✅ .specify/templates/checklist-template.md — no changes needed.
    ✅ .specify/templates/agent-file-template.md — no changes needed.
    ✅ No command files in .specify/templates/commands/ (directory absent).
    ✅ README.md — single-line placeholder; no principle refs to update.

  Deferred TODOs:
    - RATIFICATION_DATE set to today (first adoption). Update if a
      different ceremonial date is preferred.
-->

# Gaze Constitution

## Core Principles

### I. Accuracy

Gaze MUST correctly identify all observable side effects produced by a
test target. An observable side effect includes return values, mutations
to shared state, emitted events, I/O operations, and any other
externally detectable change.

- Every reported "unasserted change" MUST correspond to a real
  observable side effect; false positives erode trust and MUST be
  treated as bugs.
- Every actual observable side effect that goes unreported is a false
  negative; false negatives MUST be tracked, measured, and driven
  toward zero.
- Accuracy claims MUST be backed by automated regression tests that
  cover known-good and known-bad assertion scenarios.

**Rationale**: The entire value proposition of Gaze depends on users
trusting its output. Inaccurate results — in either direction — make
the tool worse than useless.

### II. Minimal Assumptions

Gaze MUST operate with the fewest possible assumptions about the host
project's language, test framework, or coding style.

- Analysis MUST NOT require users to annotate or restructure their
  existing test code unless strictly necessary and clearly documented.
- When assumptions are unavoidable (e.g., a supported language list),
  they MUST be explicit in documentation and enforced at analysis
  entry points — never silently ignored.
- New language or framework support MUST NOT break or alter behavior
  for already-supported environments.

**Rationale**: A test-quality tool that demands significant setup or
convention changes creates friction that discourages adoption. Gaze
earns trust by working with what already exists.

### III. Actionable Output

Every piece of output Gaze produces MUST guide the user toward a
concrete improvement in their test suite.

- Reports MUST identify the specific test, the specific test target,
  and the specific unasserted observable change — not just aggregate
  scores.
- Output formats MUST support both human-readable (terminal/CI logs)
  and machine-readable (JSON) consumption.
- Metrics MUST be comparable across runs so users can measure progress
  over time.

**Rationale**: Metrics without actionable detail are vanity numbers.
Gaze exists to help developers write better tests, and that requires
telling them exactly what to fix.

## Development Workflow

- **Branching**: All work MUST occur on feature branches. Direct
  commits to the main branch are prohibited except for trivial
  documentation fixes.
- **Code Review**: Every pull request MUST receive at least one
  approving review before merge.
- **Continuous Integration**: The CI pipeline MUST pass (build, lint,
  tests) before a pull request is eligible for merge.
- **Releases**: Follow semantic versioning (MAJOR.MINOR.PATCH).
  Breaking changes to public APIs or analysis behavior require a
  MAJOR bump.
- **Commit Messages**: Use conventional commit format
  (`type: description`) to enable automated changelog generation.

## Governance

This constitution is the highest-authority document for the Gaze
project. All development practices, pull request reviews, and
architectural decisions MUST be consistent with the principles defined
above.

- **Amendments**: Any change to this constitution MUST be proposed via
  pull request, reviewed, and approved before merge. The amendment
  MUST include a migration plan if it alters or removes existing
  principles.
- **Versioning**: The constitution follows semantic versioning:
  - MAJOR: Principle removal or incompatible redefinition.
  - MINOR: New principle or materially expanded guidance.
  - PATCH: Clarifications, wording, or non-semantic refinements.
- **Compliance Review**: At each planning phase (spec, plan, tasks),
  the Constitution Check gate MUST verify that the proposed work
  aligns with all active principles.

**Version**: 1.0.0 | **Ratified**: 2026-02-20 | **Last Amended**: 2026-02-20
