# Feature Specification: Contract Coverage Gap Remediation

**Feature Branch**: `008-contract-coverage-gaps`
**Created**: 2026-02-27
**Status**: Draft
**Input**: User description: "Spec 008: Contract Coverage Gap Remediation"

## User Scenarios & Testing *(mandatory)*

### User Story 1 — Classification Signal Verification (Priority: P1)

A developer modifies one of the classification signal analyzers
(`AnalyzeVisibilitySignal`, `AnalyzeGodocSignal`, `AnalyzeCallerSignal`)
and wants confidence that the change produces correct signal weights
and reasoning strings. Currently, no direct unit tests exist for these
functions — all coverage flows indirectly through the `Classify()`
integration pipeline, which only checks aggregate classification
labels. A regression in an individual signal (e.g., returning weight 0
instead of 15 for a contractual godoc keyword) would silently reduce
classification confidence without failing any test.

After this feature, each signal analyzer has dedicated unit tests that
verify weight computation, reasoning text, edge cases (nil inputs,
boundary values), and the priority ordering of competing signals. A
regression in any dimension immediately fails a focused test.

**Why this priority**: The classification signals are pure functions
with clear contracts (input → Signal struct). They are the lowest-effort,
highest-value targets: no test infrastructure changes needed, and fixing
them moves 3 functions out of the "zero contract coverage" category.
`AnalyzeNamingSignal` already has direct tests — the other three are
the gap.

**Independent Test**: Run `go test -run TestAnalyzeVisibilitySignal
./internal/classify/...` (and equivalent for Godoc/Caller) to verify
each signal function produces correct weights and reasoning for all
input combinations.

**Acceptance Scenarios**:

1. **Given** a function declaration for an exported function with an
   exported return type and an exported receiver, **When**
   `AnalyzeVisibilitySignal` is called, **Then** the returned Signal
   has weight 20 (clamped max of 8+6+6) and reasoning mentions all
   three dimensions.
2. **Given** a function declaration whose godoc contains "returns",
   **When** `AnalyzeGodocSignal` is called with effectType
   `ReturnValue`, **Then** the returned Signal has weight +15 and
   reasoning mentions "returns."
3. **Given** a function object referenced by 4+ cross-package callers,
   **When** `AnalyzeCallerSignal` is called, **Then** the returned
   Signal has weight 15 (capped) and reasoning mentions the caller
   count.

---

### User Story 2 — Analysis Core Contract Assertions (Priority: P1)

A developer modifies `AnalyzeP1Effects`, `AnalyzeP2Effects`, or
`AnalyzeReturns` and wants to verify that the function's direct
output is correct — independent of the higher-level
`AnalyzeFunctionWithSSA` pipeline. Currently, all 46+ tests for these
functions call them indirectly through `AnalyzeFunctionWithSSA`, which
also runs mutations, SSA analysis, and other phases. If the pipeline
changes its call pattern, these functions could silently lose coverage.
Additionally, the quality pipeline cannot attribute existing test
assertions to these specific functions because the test-to-target
pairing resolves to `AnalyzeFunctionWithSSA`, not the individual
analysis functions — resulting in 0% contract coverage despite
strong behavioral testing.

After this feature, each analysis function has direct unit tests that
call the function with crafted inputs (loaded from test fixtures via
`go/packages`) and assert on the returned `[]SideEffect` slice
directly. The quality pipeline can then attribute assertions to these
functions, moving them from "Dangerous" (Q4) to "Safe" (Q1).

**Why this priority**: `AnalyzeP1Effects` has the highest GazeCRAP
score in the entire codebase (1,056). Together, these three functions
represent the largest GazeCRAP reduction opportunity.

**Independent Test**: Run `go test -run TestAnalyzeP1Effects_Direct
./internal/analysis/...` to verify direct invocation produces correct
side effects for each detection category.

**Acceptance Scenarios**:

1. **Given** a function declaration containing a global variable
   assignment, **When** `AnalyzeP1Effects` is called directly,
   **Then** the returned slice contains a `GlobalMutation` effect
   with tier P1.
