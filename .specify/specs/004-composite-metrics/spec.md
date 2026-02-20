# Feature Specification: Composite Quality Metrics & Self-Analysis

**Feature Branch**: `004-composite-metrics`
**Created**: 2026-02-20
**Status**: Draft
**Input**: User description: "Measure this project's CRAPload to
manage code quality, compare Gaze and CRAP metrics for insights,
dogfooding via CI and manual command"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - CRAP Score Per Function (Priority: P1)

A developer runs `gaze crap ./...` and gets a per-function CRAP
score combining cyclomatic complexity with `go test` line coverage.
This is the classic CRAP4J metric adapted for Go. The formula is:
`CRAP(m) = comp(m)^2 * (1 - cov(m)/100)^3 + comp(m)` where
`comp` is cyclomatic complexity and `cov` is line coverage
percentage.

**Why this priority**: CRAP is the industry-standard baseline. It
MUST work independently before Gaze-specific metrics can be
compared against it. Also immediately useful for managing Gaze's
own code quality without waiting for Specs 002-003.

**Independent Test**: Can be tested by generating a coverage
profile for a known Go package, running CRAP analysis, and
verifying the scores match hand-computed values from the formula.

**Acceptance Scenarios**:

1. **Given** a function with complexity 5 and 0% coverage, **When**
   Gaze computes CRAP, **Then** CRAP = 5^2 * (1-0)^3 + 5 = 30.
2. **Given** a function with complexity 5 and 100% coverage,
   **When** Gaze computes CRAP, **Then** CRAP = 5^2 * 0^3 + 5 = 5.
3. **Given** a function with complexity 10 and 50% coverage,
   **When** Gaze computes CRAP, **Then**
   CRAP = 100 * 0.125 + 10 = 22.5.
4. **Given** a coverage profile and a `--crap-threshold=15` flag,
   **When** Gaze reports, **Then** functions with CRAP >= 15 are
   flagged and the CRAPload count is reported.
5. **Given** no coverage profile exists, **When** Gaze runs CRAP
   analysis, **Then** Gaze runs `go test -coverprofile`
   automatically and reports a warning that all functions default
   to 0% coverage if tests fail.

---

### User Story 2 - GazeCRAP Score (Priority: P2)

A developer runs `gaze crap --gaze ./...` and gets a GazeCRAP
score per function: the CRAP formula with Gaze's contract coverage
substituted for line coverage. The formula is:
`GazeCRAP(m) = comp(m)^2 * (1 - contractCov(m))^3 + comp(m)`
where `contractCov` is the contract coverage ratio (0.0-1.0) from
Spec 003.

This measures whether complex functions have tests that verify
their *contractual* side effects, not just execute their code
paths. A function with 100% line coverage but 0% contract
coverage looks safe by traditional CRAP but is revealed as risky
by GazeCRAP.

**Why this priority**: This is Gaze's novel contribution — the
metric that no other tool provides. It depends on Specs 001-003
(contract coverage) being available.

**Independent Test**: Can be tested by analyzing a function with
known complexity and known contract coverage, verifying the
GazeCRAP formula produces the correct score.

**Acceptance Scenarios**:

1. **Given** a function with complexity 10, 90% line coverage, but
   only 25% contract coverage, **When** Gaze computes both CRAP
   and GazeCRAP, **Then** CRAP = 10 (looks safe) but
   GazeCRAP = 100 * 0.422 + 10 = 52.2 (reveals risk). The
   function looks fine by traditional metrics but is under-tested
   on its contractual obligations.
2. **Given** a function with complexity 3, 50% line coverage, but
   100% contract coverage, **When** Gaze computes both, **Then**
   CRAP = 4.125 (moderate) but GazeCRAP = 3 (excellent). The
   test does not run all code paths but verifies every
   contractual side effect.
3. **Given** both `--gaze` and default mode, **When** output is
   produced, **Then** both CRAP and GazeCRAP appear side-by-side
   per function for comparison.

---

### User Story 3 - Quadrant Report & Correlation (Priority: P3)

Gaze produces a quadrant analysis classifying each function into
risk categories based on CRAP and GazeCRAP, plus a raw data export
for external analysis (scatter plots, statistical correlation).

