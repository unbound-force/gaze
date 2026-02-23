# Feature Specification: Test Quality Metrics & Reporting

**Feature Branch**: `003-test-quality-metrics`
**Created**: 2026-02-20
**Status**: Complete
**Input**: User description: "Metrics to give weight against all
possible side effects, important to be able to refactor a test
target without changing the unit test"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Contract Coverage Score (Priority: P1)

A developer runs Gaze on a test function and its target, and
receives a Contract Coverage percentage: the ratio of contractual
side effects that the test asserts on to the total number of
contractual side effects. This is the primary quality metric.

**Why this priority**: This is Gaze's core value proposition — the
single number that answers "does this test adequately verify what
this function is supposed to do?" A test with 100% contract
coverage asserts on every contractual side effect.

**Independent Test**: Can be tested by writing a Go test that
asserts on some but not all contractual side effects of its target,
running Gaze, and verifying the reported percentage matches the
known ratio.

**Acceptance Scenarios**:

1. **Given** a function with 4 contractual side effects and a test
   that asserts on 3 of them, **When** Gaze scores the test,
   **Then** Contract Coverage is 75% and the 1 unasserted
   contractual effect is identified by name and location.
2. **Given** a function with 2 contractual side effects and a test
   that asserts on both, **When** Gaze scores the test, **Then**
   Contract Coverage is 100%.
3. **Given** a function with 3 contractual side effects and a test
   that asserts on none, **When** Gaze scores the test, **Then**
   Contract Coverage is 0% and all 3 unasserted effects are listed.
4. **Given** a pure function (return value only, classified as
   contractual) and a test that checks the return, **When** Gaze
   scores, **Then** Contract Coverage is 100%.

---

### User Story 2 - Over-Specification Score (Priority: P2)

A developer receives an Over-Specification Score indicating how
many incidental side effects the test asserts on. This measures
refactoring fragility — tests that assert on implementation details
will break when the implementation changes, even if the contract is
preserved.

**Why this priority**: Directly addresses the user's requirement to
"refactor a test target without changing the unit test." A test
with a high over-specification score is fragile. This metric is
second priority because it requires contract coverage (US1) to be
meaningful — you need to know what's contractual first.

**Independent Test**: Can be tested by writing a test that asserts
on both contractual and incidental side effects, running Gaze,
and verifying the over-specification count and score match.

**Acceptance Scenarios**:

1. **Given** a test that asserts on 3 contractual and 2 incidental
   side effects, **When** Gaze scores, **Then** Over-Specification
   Score reports 2 incidental assertions, lists them, and flags
   them as refactoring risks.
2. **Given** a test that asserts only on contractual side effects,
   **When** Gaze scores, **Then** Over-Specification Score is 0
   (clean).
3. **Given** a test that asserts on a log message (classified
   incidental), **When** Gaze scores, **Then** the log assertion
   is flagged with a suggestion: "Consider removing this
   assertion — logging is an implementation detail that may change
   during refactoring."

---

### User Story 3 - Progress Tracking Across Runs (Priority: P3)

Gaze results are comparable across runs. A developer can track
their test quality improvement over time by comparing reports from
different dates/commits. Output includes enough metadata to enable
diffing and trend analysis.

**Why this priority**: Required by Constitution Principle III
("Metrics MUST be comparable across runs"). Important for adoption
but not blocking for initial value delivery.

**Independent Test**: Can be tested by running Gaze twice on the
same code and verifying identical output, then modifying a test,
re-running, and verifying the diff is correct and meaningful.

**Acceptance Scenarios**:

1. **Given** two Gaze runs on identical code, **When** comparing
   JSON outputs, **Then** the side effect lists, classifications,
   and scores are identical (excluding timestamps and duration).
2. **Given** a test that is improved (one more contractual
   assertion added) between runs, **When** comparing reports,
   **Then** Contract Coverage shows the increase and the newly
   covered side effect is identified.
3. **Given** JSON output from two runs, **When** diffed, **Then**
   side effects are identified by stable IDs (not positional
   indexes) so diffs are meaningful.

---

### User Story 4 - CI Integration (Priority: P4)

Gaze supports CI-friendly exit codes and threshold-based pass/fail.
A team can add Gaze to their CI pipeline and fail builds when
contract coverage drops below a configurable threshold.

**Why this priority**: Important for enforcement but requires all
other user stories to function. Teams typically adopt a tool
manually before adding CI gates.

**Independent Test**: Can be tested by running Gaze with a
threshold flag against tests with known coverage and verifying
exit codes.

**Acceptance Scenarios**:

1. **Given** a `--min-contract-coverage=80` flag and a test with
   75% coverage, **When** Gaze runs, **Then** exit code is
   non-zero (failure).
2. **Given** a `--min-contract-coverage=80` flag and a test with
   90% coverage, **When** Gaze runs, **Then** exit code is 0
   (success).
3. **Given** a `--max-over-specification=2` flag and a test with
   3 incidental assertions, **When** Gaze runs, **Then** exit
   code is non-zero.
4. **Given** no threshold flags, **When** Gaze runs, **Then**
   exit code is always 0 (report-only mode, no enforcement).

---

### Edge Cases

