# Feature Specification: Side Effect Detection Engine

**Feature Branch**: `001-side-effect-detection`
**Created**: 2026-02-20
**Status**: Draft
**Input**: User description: "Determine all observable side effects
produced by a test target in Go"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Analyze a Single Function (Priority: P1)

A developer points Gaze at a specific Go function (by package path
and function/method name) and receives a structured list of every
observable side effect that function produces. This is the core
analysis loop — one function in, one side effect report out.

**Why this priority**: This is the atomic unit of Gaze's value. Every
higher-level feature (package scanning, classification, metrics)
depends on accurate single-function analysis.

**Independent Test**: Can be fully tested by analyzing a set of
known Go functions with predetermined side effects and verifying
Gaze detects all of them with zero false positives.

**Acceptance Scenarios**:

1. **Given** a Go function that returns `(int, error)`, **When**
   Gaze analyzes it, **Then** the result contains two side effects:
   one for each return position, typed as `ReturnValue`.
2. **Given** a method with a pointer receiver that assigns to
   `self.field`, **When** Gaze analyzes it, **Then** the result
   contains a `ReceiverMutation` side effect identifying the
   mutated field.
3. **Given** a function that calls `os.WriteFile`, **When** Gaze
   analyzes it, **Then** the result contains a `FileSystemWrite`
   side effect.
4. **Given** a function with zero side effects (pure function),
   **When** Gaze analyzes it, **Then** the result contains an
   empty side effect list with no false positives.
5. **Given** a function that modifies a package-level variable,
   **When** Gaze analyzes it, **Then** the result contains a
   `GlobalMutation` side effect identifying the variable.
6. **Given** a function with a named return modified in a defer,
   **When** Gaze analyzes it, **Then** the result contains a
   `DeferredReturnMutation` side effect.

---

### User Story 2 - Analyze an Entire Package (Priority: P2)

A developer points Gaze at a Go package and receives side effect
analysis for all exported functions and methods in that package.
This enables batch analysis without specifying each function
individually.

**Why this priority**: Practical usability — developers work at the
package level. But this is a loop over US1, not new analysis logic.

**Independent Test**: Can be tested by analyzing a package with
multiple exported functions and verifying each function's side
effects are correctly reported.

**Acceptance Scenarios**:

1. **Given** a Go package with 5 exported functions, **When** Gaze
   analyzes the package, **Then** the result contains exactly 5
   function entries, each with their respective side effects.
2. **Given** a package with unexported functions, **When** Gaze
   analyzes the package, **Then** unexported functions are excluded
   from results by default.
3. **Given** a flag `--include-unexported`, **When** Gaze analyzes
   the package with that flag, **Then** unexported functions are
   included.

---

### User Story 3 - Structured Output Formats (Priority: P3)

Gaze outputs results in both human-readable (terminal table/text)
and machine-readable (JSON) formats, selectable via a flag.

**Why this priority**: Required by Constitution Principle III
(Actionable Output) but the core value is in the analysis, not the
formatting.

**Independent Test**: Can be tested by running the same analysis
with `--format=json` and `--format=text` and validating both
outputs are correct and parseable.

**Acceptance Scenarios**:

1. **Given** an analysis result, **When** output format is `json`,
   **Then** output is valid JSON conforming to a documented schema.
2. **Given** an analysis result, **When** output format is `text`
   (default), **Then** output is a human-readable table listing each
   side effect with type, location, and description.
3. **Given** an analysis result piped to another tool, **When**
   format is `json`, **Then** the JSON is parseable by `jq` and
   contains all fields documented in the schema.

---

### Edge Cases

- What happens when the target function does not exist in the
  specified package? Gaze MUST return a clear error, not an empty
  result.
- How does Gaze handle build errors in the target package? Gaze
  MUST report the build error and exit with a non-zero code.
- How does Gaze handle functions with extremely deep call chains
  (transitive side effects)? For v1, Gaze analyzes only the
  direct function body — transitive analysis is out of scope
  and MUST be documented as a known limitation.
- How does Gaze handle generated code (e.g., protobuf stubs)?
  Gaze MUST analyze them the same as hand-written code; the
  taxonomy applies regardless of origin.
- How does Gaze handle CGo functions? Gaze MUST report a single
  `CgoCall` side effect for each C function invoked, with a note
  that the C side is opaque.
- What happens when a function uses `unsafe.Pointer`? Gaze MUST
  report an `UnsafeMutation` side effect and flag that the
  analysis may be incomplete.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Gaze MUST accept a Go package path and optional
  function name as input and produce a side effect analysis.
