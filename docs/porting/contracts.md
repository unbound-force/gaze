# Behavioral Contracts

This document defines the language-agnostic behavioral contracts that any Gaze implementation must honor. A **behavioral contract** is a testable rule about how Gaze detects, classifies, and scores side effects. If your port violates any contract marked MUST, it is not a conforming Gaze implementation.

These contracts are extracted from the reference Go implementation and stated without Go-specific details. A porter should be able to implement these rules in Python, Rust, TypeScript, or any language with access to its own AST and type system.

> **Notation**: Contracts use RFC 2119 keywords (MUST, SHOULD, MAY). Each contract has a unique ID for cross-referencing in test suites.

---

## 1. Effect Taxonomy Contract

The effect taxonomy defines what Gaze detects. Every side effect belongs to exactly one type, and every type belongs to exactly one priority tier.

### EC-001: Tier Membership

A port MUST assign each side effect type to exactly one of five priority tiers (P0–P4). The tier assignments are fixed and MUST NOT be configurable.

| Tier | Effect Types | Count |
|------|-------------|-------|
| P0 — Must Detect | ReturnValue, ErrorReturn, SentinelError, ReceiverMutation, PointerArgMutation, ErrorSignal | 6 |
| P1 — High Value | SliceMutation, MapMutation, GlobalMutation, WriterOutput, HTTPResponseWrite, ChannelSend, ChannelClose, DeferredReturnMutation, GeneratorYield, ContainerMutation, StreamOutput | 11 |
| P2 — Important | FileSystemWrite, FileSystemDelete, FileSystemMeta, DatabaseWrite, DatabaseTransaction, GoroutineSpawn, Panic, CallbackInvocation, LogWrite, ContextCancellation, AsyncGeneratorYield, MetaprogrammingMutation, DescriptorEffect, ResourceManagement, ImportSideEffect, MonkeyPatch | 16 |
| P3 — Nice to Have | StdoutWrite, StderrWrite, EnvVarMutation, MutexOp, WaitGroupOp, AtomicOp, TimeDependency, ProcessExit, RecoverBehavior | 9 |
| P4 — Exotic | ReflectionMutation, UnsafeMutation, CgoCall, FinalizerRegistration, SyncPoolOp, ClosureCaptureMutation | 6 |

**Total: 48 effect types.**

### EC-002: P0 Zero Tolerance

A port MUST detect all P0 effects with zero false negatives and zero false positives. P0 effects are the function's direct observable outputs — return values, error returns, sentinel errors, receiver mutations, and pointer argument mutations. These are the foundation of contract coverage.

### EC-003: Effect Identity

Each detected side effect MUST have a stable, deterministic identifier. The reference implementation uses `sha256(package + function + effectType + location)` truncated to 8 hex characters, prefixed with `se-`. A port MAY use a different hashing scheme, but the ID MUST be:

- **Deterministic**: Same input produces the same ID across runs.
- **Unique**: No two distinct effects in the same function produce the same ID.
- **Stable**: The ID does not change unless the effect's location or type changes.

### EC-004: Effect Structure

Each detected side effect MUST carry these fields:

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Stable identifier (see EC-003) |
| `type` | enum | One of the 48 `SideEffectType` values |
| `tier` | enum | P0–P4, derived from type (see EC-001) |
| `location` | string | Source position (file:line:col) |
| `description` | string | Human-readable explanation |
| `target` | string | Affected entity (field name, variable, channel, return type, etc.) |
| `classification` | object or null | Classification result (see Classification Contract) |

### EC-005: Language Adaptation

The 48 effect types are defined in terms of programming language concepts. A port MUST map each type to its language equivalent:

