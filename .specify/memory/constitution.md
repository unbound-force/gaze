<!--
  SYNC IMPACT REPORT
  ==================
  Version change: 1.3.0 → 1.4.0 (MINOR: Spec-First Development
    replaced by Tier-Routed Workflow)
  Amendment date: 2026-11-25

  Modified sections:
    - Development Workflow: "Spec-First Development" (every
      non-trivial change needs Speckit or OpenSpec) replaced by
      "Tier-Routed Workflow" — Speckit, OpenSpec, or Quick chosen
      per change, matching /uf.unleash tier routing. Risk floors
      keep security, auth, data-loss, and API-breaking changes out
      of Quick. Branch naming gains `quick/<name>`.

  Unchanged principles: all

  Previous version history:
    - 1.3.0 (2026-03-15): Review Council Gate added
    - 1.2.0 (2026-03-15): Added Spec-First Development requirement
    - 1.1.0 (2026-03-05): Added Principle IV: Testability
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

- **Tier-Routed Workflow**: Every change MUST follow a workflow
  tier chosen for that change's scope and risk; no single tier is
  mandatory. Speckit (`specs/NNN-<name>/`) for strategic work: 3+
  user stories, cross-cutting or cross-repo. OpenSpec
  (`openspec/changes/<name>/`) for tactical work: 1-2 stories,
  bounded blast radius. Quick (no spec artifacts) for trivial
  fixes, chores, and docs changes where ceremony exceeds value.
  `/uf.unleash` routes a change with no spec through the
  `tier-routing` rubric; manual work applies the same rubric.
  Risk floors escalate only: security-sensitive code,
  authentication or authorization flows, data-loss paths, or a
  public API break MUST use OpenSpec at minimum, never Quick.
  When signals conflict between adjacent tiers, choose the
  heavier one. Every tier, Quick included, keeps the full
  quality gates (CI, code review). Constitution amendments follow
  the Governance section.
- **Branching**: All work MUST occur on feature branches. Direct
  commits to the main branch are prohibited except for trivial
  documentation fixes. Speckit branches: `NNN-<name>`. OpenSpec
  branches: `opsx/<name>`. Quick branches: `quick/<name>`.
- **Code Review**: Every pull request MUST receive at least one
  approving review before merge.
- **Review Council Gate**: Before submitting a pull request, agents
  MUST run the `/review-council` command and receive an APPROVE
  verdict from all four reviewers (Adversary, Architect, Guard,
  Tester). Any REQUEST CHANGES findings MUST be resolved before
  PR submission. There MUST be minimal to no code changes between
  the council's APPROVE and the PR submission — the council reviews
  the code that will be submitted, not a draft that changes afterward.
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

**Version**: 1.4.0 | **Ratified**: 2026-02-20 | **Last Amended**: 2026-11-25