2. **Given** a function declaration containing `os.WriteFile()`,
   **When** `AnalyzeP2Effects` is called directly, **Then** the
   returned slice contains a `FileSystemWrite` effect with tier P2.
3. **Given** a function declaration with two return values (int,
   error), **When** `AnalyzeReturns` is called directly, **Then**
   the returned slice contains one `ReturnValue` effect and one
   `ErrorReturn` effect, both with tier P0.

---

### User Story 3 — CLI Layer Test Hardening (Priority: P2)

A developer modifies the TUI rendering in `renderAnalyzeContent` or
the quality pipeline orchestration in `buildContractCoverageFunc` and
wants to verify correctness. Currently, `renderAnalyzeContent` has
**zero test coverage of any kind** (no test file exists for
`interactive.go`), and `buildContractCoverageFunc` has two tests that
only verify "does not panic" without checking actual coverage values.

After this feature, `renderAnalyzeContent` has unit tests verifying
output content (function names, effect types, table structure), and
`buildContractCoverageFunc` has strengthened tests that assert on
actual coverage percentages returned by the closure.

**Why this priority**: P2 because these are CLI-layer functions with
more complex test setup requirements (TUI rendering, full pipeline
orchestration). The value is real but the effort-to-impact ratio is
lower than Groups A and B.

**Independent Test**: Run `go test -run TestRenderAnalyzeContent
./cmd/gaze/...` to verify the renderer produces correct output for
known inputs.

**Acceptance Scenarios**:

1. **Given** an empty `[]AnalysisResult` slice, **When**
   `renderAnalyzeContent` is called, **Then** the output contains
   a title line indicating zero functions.
2. **Given** an `AnalysisResult` with side effects, **When**
   `renderAnalyzeContent` is called, **Then** the output contains
   the function's qualified name, each effect's tier, and truncated
   descriptions.
3. **Given** a valid package pattern resolving to a testable package,
   **When** `buildContractCoverageFunc` is called and the returned
   closure is invoked with a known function, **Then** the closure
   returns a coverage percentage > 0 and `ok == true`.

---

### User Story 4 — Baseline Measurement and Verification (Priority: P1)

Before adding any new tests, the developer needs a measured baseline
of the project's current weighted average contract coverage (spec 007
deferred tasks T037 and T044). After all contract coverage gaps are
remediated, the developer re-measures to verify improvement. This
provides a quantified before/after comparison demonstrating the value
of the remediation effort.

**Why this priority**: P1 because baseline measurement must happen
before any changes to establish the comparison point. This also
completes the deferred validation tasks from spec 007.

**Independent Test**: Run `gaze quality --format=json ./...` and
compute the weighted average contract coverage from the JSON output.
Run the quickstart.md validation steps from spec 007.

**Acceptance Scenarios**:

1. **Given** the codebase before any spec 008 changes, **When**
   `gaze quality --format=json` is run, **Then** a baseline weighted
   average contract coverage number is recorded.
2. **Given** the codebase after all spec 008 changes, **When**
   `gaze quality --format=json` is run, **Then** the weighted average
   contract coverage is higher than the baseline.

---

### User Story 5 — GazeCRAP Score Improvement (Priority: P2)

After adding contract-level assertions for the target functions, the
developer runs `gaze crap` and verifies that the GazeCRAPload
(count of functions exceeding the GazeCRAP threshold) has decreased,
and that the worst-offender functions have moved from the "Dangerous"
(Q4) or "Underspecified" (Q3) quadrant to a better quadrant.

**Why this priority**: P2 because this is a verification story that
depends on all other stories being complete. It validates the
aggregate outcome.

**Independent Test**: Run `gaze crap --format=json ./...` and compare
the quadrant distribution and GazeCRAPload against the pre-change
baseline.

**Acceptance Scenarios**:

1. **Given** the completed remediation, **When** `gaze crap` is run,
   **Then** the GazeCRAPload is lower than the pre-change baseline.