- What happens when Gaze cannot determine which side effect a test
  assertion maps to? Gaze MUST report the assertion as "unmapped"
  and exclude it from both metrics. The unmapped assertion count
  MUST be reported separately.
- How does Gaze handle table-driven tests? Each sub-test row MUST
  be analyzed independently if the test uses `t.Run`. The
  aggregate coverage is the union of assertions across all
  sub-tests.
- How does Gaze handle test helper functions? Assertions made in
  helpers called by the test MUST be attributed to the test that
  called the helper.
- What happens when classification confidence is low (ambiguous
  side effects)? Ambiguous effects MUST be reported separately
  and NOT counted in Contract Coverage or Over-Specification.
  They MUST appear in a distinct "ambiguous" section of the
  report.
- How does Gaze handle tests that test multiple functions? Gaze
  MUST require a 1:1 test-to-target mapping. If a test calls
  multiple target functions, Gaze MUST report this and request
  the user specify which function is the target, or analyze
  each separately.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Gaze MUST compute Contract Coverage as:
  `(contractual side effects asserted on) /
  (total contractual side effects) * 100`, expressed as a
  percentage.
- **FR-002**: Gaze MUST compute Over-Specification Score as the
  count of incidental side effects that the test asserts on,
  along with a list identifying each one.
- **FR-003**: Gaze MUST identify which specific test assertions
  map to which specific side effects. Unmapped assertions MUST
  be reported separately.
- **FR-004**: Gaze MUST identify which contractual side effects
  are NOT asserted on and report them as "gaps" with the side
  effect's type, description, and source location.
- **FR-005**: Gaze MUST assign stable identifiers to side effects
  so that results are diffable across runs.
- **FR-006**: Gaze MUST support `--min-contract-coverage=N` flag
  that causes a non-zero exit code when coverage is below N%.
- **FR-007**: Gaze MUST support `--max-over-specification=N` flag
  that causes a non-zero exit code when incidental assertion
  count exceeds N.
- **FR-008**: Gaze MUST default to report-only mode (exit code 0)
  when no threshold flags are provided.
- **FR-009**: JSON output MUST extend the Spec 002 schema
  (additive) with metrics fields.
- **FR-010**: Human-readable output MUST include a summary line
  (e.g., "Contract Coverage: 75% (3/4) |
  Over-Specified: 2 incidental assertions").
- **FR-011**: Gaze MUST handle table-driven tests by analyzing
  each `t.Run` sub-test and computing the union of assertions.
- **FR-012**: Gaze MUST attribute assertions in test helper
  functions to the calling test.
- **FR-013**: Ambiguous side effects (confidence 50-79) MUST be
  excluded from both Contract Coverage and Over-Specification
  calculations and reported in a separate section.
- **FR-014**: Gaze MUST support Go 1.24+ (consistent with
  Specs 001 and 002).
- **FR-015**: For v1, assertion detection covers standard Go test
  patterns: `if got != want`, `t.Errorf`, `t.Fatalf`, and
  popular assertion libraries (`testify/assert`,
  `testify/require`, `google/go-cmp`). Additional libraries
  MAY be added.
- **FR-016**: Gaze MUST report a summary at the package level
  when multiple test functions are analyzed, including aggregate
  Contract Coverage and total Over-Specification count.

### Key Entities

- **AssertionMapping**: Links a test assertion to the side effect
  it verifies. Attributes: assertion_location (file:line),
  assertion_type (equality, error check, panic check, etc.),
  side_effect_id (stable ID from Spec 001), confidence (how
  certain is the mapping).
- **ContractCoverage**: The primary metric. Attributes: percentage
  (float), covered_count (int), total_contractual (int),
  gaps ([]SideEffect — contractual effects not asserted on).
- **OverSpecificationScore**: The fragility metric. Attributes:
  count (int), incidental_assertions ([]AssertionMapping —
  assertions on incidental effects), suggestions ([]string —
  actionable advice per incidental assertion).
- **QualityReport**: Complete output for one test-target pair.
  Attributes: test_function (name, location), target_function
  (FunctionTarget), contract_coverage (ContractCoverage),
  over_specification (OverSpecificationScore),
  ambiguous_effects ([]SideEffect), unmapped_assertions
  ([]assertion_location), metadata (run timestamp, Gaze version,
  commit hash if in git repo).
- **PackageSummary**: Aggregate metrics for a package. Attributes:
  total_tests (int), average_contract_coverage (float),
  total_over_specifications (int), worst_coverage_tests
  ([]QualityReport, bottom 5).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Contract Coverage correctly computes the ratio for a
  benchmark suite of 20+ test-target pairs with known coverage.
- **SC-002**: Over-Specification Score correctly identifies all
  assertions on incidental side effects in the benchmark suite.
- **SC-003**: Assertion-to-side-effect mapping achieves >= 90%
  accuracy for standard Go test patterns (direct comparison,
  testify, go-cmp).
- **SC-004**: Two runs on identical code produce identical metrics
  (determinism excluding timestamps).
- **SC-005**: CI threshold enforcement correctly exits non-zero
  when thresholds are violated and zero when they are met, across
  10+ test scenarios.
- **SC-006**: Package-level summary correctly aggregates individual
  test reports.
- **SC-007**: Table-driven test support correctly unions assertions
  across sub-tests for coverage calculation.
