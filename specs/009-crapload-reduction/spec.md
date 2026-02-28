# Feature Specification: CRAPload Reduction

**Feature Branch**: `009-crapload-reduction`
**Created**: 2026-02-27
**Status**: Complete
**Input**: User description: "please make a spec to address the top 5 priority improvements"

## User Scenarios & Testing *(mandatory)*

### User Story 1 — Contract Coverage for Document Filter (Priority: P1)

A developer runs the quality analysis on the project and sees the document filter function flagged as the only Q3 function in the entire codebase — the worst GazeCRAP score (72) despite having near-perfect line coverage. The function has 0% contract coverage, meaning tests execute the code but no test assertions verify the function's observable output (a boolean filtering decision). The developer wants to close this gap so the function moves to Q1 and no longer dominates the "worst GazeCRAP" list.

**Why this priority**: This is the single highest GazeCRAP score in the project and the only Q3 function. Closing the contract coverage gap eliminates the entire Q3 quadrant from the health report and delivers the largest single-function GazeCRAP reduction possible.

**Independent Test**: Can be fully tested by running the quality analysis on the document scanner package before and after, verifying that the filter function moves from Q3 to Q1 with GazeCRAP below the threshold.

**Acceptance Scenarios**:

1. **Given** the filter function currently has 0% contract coverage, **When** direct tests are added that assert on the boolean return value for each distinct code path (include match, exclude match, default inclusion, nil config), **Then** the function achieves at least 80% contract coverage.
2. **Given** the new contract-level tests, **When** running the quality analysis, **Then** the filter function's GazeCRAP score drops below 15 and the function is classified as Q1 (Safe).
3. **Given** the new tests, **When** running the full test suite, **Then** all existing tests continue to pass with zero regressions.

---

### User Story 2 — Unit Tests for Module Loader (Priority: P1)

A developer reviews the quality report and sees the module loader function with a CRAP score of 56 and 0% line coverage. This function loads all packages in a module for static analysis — a critical path in the analysis pipeline — yet has no direct tests whatsoever. A regression in this function would cascade across all downstream features. The developer wants to establish a regression safety net and reduce the CRAP score.

**Why this priority**: Zero coverage on a critical infrastructure function is the highest-risk scenario. Adding tests delivers the largest CRAP reduction per line of test code and protects a foundational pipeline component.

**Independent Test**: Can be tested by running the quality analysis on the loader package before and after and verifying the CRAP score drops below 15.

**Acceptance Scenarios**:

1. **Given** a valid module directory, **When** the module loader is called, **Then** it returns a result containing at least one package with resolved type information.
2. **Given** a non-existent or empty directory, **When** the module loader is called, **Then** it returns a descriptive error without panicking.
3. **Given** a module containing both valid packages and packages with compilation errors, **When** the module loader is called, **Then** it returns only the valid packages and excludes the broken ones.
4. **Given** the new tests, **When** running the quality analysis, **Then** the module loader's CRAP score drops below 15.

---

### User Story 3 — Testable CLI Commands (Priority: P2)

A developer examines the quality report and sees two CLI command functions — the CRAP report runner (CRAP 52) and the self-check runner (CRAP 43) — flagged as high-risk. Both functions already follow the testable CLI pattern (accepting params structs with writer outputs) but their heavy pipeline dependencies are hard-wired, making fast unit testing impossible without running the full analysis pipeline. The developer wants to make these dependencies pluggable so that fast unit tests can cover all branches without spawning subprocesses or loading real packages.

**Why this priority**: These two functions account for a combined CRAP of 95 and represent the CLI's two primary user-facing commands. Making them testable establishes the pattern for all future CLI commands and removes 2 functions from the CRAPload.

**Independent Test**: Can be tested by running the quality analysis on the CLI package before and after and verifying both functions drop below CRAP 20.

**Acceptance Scenarios**:

1. **Given** the CRAP report runner with pluggable dependencies, **When** a developer provides stub implementations of the analysis and coverage functions, **Then** unit tests can exercise all branches (valid text output, valid JSON output, coverage unavailability warning, threshold pass, threshold breach) without running the real pipeline.
2. **Given** the self-check runner with pluggable dependencies, **When** a developer provides stubs, **Then** unit tests can exercise the happy path, format validation, and threshold checking without spawning subprocesses.
3. **Given** the pluggability changes, **When** running the existing CLI commands as an end user, **Then** behavior is identical to the current implementation — no user-visible changes.
4. **Given** the new fast unit tests, **When** running the standard (non-long-running) test suite, **Then** both functions achieve at least 70% line coverage.

---

### User Story 4 — Decompose Quality Pipeline Orchestrator (Priority: P2)