2. **Given** the completed remediation, **When** `gaze crap` is run,
   **Then** `AnalyzeP1Effects`, `AnalyzeP2Effects`, and
   `AnalyzeReturns` are no longer in the Dangerous (Q4) quadrant.

---

### Edge Cases

- What happens when a signal analyzer receives a nil `funcDecl` or
  nil `funcObj`? Tests MUST verify zero-signal returns.
- What happens when `AnalyzeGodocSignal` receives a function whose
  godoc contains both incidental and contractual keywords? The
  incidental keyword MUST take priority (weight -15).
- What happens when `AnalyzeCallerSignal` receives a function whose
  only callers are in the same package? Same-package callers MUST
  be excluded; weight MUST be 0.
- What happens when `AnalyzeP1Effects` or `AnalyzeP2Effects` is
  called with a function that has no body (`fd.Body == nil`)? MUST
  return an empty slice without panic.
- What happens when `renderAnalyzeContent` receives results with
  descriptions longer than 50 characters? Descriptions MUST be
  truncated.
- What happens when `buildContractCoverageFunc` receives a pattern
  that resolves to zero packages? MUST return nil (not a closure
  that panics).

## Requirements *(mandatory)*

### Functional Requirements

#### Group A: Classification Signal Tests

- **FR-001**: `AnalyzeVisibilitySignal` MUST have direct unit tests
  verifying weight computation for each independent dimension:
  exported function (+8), exported return type (+6), exported
  receiver type (+6), and the weight clamp to maxVisibilityWeight
  (20).
- **FR-002**: `AnalyzeVisibilitySignal` tests MUST verify the
  `Reasoning` string contains each matching dimension's description.
- **FR-003**: `AnalyzeVisibilitySignal` tests MUST verify zero-signal
  returns for unexported functions, nil `funcDecl`, and nil `funcObj`.
- **FR-004**: `AnalyzeGodocSignal` MUST have direct unit tests
  verifying weight +15 for each contractual keyword when paired with
  a matching `effectType`, and weight 0 for non-matching effectTypes.
- **FR-005**: `AnalyzeGodocSignal` tests MUST verify incidental
  keyword priority: when godoc contains both "logs" and "returns",
  weight MUST be -15 (incidental wins).
- **FR-006**: `AnalyzeGodocSignal` tests MUST verify zero-signal
  returns for nil `funcDecl` and nil `Doc`.
- **FR-007**: `AnalyzeCallerSignal` MUST have direct unit tests
  verifying the weight tiers: 1 caller → 5, 2-3 callers → 10,
  4+ callers → 15.
- **FR-008**: `AnalyzeCallerSignal` tests MUST verify that
  same-package callers are excluded from the count.
- **FR-009**: `AnalyzeCallerSignal` tests MUST verify zero-signal
  returns for nil `funcObj` and for functions with zero cross-package
  callers.

#### Group B: Analysis Core Tests

- **FR-010**: `AnalyzeP1Effects` MUST have direct unit tests that
  call the function with a `*ast.FuncDecl` loaded from a test fixture
  (not constructed manually) and assert on the returned
  `[]taxonomy.SideEffect` slice.
- **FR-011**: `AnalyzeP1Effects` direct tests MUST cover at minimum:
  `GlobalMutation`, `ChannelSend`, `ChannelClose`, `WriterOutput`,
  `HTTPResponseWrite`, `MapMutation`, `SliceMutation`, and a pure
  function (empty slice).
- **FR-012**: `AnalyzeP2Effects` MUST have direct unit tests using
  the same fixture-loading approach, covering at minimum:
  `GoroutineSpawn`, `Panic`, `FileSystemWrite`, `FileSystemDelete`,
  `LogWrite`, `ContextCancellation`, `CallbackInvocation`,
  `DatabaseWrite`, and a pure function.
- **FR-013**: `AnalyzeReturns` MUST have direct unit tests covering
  at minimum: single return, multiple returns, error return, named
  returns, deferred return mutation, and no returns.
- **FR-014**: All Group B direct tests MUST assert on the `Type`,
  `Tier`, and non-empty `Description` of each returned side effect.