The four quadrants are:
- **Q1 — Safe**: Low CRAP, low GazeCRAP. Simple and well-tested.
- **Q2 — Complex but contract-tested**: High CRAP, low GazeCRAP.
  Complex code but tests verify contracts. May benefit from
  refactoring but tests are solid.
- **Q3 — Simple but under-specified**: Low CRAP, high GazeCRAP.
  Simple code but tests do not verify the contract. Easy to fix.
- **Q4 — Dangerous**: High CRAP, high GazeCRAP. Complex AND tests
  do not verify the contract. Highest priority for attention.

**Why this priority**: Analysis and visualization layer on top of
the raw metrics. Valuable for insight but not blocking for metric
computation.

**Independent Test**: Can be tested by providing functions spanning
all four quadrants and verifying correct classification.

**Acceptance Scenarios**:

1. **Given** functions analyzed with both CRAP and GazeCRAP,
   **When** quadrant report runs, **Then** each function is
   classified into Q1-Q4 based on its CRAP score vs
   `--crap-threshold` and its GazeCRAP score vs
   `--gaze-crap-threshold`.
2. **Given** `--format=json`, **When** correlation data is
   exported, **Then** output includes per-function: name,
   complexity, line_coverage, contract_coverage, crap,
   gaze_crap, quadrant classification.
3. **Given** `--format=text`, **When** quadrant report is
   displayed, **Then** functions are grouped by quadrant with
   counts and a summary.
4. **Given** a function with CRAP=20 (above crap-threshold=15)
   and GazeCRAP=8 (below gaze-crap-threshold=15), **When**
   classified, **Then** it falls in Q2 (complex but
   contract-tested).

---

### User Story 4 - Self-Analysis Command (Priority: P4)

`gaze self-check` runs the full analysis pipeline on Gaze's own
codebase: CRAP, GazeCRAP, contract coverage, and
over-specification. This is both a dogfooding mechanism and a
project health dashboard. Analysis covers exported functions only,
consistent with Gaze's default behavior.

**Why this priority**: Dogfooding is important but depends on all
other user stories functioning. It is a specific application of the
general capability.

**Independent Test**: Can be tested by running `gaze self-check`
and verifying it completes without error, produces valid output,
and covers all Gaze source packages.

**Acceptance Scenarios**:

1. **Given** the Gaze project repository, **When**
   `gaze self-check` runs, **Then** it runs
   `go test -coverprofile`, computes CRAP and GazeCRAP for all
   exported functions, and produces a full report.
2. **Given** `gaze self-check --ci`, **When** CRAPload exceeds
   threshold, **Then** exit code is non-zero.
3. **Given** `gaze self-check`, **When** results are produced,
   **Then** the report includes: total CRAPload, total
   GazeCRAPload, average contract coverage, worst offender list
   (top 5 by GazeCRAP).
4. **Given** `gaze self-check`, **When** analysis runs, **Then**
   only exported functions are analyzed (unexported functions are
   excluded).

---

### User Story 5 - CI Integration (Priority: P5)

Gaze provides CI-friendly flags for both CRAP and GazeCRAP
thresholds, with exit codes and summary output suitable for CI
pipelines.

**Why this priority**: Enforcement layer. Depends on all metrics
being computed correctly.

**Independent Test**: Can be tested by running against known code
with various thresholds and verifying exit codes.

**Acceptance Scenarios**:

1. **Given** `--max-crapload=5` and a codebase with CRAPload 7,
   **When** Gaze runs, **Then** exit code is non-zero and the
   report lists the 7 crappy functions.
2. **Given** `--max-gaze-crapload=3` and a codebase with
   GazeCRAPload 2, **When** Gaze runs, **Then** exit code is 0
   (pass).