A developer examining the quality pipeline orchestrator function sees a 100-line function with the highest CRAP score in the entire project (70). It orchestrates four sequential steps — package resolution, side-effect analysis, classification, and quality assessment — all tightly coupled in a single function with 18 cyclomatic complexity. The developer wants to decompose it into smaller, independently testable functions that each handle one pipeline stage.

**Why this priority**: This is the highest CRAP score in the project. Decomposition reduces per-function complexity and enables targeted testing of each pipeline stage, which is impossible with the current monolithic structure.

**Independent Test**: Can be tested by running the quality analysis on the CLI package before and after and verifying that no function in the decomposed pipeline has a CRAP score above 30.

**Acceptance Scenarios**:

1. **Given** the decomposed pipeline functions, **When** a developer tests each function independently, **Then** package resolution, per-package analysis, and coverage map aggregation can each be verified with focused inputs.
2. **Given** the decomposed pipeline, **When** running the quality analysis, **Then** no individual function in the pipeline has a CRAP score above 30.
3. **Given** the decomposed pipeline, **When** running the CLI command end-to-end, **Then** the output is identical to the current implementation.
4. **Given** the decomposition, **When** running the full test suite, **Then** all existing tests pass with zero regressions.

---

### User Story 5 — Decompose Effect Detection Engines (Priority: P3)

A developer examining the P1 effect detection function sees cyclomatic complexity of 32 — the highest in the project — and GazeCRAP of 32 (Q4). Despite perfect contract and line coverage, the function is inherently risky to modify because a single monolithic callback handles seven distinct effect types (GlobalMutation, MapMutation, SliceMutation, ChannelSend, ChannelClose, WriterOutput, HTTPResponseWrite) across four AST node type arms. The P2 effect detection function has the same structural problem (CC=18, GazeCRAP 18, Q4). The developer wants to decompose both callbacks into focused handler functions, each responsible for one node type, making future additions of new effect categories safer and more modular.

**Why this priority**: This is the lowest urgency because coverage is already perfect — the risk is purely structural. Decomposition reduces per-function complexity and makes the architecture more maintainable, but does not address any testing gap.

**Independent Test**: Can be tested by running the quality analysis on the analysis package before and after and verifying that no function has a GazeCRAP score above 15.

**Acceptance Scenarios**:

1. **Given** the decomposed P1 effect detection with per-node-type handlers, **When** running the full side-effect analysis on any package, **Then** the detection results are identical to the current implementation — same effects, same types, same descriptions.
2. **Given** the decomposed handlers, **When** running the quality analysis, **Then** no individual handler function has a GazeCRAP score above 15.
3. **Given** the P2 effect detection function is decomposed using the same pattern, **Then** it also drops below GazeCRAP 15.
4. **Given** the decomposition, **When** a developer adds a new effect category in the future, **Then** they can add a new handler function without modifying the existing handlers.
5. **Given** all decomposition changes, **When** running the full test suite, **Then** all existing tests pass with zero regressions.

---

### Edge Cases

- What happens when the document filter receives a path with backslash separators? The function normalizes paths internally; tests MUST verify this behavior.
- What happens when the module loader is pointed at a directory containing a module file but no source files? It MUST return a descriptive error (no packages found).
- What happens when the CRAP report runner receives an empty patterns list? It MUST either use a sensible default or return a descriptive error.
- What happens when the pipeline orchestrator resolves patterns that match only test-variant packages? These MUST be filtered out, preserving current behavior.
- What happens when the decomposed effect detection encounters a node type not handled by any handler? The dispatcher MUST continue the walk, matching current behavior.

## Requirements *(mandatory)*

### Functional Requirements

#### US1 — Document Filter Contract Coverage

- **FR-001**: Direct tests MUST assert on the boolean return value of the document filter function for each distinct code path: include-pattern match, include-pattern miss, exclude-pattern match, default inclusion, and nil config fallback.
- **FR-002**: Tests MUST cover the glob-matching delegation path, including patterns with and without path separators (base-name matching vs. full-path matching).
- **FR-003**: After test additions, the document filter function MUST achieve at least 80% contract coverage as reported by the quality analysis.

#### US2 — Module Loader Unit Tests

- **FR-004**: Direct tests MUST exercise the module loader with a valid module directory and verify that the returned result contains at least one package with resolved types.
- **FR-005**: Tests MUST verify the error path when pointed at a non-existent or empty directory.
- **FR-006**: Tests MUST verify that packages with compilation errors are excluded from the returned result while valid packages are retained.
- **FR-007**: Slow tests that load real modules MUST be guarded to keep the standard test suite fast (under the default timeout).