- **FR-002**: Gaze MUST detect all P0-tier side effects (return
  values, pointer receiver mutation, pointer argument mutation,
  error returns including sentinel/wrapped errors) with zero
  known false negatives.
- **FR-003**: Gaze MUST detect all P1-tier side effects (slice/map
  mutation, package-level state mutation, io.Writer output, HTTP
  response writing, channel send/close, named returns modified
  in defer) with a documented false-negative rate.
- **FR-004**: Gaze MUST detect P2-tier side effects (file system
  I/O, database operations, goroutine spawning, panic, callback
  invocation, logging, context cancellation).
- **FR-005**: Gaze SHOULD detect P3-tier side effects (stdout/
  stderr, env vars, sync primitives, atomic ops, time.Now,
  os.Exit, recover).
- **FR-006**: Gaze MAY detect P4-tier side effects (reflection,
  unsafe, CGo, finalizers, sync.Pool, allocation patterns,
  closure capture mutation). Where detection is infeasible,
  Gaze MUST document the limitation.
- **FR-007**: Gaze MUST NOT produce false positives for P0-tier
  effects. A reported side effect MUST correspond to a real
  observable change.
- **FR-008**: Gaze MUST output results in JSON format conforming
  to a documented schema.
- **FR-009**: Gaze MUST output results in a human-readable text
  format suitable for terminal display.
- **FR-010**: Gaze MUST report the priority tier (P0-P4) for each
  detected side effect.
- **FR-011**: Gaze MUST report the source location (file:line) of
  each detected side effect.
- **FR-012**: Gaze MUST use the `go/ast`, `go/types`, and
  `golang.org/x/tools/go/ssa` packages (or equivalent) for
  analysis. It MUST NOT require modifying the target source code.
- **FR-013**: Gaze MUST support Go 1.24+ (consistent with go.mod
  and Spec 004 FR-016).
- **FR-014**: Analysis scope for v1 is direct function body only.
  Transitive side effects (effects in called functions) are
  explicitly out of scope and MUST be documented.

### Key Entities

- **SideEffect**: A single observable change detected in a
  function. Attributes: type (enum from taxonomy), priority tier
  (P0-P4), source location (file:line:col), description (human-
  readable), affected target (variable name, field name, channel
  name, etc.).
- **FunctionTarget**: The function under analysis. Attributes:
  package path, function/method name, receiver type (if method),
  signature, source location.
- **AnalysisResult**: The output of analyzing one function.
  Attributes: target (FunctionTarget), side effects ([]SideEffect),
  analysis metadata (duration, Go version, Gaze version, any
  warnings or limitations encountered).
- **SideEffectType**: Enum of all side effect types from the
  taxonomy. Grouped by priority tier:
  - P0: `ReturnValue`, `ErrorReturn`, `SentinelError`,
    `ReceiverMutation`, `PointerArgMutation`
  - P1: `SliceMutation`, `MapMutation`, `GlobalMutation`,
    `WriterOutput`, `HTTPResponseWrite`, `ChannelSend`,
    `ChannelClose`, `DeferredReturnMutation`
  - P2: `FileSystemWrite`, `FileSystemDelete`, `FileSystemMeta`,
    `DatabaseWrite`, `DatabaseTransaction`, `GoroutineSpawn`,
    `Panic`, `CallbackInvocation`, `LogWrite`,
    `ContextCancellation`
  - P3: `StdoutWrite`, `StderrWrite`, `EnvVarMutation`,
    `MutexOp`, `WaitGroupOp`, `AtomicOp`, `TimeDependency`,
    `ProcessExit`, `RecoverBehavior`
  - P4: `ReflectionMutation`, `UnsafeMutation`, `CgoCall`,
    `FinalizerRegistration`, `SyncPoolOp`,
    `ClosureCaptureMutation`

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Gaze detects 100% of P0-tier side effects in a
  benchmark suite of 50+ known Go functions with zero false
  negatives and zero false positives.
- **SC-002**: Gaze detects >= 95% of P1-tier side effects in the
  benchmark suite.
- **SC-003**: Gaze detects >= 85% of P2-tier side effects in the
  benchmark suite.
- **SC-004**: Gaze completes single-function analysis in < 500ms
  for functions up to 200 lines of code.
- **SC-005**: Gaze completes package-level analysis in < 5s for
  packages with up to 50 exported functions.
- **SC-006**: JSON output is valid against a published JSON Schema
  and parseable by standard tools (jq, Go json.Unmarshal).
- **SC-007**: Human-readable output fits in an 80-column terminal
  without horizontal scrolling for typical results.