- **FR-015**: Group B tests MUST NOT duplicate the behavioral
  coverage of existing indirect tests. They MUST focus on direct
  invocation and contract verification, not re-testing detection
  logic through `AnalyzeFunctionWithSSA`.

#### Group C: CLI Layer Tests

- **FR-016**: `renderAnalyzeContent` MUST have unit tests that verify
  the output string contains expected function names, side effect
  types, and tier indicators for known inputs.
- **FR-017**: `renderAnalyzeContent` tests MUST verify description
  truncation at 50 characters and correct handling of empty results.
- **FR-018**: `buildContractCoverageFunc` existing tests MUST be
  strengthened to assert on actual coverage percentage values (not
  just "does not panic").
- **FR-019**: `buildContractCoverageFunc` tests MUST verify that the
  returned closure produces `ok == true` and a reasonable percentage
  (> 0) for a known well-tested function.

#### Baseline and Verification

- **FR-020**: Before any code changes, the weighted average contract
  coverage MUST be measured and recorded as a baseline (completing
  spec 007 deferred task T037).
- **FR-021**: The spec 007 quickstart.md validation steps MUST be
  executed and results recorded (completing deferred task T044).
- **FR-022**: After all changes, the weighted average contract
  coverage MUST be re-measured and compared against the baseline.
- **FR-023**: After all changes, the GazeCRAPload MUST be re-measured
  and compared against the baseline.

### Key Entities

- **Signal**: A classification input produced by a signal analyzer,
  containing `Source` (string), `Weight` (int), and `Reasoning`
  (string). Signals are aggregated by `ComputeScore` to produce
  classification confidence.
- **SideEffect**: A detected observable effect of a function,
  containing `ID`, `Type`, `Tier`, `Description`, `Location`, and
  `Target`. Side effects are the contractual outputs of the analysis
  functions.
- **GazeCRAP Score**: A composite metric that penalizes functions
  with high complexity and low contract coverage. Functions with
  high GazeCRAP scores despite high line coverage represent the
  "tests that execute but don't verify" anti-pattern.
- **Quadrant**: The GazeCRAP classification of a function: Q1 (Safe),
  Q2 (Complex but Tested), Q3 (Underspecified), Q4 (Dangerous).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: All 10 target functions have direct unit tests with
  contract-level assertions that verify return values, not just
  "no panic" behavior. The functions are: `AnalyzeVisibilitySignal`,
  `AnalyzeGodocSignal`, `AnalyzeCallerSignal`, `AnalyzeP1Effects`,
  `AnalyzeP2Effects`, `AnalyzeReturns`, `renderAnalyzeContent`,
  `buildContractCoverageFunc` (strengthened), plus the 007 deferred
  validations (T037, T044).
- **SC-002**: The GazeCRAPload (count of functions above the GazeCRAP
  threshold) decreases from the pre-change baseline.
- **SC-003**: `AnalyzeP1Effects` and `AnalyzeP2Effects` move out of
  the Dangerous (Q4) quadrant.
- **SC-004**: The weighted average contract coverage (across all
  packages with quality data) increases from the pre-change baseline.
- **SC-005**: Zero regressions — the full test suite (`go test -race
  -count=1 -short ./...`) passes with no new failures.
- **SC-006**: No lint violations introduced (`golangci-lint run`
  passes).

## Assumptions

- Existing test fixtures in `testdata/src/` (e.g., `p1effects`,
  `p2effects`, `returns`) can be loaded via `go/packages` and their
  `*ast.FuncDecl` nodes extracted for direct unit test invocation.
  This avoids the complexity of constructing AST nodes manually.
- The `AnalyzeCallerSignal` tests will need a multi-package fixture
  where one package's functions are called from another package,
  to verify cross-package caller counting.
- `renderAnalyzeContent` can be tested by constructing
  `[]taxonomy.AnalysisResult` structs directly (no fixture loading
  needed — it's a pure rendering function).
- `buildContractCoverageFunc` tests that invoke the full pipeline
  will be guarded by `testing.Short()` since they require loading
  and analyzing real packages.