3. **Given** a CI environment, **When** Gaze runs with threshold
   flags, **Then** output includes a one-line summary suitable
   for CI status (e.g., "CRAPload: 3/5 (PASS) |
   GazeCRAPload: 1/3 (PASS)").
4. **Given** no threshold flags, **When** Gaze runs, **Then**
   exit code is always 0 (report-only mode).

---

### Edge Cases

- What happens when a function has no tests at all (not even
  imported)? Coverage MUST default to 0%, and the CRAP formula
  MUST use 0% coverage (worst case).
- How does Gaze handle functions in `_test.go` files? Test
  functions MUST be excluded from CRAP analysis — they are test
  infrastructure, not production code.
- What happens when `go test` fails? Gaze MUST report the failure
  and abort with a clear error. The `--coverprofile` flag allows
  providing a pre-generated profile to skip this step.
- How does Gaze handle generated code? Generated code (detected
  via `// Code generated` comment per Go convention) MUST be
  excluded from CRAPload counts but MAY be included in detailed
  output with a `[generated]` marker.
- What happens when a function exists in code but not in the
  coverage profile? It has 0% coverage — it was never executed
  during tests.
- How are method receivers handled in complexity-coverage join?
  Join on file path + line number, not function name alone, to
  handle methods across different types with the same name.
- What happens when contract coverage is unavailable for a
  function (Specs 001-003 not yet complete)? GazeCRAP MUST be
  reported as null/unavailable, and the function MUST be excluded
  from GazeCRAPload counts. Classic CRAP is still reported.
- What happens when CRAP and GazeCRAP thresholds differ and a
  function exceeds one but not the other? Quadrant classification
  MUST use each metric's own threshold independently. A function
  can be in Q2 (high CRAP, low GazeCRAP) or Q3 (low CRAP, high
  GazeCRAP).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Gaze MUST compute per-function cyclomatic complexity
  using the `github.com/fzipp/gocyclo` library or equivalent
  AST-based analysis.
- **FR-002**: Gaze MUST parse `go test -coverprofile` output to
  extract per-function line coverage percentages using the
  `golang.org/x/tools/cover` package.
- **FR-003**: Gaze MUST compute classic CRAP score per function
  using the formula:
  `CRAP(m) = comp(m)^2 * (1 - cov(m)/100)^3 + comp(m)`.
- **FR-004**: Gaze MUST compute GazeCRAP score per function using
  Gaze's contract coverage in place of line coverage:
  `GazeCRAP(m) = comp(m)^2 * (1 - contractCov(m))^3 + comp(m)`
  where `contractCov` is a ratio from 0.0 to 1.0.
- **FR-005**: Gaze MUST report CRAPload (count of functions with
  CRAP >= crap-threshold) and GazeCRAPload (count of functions
  with GazeCRAP >= gaze-crap-threshold).
- **FR-006**: The default CRAP threshold MUST be 15, configurable
  via `--crap-threshold`. The default GazeCRAP threshold MUST be
  15, independently configurable via `--gaze-crap-threshold`.
- **FR-007**: Gaze MUST classify each function into a quadrant:
  - Q1 (Safe): CRAP < crap-threshold AND GazeCRAP < gaze-crap-threshold
  - Q2 (Complex but contract-tested): CRAP >= crap-threshold AND GazeCRAP < gaze-crap-threshold
  - Q3 (Simple but under-specified): CRAP < crap-threshold AND GazeCRAP >= gaze-crap-threshold
  - Q4 (Dangerous): CRAP >= crap-threshold AND GazeCRAP >= gaze-crap-threshold
- **FR-008**: Gaze MUST output results in both JSON
  (machine-readable) and text (human-readable) formats.
- **FR-009**: JSON output MUST include per-function: name, package,
  file, line, complexity, line_coverage, contract_coverage, crap,
  gaze_crap, quadrant.
- **FR-010**: Gaze MUST support `gaze self-check` as a convenience
  command that runs the full pipeline on its own source code,
  analyzing exported functions only.
- **FR-011**: Gaze MUST support `--max-crapload=N` and
  `--max-gaze-crapload=N` flags for CI threshold enforcement
  (non-zero exit on violation).
- **FR-012**: Gaze MUST automatically run
  `go test -coverprofile` if no coverage profile is provided, or
  accept one via `--coverprofile` flag.
- **FR-013**: Gaze MUST exclude `_test.go` files and functions
  with `// Code generated` headers from CRAPload counts.
- **FR-014**: Gaze MUST join complexity data with coverage data by
  file path and function declaration position, handling the
  import-path-to-filesystem-path mapping correctly.
- **FR-015**: When contract coverage is unavailable for a function,
  Gaze MUST fall back to classic CRAP only, report GazeCRAP as
  unavailable, and exclude the function from GazeCRAPload counts.
  A warning MUST be emitted.
- **FR-016**: Gaze MUST support Go 1.24+ (consistent with
  Spec 001 as updated).

### Key Entities

- **CRAPScore**: Per-function composite score. Attributes:
  function (FunctionTarget from Spec 001), complexity (int),
  line_coverage (float64, 0-100), contract_coverage (float64,
  0-100, nullable if unavailable), crap (float64), gaze_crap
  (float64, nullable), quadrant (Q1-Q4, nullable if gaze_crap
  unavailable).
- **CRAPReport**: Aggregate report. Attributes: scores
  ([]CRAPScore), crapload (int), gaze_crapload (int),
  crap_threshold (float64), gaze_crap_threshold (float64),
  summary (CRAPSummary).
- **CRAPSummary**: Aggregate statistics. Attributes:
  total_functions (int), avg_complexity (float64),
  avg_line_coverage (float64), avg_contract_coverage (float64,
  nullable), avg_crap (float64), avg_gaze_crap (float64,
  nullable), worst_crap ([]CRAPScore, top 5),
  worst_gaze_crap ([]CRAPScore, top 5),
  quadrant_counts (map[Quadrant]int).
- **Quadrant**: Classification enum: Q1_Safe,
  Q2_ComplexButTested, Q3_SimpleButUnderspecified, Q4_Dangerous.
- **CorrelationData**: Raw export for external analysis.
  Attributes: per-function records with all metric fields plus
  over_specification_count (from Spec 003, nullable).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: CRAP scores match hand-computed values for a
  benchmark suite of 20+ functions with known complexity and
  coverage (tolerance: +/- 0.01).
- **SC-002**: GazeCRAP scores correctly use contract coverage from
  Spec 003 in the formula, verified against hand-computed values.
- **SC-003**: CRAPload count matches the number of functions with
  CRAP >= crap-threshold for the benchmark suite.
- **SC-004**: Quadrant classification is correct for functions
  spanning all four quadrants, using independent thresholds.
- **SC-005**: `gaze self-check` completes successfully on the Gaze
  codebase and produces a valid report with all exported
  functions covered.
- **SC-006**: CI threshold enforcement correctly exits non-zero
  when CRAPload or GazeCRAPload exceeds the respective limit.
- **SC-007**: Per-function complexity matches `gocyclo` output for
  all analyzed functions.
- **SC-008**: Coverage parsing matches `go tool cover -func`
  output for all analyzed functions.

## Dependencies

Classic CRAP (US1) has no dependency on other Gaze specs — it
only needs `gocyclo` + `go test -coverprofile`. This means US1
can be implemented immediately, in parallel with Specs 002 and 003.

GazeCRAP (US2) depends on:
- Spec 001: Side effect detection (to know what effects exist)
- Spec 002: Contractual classification (to know which are
  contractual)
- Spec 003: Test quality metrics (to compute contract coverage)

Quadrant analysis (US3) depends on both CRAP and GazeCRAP.

Self-check (US4) and CI integration (US5) depend on all of the
above.

```
                    ┌─────────────────────┐
                    │ gocyclo + go test   │
                    │ -coverprofile       │
                    └──────────┬──────────┘
                               │
                               ▼
                    ┌─────────────────────┐
                    │ US1: Classic CRAP   │
                    │ (no Gaze deps)      │
                    └──────────┬──────────┘
                               │
    ┌──────────────────────────┤
    │                          │
    ▼                          ▼
┌────────────┐    ┌─────────────────────┐
│ Spec 001   │    │                     │
│ Spec 002   │───▶│ US2: GazeCRAP       │
│ Spec 003   │    │                     │
└────────────┘    └──────────┬──────────┘
                             │
                             ▼
                  ┌─────────────────────┐
                  │ US3: Quadrant +     │
                  │ Correlation         │
                  └──────────┬──────────┘
                             │
                     ┌───────┴───────┐
                     ▼               ▼
              ┌────────────┐  ┌────────────┐
              │ US4: Self- │  │ US5: CI    │
              │ check      │  │ Integration│
              └────────────┘  └────────────┘
```