- **ReturnValue** → any value returned from a function/method
- **ErrorReturn** → language-specific error mechanism (exceptions in Python, `Result::Err` in Rust, thrown errors in TypeScript)
- **SentinelError** → named, exported error constants/values that callers match against
- **ReceiverMutation** → mutation of `self`/`this`/the receiver object
- **PointerArgMutation** → mutation of a parameter passed by reference
- **SliceMutation** → mutation of a dynamic array/list parameter
- **MapMutation** → mutation of a dictionary/hash map parameter
- **GlobalMutation** → write to module-level / global state
- **WriterOutput** → write to an injected output stream (e.g., `io.Writer`, `Write` trait, `Writable`)
- **HTTPResponseWrite** → write to an HTTP response object
- **ChannelSend / ChannelClose** → send to or close a concurrency channel/queue
- **DeferredReturnMutation** → mutation of a return value in a deferred/finally block
- **GoroutineSpawn** → spawning a concurrent task (goroutine, thread, async task)
- **Panic** → unrecoverable error / panic / abort
- **CallbackInvocation** → invocation of a function parameter (callback, closure, handler)
- **CgoCall** → call to foreign function interface (FFI, ctypes, napi)
- **ErrorSignal** → language-neutral error signaling mechanism (`raise`, `throw`, `return err`)
- **GeneratorYield** → yielding a value from a generator/iterator (`yield` in Python, `yield return` in C#)
- **ContainerMutation** → mutation of a generic container parameter (set, deque, priority queue)
- **StreamOutput** → write to a generic output stream
- **AsyncGeneratorYield** → yielding from an async generator (`async def gen(): yield x`)
- **MetaprogrammingMutation** → runtime mutation of program structure (`setattr`, metaclasses)
- **DescriptorEffect** → side effect via descriptor protocol (`__set__`, `__delete__`)
- **ResourceManagement** → acquisition or release of an external resource (context managers, RAII)
- **ImportSideEffect** → module import that triggers observable side effects
- **MonkeyPatch** → runtime replacement of functions/methods/attributes

Types without a direct equivalent in the target language SHOULD be omitted from detection but MUST remain in the taxonomy for compatibility. For example, `CgoCall` maps to FFI in any language, but `SyncPoolOp` may not have an equivalent.

---

## 2. Classification Contract

Classification determines whether a side effect is **contractual** (part of the function's intended behavior), **incidental** (an implementation detail), or **ambiguous** (insufficient evidence).

### CC-001: Confidence Scoring Formula

A port MUST compute a confidence score using this formula:

```text
score = base_confidence + tier_boost(effect_type) + sum(signal_weights) - contradiction_penalty
```

Where:

| Component | Value | Description |
|-----------|-------|-------------|
| `base_confidence` | 50 | Neutral starting point for all effects |
| `tier_boost(P0)` | +25 | P0 effects start at 75 (definitionally contractual) |
| `tier_boost(P1)` | +10 | P1 effects start at 60 |
| `tier_boost(P2–P4)` | 0 | No boost; contractual nature depends on context |
| `signal_weights` | varies | Sum of all signal weights (positive and negative) |
| `contradiction_penalty` | -20 | Applied when both positive AND negative signals exist |

The effective starting scores are:

- **P0 effects**: 75 (base 50 + boost 25)
- **P1 effects**: 60 (base 50 + boost 10)
- **P2–P4 effects**: 50 (base 50 + boost 0)

### CC-002: Score Clamping

The final confidence score MUST be clamped to the range [0, 100].

### CC-003: Label Thresholds

A port MUST classify effects into three labels based on the clamped confidence score:

| Label | Condition | Default Threshold |
|-------|-----------|-------------------|
| `contractual` | score >= contractual_threshold | 80 |
| `ambiguous` | incidental_threshold <= score < contractual_threshold | [50, 80) |
| `incidental` | score < incidental_threshold | < 50 |

The thresholds MUST be configurable. The defaults are `contractual = 80` and `incidental = 50`.

### CC-004: Contradiction Detection

When both positive-weight and negative-weight signals are present for the same effect, a port MUST apply a contradiction penalty of -20 to the score. The contradiction signal MUST be recorded in the signal list with source `"contradiction"` and weight `-20`.

### CC-005: Five Signal Categories

A port MUST implement signal analyzers for these five categories. The specific detection mechanism is language-dependent, but the semantic intent and weight ranges MUST be preserved:

#### Signal 1: Interface Satisfaction (max weight: +30)

Checks whether the function's receiver/class implements an interface/trait/protocol that declares this method. If the method is part of an interface contract, the side effect is strong evidence of contractual behavior.

- **Weight**: +30 when the method satisfies an interface
- **Weight**: 0 (no signal) when no interface match is found
- **Language mapping**: Go interfaces, Rust traits, Python protocols/ABCs, TypeScript interfaces

#### Signal 2: API Visibility (max weight: +20)

Checks whether the function and its types are part of the public API surface. Three independent dimensions contribute:

| Dimension | Weight | Condition |
|-----------|--------|-----------|
| Exported function | +8 | Function is public/exported |
| Exported return type | +6 | Return type is public/exported |
| Exported receiver type | +6 | Receiver/class type is public/exported |

The total is clamped to +20. A fully public method on a public type returning a public type scores +20.

- **Language mapping**: Go exported names (uppercase), Python `__all__` / no underscore prefix, Rust `pub`, TypeScript `export`

#### Signal 3: Caller Dependency (max weight: +15)

Counts how many distinct modules/packages call this function. More callers = stronger evidence of contractual behavior.

| Caller Count | Weight |
|-------------|--------|
| 0 | 0 (no signal) |
| 1 | +5 |
| 2–3 | +10 |
| 4+ | +15 |

#### Signal 4: Naming Convention (max weight: +10, special case: +30)

Checks the function name against language community naming conventions.

**Contractual prefixes** (weight +10): `Get`, `Fetch`, `Load`, `Read`, `Save`, `Write`, `Update`, `Set`, `Delete`, `Remove`, `Handle`, `Process`, `Compute`, `Analyze`, `Classify`, `Parse`, `Build`, `New`

Each prefix implies specific effect types are contractual. For example, `Get*` implies `ReturnValue` is contractual; `Save*` implies `ReceiverMutation`, `PointerArgMutation`, and `ErrorReturn` are contractual. The signal fires only when the effect type matches the prefix's implied types (or when the prefix implies all types, as with `Handle*` and `Process*`).

**Incidental prefixes** (weight -10): `log`, `Log`, `debug`, `Debug`, `trace`, `Trace`, `print`, `Print`

**Special case — Sentinel errors** (weight +30): Variables/constants named `Err*` with type `SentinelError` receive +30 instead of +10. This is because sentinel error declarations cannot receive interface, visibility, or documentation signals (they are package-level variables, not methods), so a stronger naming weight is the only path to the contractual threshold.

#### Signal 5: Documentation (max weight: +15)

Parses the function's documentation comment for behavioral declarations.

**Contractual keywords** (weight +15 for direct match, +5 for indirect): `returns`, `writes`, `modifies`, `updates`, `sets`, `persists`, `stores`, `deletes`, `removes`

Each keyword implies specific effect types. A direct match (keyword implies the detected effect type) scores +15. An indirect match (keyword found but effect type doesn't match) scores +5.

**Incidental keywords** (weight -15): `logs`, `prints`, `traces`, `debugs`

- **Language mapping**: Go GoDoc, Python docstrings, Rust doc comments (`///`), TypeScript JSDoc

### CC-006: Signal Recording

Each signal MUST be recorded in the classification result with at minimum:

| Field | Description |
|-------|-------------|
| `source` | Signal category identifier (e.g., `"interface"`, `"visibility"`, `"caller"`, `"naming"`, `"godoc"`) |
| `weight` | Numeric contribution to the confidence score (can be negative) |

A port SHOULD also record `reasoning` (human-readable explanation) and MAY record `source_file` and `excerpt` for verbose output.

---

## 3. Scoring Contract

Scoring combines cyclomatic complexity with coverage to produce risk metrics.

### SC-001: CRAP Formula

A port MUST compute the CRAP score using this exact formula:

```text
CRAP(m) = complexity² × (1 - coverage/100)³ + complexity
```

Where:

- `complexity` = cyclomatic complexity of the function (integer >= 1)
- `coverage` = line coverage percentage (float, 0–100)

**Properties**:

- At 100% coverage: `CRAP = complexity` (the cubic term vanishes)
- At 0% coverage: `CRAP = complexity² + complexity`
- Higher complexity amplifies the penalty for missing coverage

### SC-002: GazeCRAP Formula

GazeCRAP uses the same formula as CRAP but substitutes **contract coverage** for line coverage:

```text
GazeCRAP(m) = complexity² × (1 - contract_coverage/100)³ + complexity
```

Where `contract_coverage` is the percentage of contractual side effects that are asserted on by tests (0–100).

GazeCRAP is only available when the classification and quality assessment pipelines have run. When contract coverage data is unavailable, GazeCRAP MUST be null/absent (not zero).

### SC-003: CRAPload and GazeCRAPload

- **CRAPload** = count of functions where `CRAP >= crap_threshold` (default: 15)
- **GazeCRAPload** = count of functions where `GazeCRAP >= gaze_crap_threshold` (default: 15)

Both thresholds MUST be configurable.

### SC-004: Quadrant Classification

When both CRAP and GazeCRAP are available, a port MUST classify each function into exactly one of four quadrants:

| Quadrant | Condition | Meaning |
|----------|-----------|---------|
| Q1 Safe | CRAP < threshold AND GazeCRAP < threshold | Low risk, well tested |
| Q2 Complex But Tested | CRAP >= threshold AND GazeCRAP < threshold | Complex but contracts are verified |
| Q3 Simple But Underspecified | CRAP < threshold AND GazeCRAP >= threshold | Simple but contracts are not verified |
| Q4 Dangerous | CRAP >= threshold AND GazeCRAP >= threshold | Complex AND contracts are not verified |

The two thresholds (CRAP threshold and GazeCRAP threshold) are independent and separately configurable.

### SC-005: Fix Strategy Assignment

Each function in the CRAPload (CRAP >= threshold) MUST be assigned exactly one fix strategy. Functions below the threshold MUST NOT have a fix strategy.

The assignment rules, evaluated in order:

| Rule | Condition | Strategy | Meaning |
|------|-----------|----------|---------|
| 1 | complexity >= crap_threshold AND line_coverage == 0 | `decompose_and_test` | Too complex AND untested |
| 2 | complexity >= crap_threshold AND line_coverage > 0 | `decompose` | Too complex; even 100% coverage can't help |
| 3 | quadrant == Q3 (Simple But Underspecified) | `add_assertions` | Has line coverage but lacks contract assertions |
| 4 | (default) | `add_tests` | Needs test coverage |

**Why Rule 2 works**: When `complexity >= crap_threshold`, then `CRAP(100% coverage) = complexity >= threshold`, so no amount of coverage can bring CRAP below threshold. The function must be decomposed.

### SC-006: Recommended Actions Ordering

Recommended actions MUST be sorted by fix strategy priority (easiest wins first), then by CRAP score descending within each strategy:

| Priority | Strategy | Rationale |
|----------|----------|-----------|
| 0 (first) | `add_tests` | Lowest effort — just add tests |
| 1 | `add_assertions` | Add assertions to existing tests |
| 2 | `decompose_and_test` | Refactor then test |
| 3 (last) | `decompose` | Refactor only (already has coverage) |

The list MUST be capped at 20 entries.

---

## 4. Output Contract

### OC-001: Dual Format

A port MUST support at minimum two output formats:

- **JSON**: Machine-readable, schema-validated output for CI pipelines and tooling
- **Text**: Human-readable terminal output

### OC-002: JSON Field Names

JSON output MUST use `snake_case` field names. The canonical field names from the reference implementation (e.g., `side_effects`, `line_coverage`, `contract_coverage`, `gaze_crap`, `fix_strategy`, `quadrant_counts`) MUST be preserved for cross-implementation compatibility.

### OC-003: Nullable Fields

Fields that depend on optional capabilities (e.g., `gaze_crap`, `contract_coverage`, `quadrant`) MUST be null/absent when the capability has not run — not zero-valued. This allows consumers to distinguish "not computed" from "computed as zero."

---

## 5. Contract Verification

A port SHOULD include a conformance test suite that verifies each contract ID. Suggested approach:

1. Create synthetic functions with known complexity, coverage, and side effects.
2. Verify that the CRAP formula produces the expected score (SC-001).
3. Verify that tier boosts produce the expected starting scores (CC-001).
4. Verify that label thresholds produce the expected classifications (CC-003).
5. Verify that quadrant classification matches the truth table (SC-004).
6. Verify that fix strategy assignment follows the rule order (SC-005).

Each contract ID (EC-001 through OC-003) maps to one or more test cases. A passing conformance suite demonstrates that the port honors all behavioral contracts.