#### US3 — CLI Command Testability

- **FR-008**: The CRAP report runner function MUST accept pluggable function dependencies for the analysis pipeline so that unit tests can provide stub implementations.
- **FR-009**: The self-check runner function MUST accept pluggable dependencies or delegate to the CRAP report runner internally to eliminate redundant pipeline wiring.
- **FR-010**: Fast unit tests MUST cover all branches of the CRAP report runner: valid text output, valid JSON output, coverage unavailability warning, threshold pass, and threshold breach error.
- **FR-011**: Fast unit tests MUST cover self-check runner branches: happy path, format validation error, module root not found error, and threshold checking.
- **FR-012**: Pluggability changes MUST NOT alter the external behavior of any CLI commands.

#### US4 — Pipeline Orchestrator Decomposition

- **FR-013**: The package resolution logic (pattern expansion and test-variant filtering) MUST be extracted into a separate, independently testable function.
- **FR-014**: The per-package analysis pipeline (analysis, classification, test loading, quality assessment) MUST be extracted into a separate function.
- **FR-015**: The coverage map aggregation logic MUST be extracted into a separate function or kept as a thin coordinator.
- **FR-016**: Each extracted function MUST have direct tests covering its primary paths.
- **FR-017**: The decomposition MUST NOT change the behavior or return value of the pipeline orchestrator's closure.

#### US5 — Effect Detection Decomposition

- **FR-018**: The inspection callback in the P1 effect detection function MUST be decomposed into per-node-type handler functions, each handling one node type (assignments, send statements, call expressions, increment/decrement statements).
- **FR-019**: Each handler function MUST be independently callable. Testability is satisfied when the handler is verified through the dispatcher's existing integration tests, which cover all node types and assert on identical output pre- and post-decomposition.
- **FR-020**: The P2 effect detection function MUST be decomposed using the same handler pattern.
- **FR-021**: Decomposition MUST NOT change the detected side effects for any existing test fixture — the analysis output MUST be identical pre- and post-decomposition.
- **FR-022**: After decomposition, no individual handler function SHOULD have cyclomatic complexity above 15.

#### Cross-Cutting

- **FR-023**: All changes MUST pass the standard test suite with zero regressions.
- **FR-024**: All changes MUST pass the linter with no new violations.
- **FR-025**: All new functions and types intended for external use MUST have documentation comments.

### Key Entities

- **CRAPload**: The count of functions with CRAP scores at or above the threshold (default 15). The primary metric this feature aims to reduce.
- **GazeCRAPload**: The count of functions with GazeCRAP scores (incorporating contract coverage) at or above the threshold (default 15). The secondary metric.
- **Quadrant**: Classification of a function based on CRAP and GazeCRAP thresholds — Q1 (Safe), Q2 (Complex but Tested), Q3 (Simple but Underspecified), Q4 (Dangerous).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: The project-wide GazeCRAPload drops from 7 to at most 4 (at least 3 functions removed from the "above threshold" list).
- **SC-002**: The project-wide CRAPload drops from 27 to at most 24 (at least 3 functions removed from the "above threshold" list).
- **SC-003**: The document filter function moves from Q3 to Q1 with GazeCRAP below 15.
- **SC-004**: The module loader function's CRAP score drops from 56 to below 15.
- **SC-005**: The CRAP report runner and self-check runner each drop below CRAP 20 in the standard test suite.
- **SC-006**: No individual function in the decomposed P1 or P2 effect detection has GazeCRAP above 15.
- **SC-007**: No individual function in the decomposed pipeline orchestrator has CRAP above 30.
- **SC-008**: All existing tests pass with zero regressions after all changes.
- **SC-009**: No new lint violations are introduced.

### Assumptions

- The document filter's existing tests (which test it indirectly through the higher-level scanner function) are not detected by the contract coverage pipeline as direct contract assertions. Dedicated direct tests that call the filter function explicitly and assert on its return value will be recognized.
- The module loader tests will use small fixture modules or the project's own module to keep execution time reasonable. The heaviest tests will be guarded to avoid exceeding the standard CI timeout.
- Pluggability in the CLI command functions will use function fields on the existing params structs rather than global state, consistent with the project's functional style preference.
- Decomposition of the P1 and P2 effect detection functions preserves the existing deduplication behavior (the seen-effects tracking) by passing state to each handler or maintaining it in the dispatcher.
- The pipeline orchestrator decomposition preserves the existing nil-return behavior when no coverage data can be collected.
- The self-check runner may be refactored to delegate to the CRAP report runner internally, since both functions perform essentially the same pipeline with different input defaults.
