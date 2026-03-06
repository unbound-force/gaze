<!--
  SYNC IMPACT REPORT
  ==================
  Version change: 1.0.0 → 1.1.0 (MINOR: new principle added)
  Amendment date: 2026-03-05
  Feature: 017-testing-persona

  Added principles:
    - IV. Testability (dual scope: Gaze internals + user codebase analysis)

  Unchanged principles:
    - I. Accuracy
    - II. Minimal Assumptions
    - III. Actionable Output

  Unchanged sections:
    - Development Workflow
    - Governance

  Templates requiring updates:
    ✅ .specify/templates/plan-template.md — no changes needed;
       Constitution Check section is generic and evaluates all active
       principles dynamically.
    ✅ .specify/templates/spec-template.md — no changes needed;
       requirements format already uses MUST/SHOULD language.
    ✅ .specify/templates/tasks-template.md — no changes needed;
       task phases are feature-driven, not principle-specific.
    ✅ .specify/templates/checklist-template.md — no changes needed.
    ✅ .specify/templates/agent-file-template.md — no changes needed.
    ✅ No command files in .specify/templates/commands/ (directory absent).
    ✅ README.md — single-line placeholder; no principle refs to update.

  Previous version history:
    - 1.0.0 (2026-02-20): Initial ratification with 3 principles
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

### IV. Testability

Every function Gaze analyzes, and every function within Gaze itself,
MUST be testable in isolation without requiring external services or
shared mutable state.

- Test contracts MUST verify observable side effects (return values,
  state mutations, I/O operations), not implementation details.
- Coverage strategy (unit vs. integration vs. e2e, with targets) MUST
  be specified in the implementation plan for all new code.
- Coverage ratchets MUST be enforced by automated tests; coverage
  regression MUST be treated as a test failure.
- Missing coverage strategy in a spec or plan is a CRITICAL-severity
  finding and MUST be resolved before implementation begins.

**Rationale**: Gaze is a test-quality tool. If Gaze's own tests are
poorly structured, it undermines the credibility of its assessments.
Testability is a first-class governance concern because untestable
code cannot be reliably verified, and unverified code cannot be
trusted — by users or by Gaze itself.

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

**Version**: 1.1.0 | **Ratified**: 2026-02-20 | **Last Amended**: 2026-03-05
