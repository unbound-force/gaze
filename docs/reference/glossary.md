# Glossary

Canonical definitions for all domain-specific terms used across Gaze documentation. Other pages link here on first use of a term — definitions are never duplicated.

---

## Ambiguous

A [classification label](#classification-label) assigned to a [side effect](#side-effect) whose [confidence score](#confidence-score) falls between the incidental and contractual thresholds (default: 50–79). Ambiguous effects are excluded from [contract coverage](#contract-coverage) calculations because there is insufficient evidence to determine whether they are part of the function's public obligation.

## Assertion Mapping

The process of linking a test assertion (e.g., `if got != want`) to the specific [side effect](#side-effect) it verifies. Gaze uses four mechanical passes — direct identity, indirect root, helper bridge, and inline call — to trace assertion expressions back to side effect objects. Each mapping carries a [confidence score](#confidence-score). Unmapped assertions are reported with a reason (helper parameter, inline call, or no effect match).

## Behavioral Contract

The set of observable [side effects](#side-effect) that a function is obligated to produce as part of its public interface. A function's behavioral contract includes its return values, error returns, and any state mutations that callers depend on. Effects classified as [contractual](#contractual) form the behavioral contract; [incidental](#incidental) effects are implementation details outside the contract.

---

## Classification Label

One of three labels assigned to each [side effect](#side-effect) by the classification engine: [contractual](#contractual), [incidental](#incidental), or [ambiguous](#ambiguous). The label is determined by the [confidence score](#confidence-score) relative to configurable thresholds.

## Classification Signal

A single piece of evidence that contributes to a [side effect's](#side-effect) [confidence score](#confidence-score). Gaze uses five signal analyzers — **interface** (effect appears in an interface method), **visibility/caller** (callers depend on the effect), **naming** (function/parameter naming conventions), **godoc** (documentation mentions the effect), and **architecture doc** (project documentation references the behavior). Each signal carries a positive or negative weight. When both positive and negative signals are present, a contradiction penalty (−20) is applied.

## Confidence Score

A numeric value (0–100) representing how strongly the evidence supports classifying a [side effect](#side-effect) as [contractual](#contractual). Computed by starting from a base score that depends on the effect's [tier](#tier) — 75 for P0, 60 for P1, 50 for P2–P4 — and adding the weights of all [classification signals](#classification-signal). The final score is clamped to 0–100. Scores at or above the contractual threshold (default: 80) yield a "contractual" label; scores below the incidental threshold (default: 50) yield "incidental"; scores in between yield "ambiguous."

## Contract Coverage

The percentage of a function's [contractual](#contractual) [side effects](#side-effect) that at least one test assertion verifies. This is Gaze's primary test quality metric.

```text
Contract Coverage % = (contractual effects asserted on / total contractual effects) × 100
```

Unlike line coverage, which measures whether code was *executed*, contract coverage measures whether tests *verified the function's observable obligations*. A function can have 90% line coverage but 0% contract coverage if its tests never assert on return values, error paths, or state mutations.

## Contractual

A [classification label](#classification-label) assigned to a [side effect](#side-effect) whose [confidence score](#confidence-score) meets or exceeds the contractual threshold (default: 80). Contractual effects are part of the function's [behavioral contract](#behavioral-contract) — callers depend on them, and tests should assert on them.

## CRAP

**Change Risk Anti-Patterns.** A composite metric that combines [cyclomatic complexity](#cyclomatic-complexity) with line coverage to identify functions that are both complex and under-tested. Higher scores indicate higher risk when changing the function.

```text
CRAP(m) = complexity² × (1 − lineCoverage/100)³ + complexity
```

A function with complexity 5 and 0% coverage has CRAP = 30. The same function with 100% coverage has CRAP = 5. The default threshold is 15.

## CRAPload

The count of functions in a project whose [CRAP](#crap) score meets or exceeds the CRAP threshold (default: 15). CRAPload is a project-level metric — a single number that summarizes how many functions are risky to change. Used as a CI quality gate via the `--max-crapload` flag.

## Cyclomatic Complexity

A measure of the number of linearly independent paths through a function's source code. Each decision point (if, for, switch case, &&, ||) adds one to the count. A function with no branches has complexity 1. Gaze uses the `gocyclo` library to compute this metric. Cyclomatic complexity is one of the two inputs to the [CRAP](#crap) formula.

---

## Diataxis

A documentation framework that organizes content into four categories: **tutorials** (learning-oriented), **how-to guides** (task-oriented), **reference** (information-oriented), and **explanation** (understanding-oriented). Gaze's `docs/` directory uses a Diataxis-inspired structure with sections for getting-started, concepts, reference, guides, architecture, and porting.

---

## Fix Strategy

A deterministic remediation label assigned to each function in the [CRAPload](#crapload), indicating the most effective action to reduce its [CRAP](#crap) score. Four strategies exist:

| Strategy | When Assigned | Action |
|----------|--------------|--------|
| **`decompose`** | Complexity ≥ CRAP threshold (even 100% coverage can't help) | Split the function into smaller units |
| **`add_tests`** | Zero line coverage, complexity below threshold | Write tests to cover the function |
| **`add_assertions`** | Has line coverage but is in Q3 (lacks contract assertions) | Add assertions to existing tests that verify observable behavior |
| **`decompose_and_test`** | High complexity AND zero coverage | Both decompose and add tests |

---

## GazeCRAP

A variant of the [CRAP](#crap) formula that replaces line coverage with [contract coverage](#contract-coverage). GazeCRAP measures whether tests *verify observable behavior*, not just whether they *execute code*.

```text
GazeCRAP(m) = complexity² × (1 − contractCoverage/100)³ + complexity
```

A function can have a low CRAP score (well-covered by line count) but a high GazeCRAP score (tests don't assert on its contractual effects). The combination of CRAP and GazeCRAP determines the function's [quadrant](#q1-safe--q2-complex-but-tested--q3-simple-but-underspecified--q4-dangerous).

## GazeCRAPload

The count of functions whose [GazeCRAP](#gazecrap) score meets or exceeds the GazeCRAP threshold (default: 15). Analogous to [CRAPload](#crapload) but based on contract coverage instead of line coverage. Used as a CI quality gate via the `--max-gaze-crapload` flag.

---

## Incidental

A [classification label](#classification-label) assigned to a [side effect](#side-effect) whose [confidence score](#confidence-score) falls below the incidental threshold (default: 50). Incidental effects are implementation details — logging calls, goroutine lifecycle, internal stdout writes — that callers do not depend on. Tests that assert on incidental effects are flagged as [over-specified](#over-specification-score).

---

## Over-Specification Score

A measure of how many test assertions target [incidental](#incidental) [side effects](#side-effect) — implementation details that are not part of the function's [behavioral contract](#behavioral-contract). Over-specified tests break during refactoring even when the function's actual contract is preserved.

```text
Over-Specification Ratio = incidental assertions / total mapped assertions
```

Reported as both a count and a ratio (0.0–1.0).

---

## P0 / P1 / P2 / P3 / P4

The five priority [tiers](#tier) in Gaze's side effect taxonomy, ordered by detection priority and contractual significance:

| Tier | Name | Examples | Detection |
|------|------|----------|-----------|
| **P0** | Must Detect | `ReturnValue`, `ErrorReturn`, `SentinelError`, `ReceiverMutation`, `PointerArgMutation` | Implemented |
| **P1** | High Value | `SliceMutation`, `MapMutation`, `GlobalMutation`, `WriterOutput`, `HTTPResponseWrite`, `ChannelSend`, `ChannelClose`, `DeferredReturnMutation` | Implemented |
| **P2** | Important | `FileSystemWrite`, `DatabaseWrite`, `GoroutineSpawn`, `Panic`, `LogWrite`, and others | Implemented |
| **P3** | Nice to Have | `StdoutWrite`, `StderrWrite`, `EnvVarMutation`, `MutexOp`, `TimeDependency`, and others | Defined — detection not yet implemented |
| **P4** | Exotic | `ReflectionMutation`, `UnsafeMutation`, `CgoCall`, `FinalizerRegistration`, and others | Defined — detection not yet implemented |

P0 effects receive a +25 [confidence score](#confidence-score) boost (starting at 75 instead of 50), reflecting that a function's direct outputs are definitionally [contractual](#contractual). P1 effects receive +10 (starting at 60).

---

## Q1 Safe / Q2 Complex But Tested / Q3 Simple But Underspecified / Q4 Dangerous

The four quadrants that classify a function based on its [CRAP](#crap) and [GazeCRAP](#gazecrap) scores relative to their respective thresholds:

| | Low GazeCRAP (< threshold) | High GazeCRAP (≥ threshold) |
|---|---|---|
| **Low CRAP (< threshold)** | **Q1 Safe** — Low complexity, tests verify the contract | **Q3 Simple But Underspecified** — Low complexity, but tests don't verify observable behavior |
| **High CRAP (≥ threshold)** | **Q2 Complex But Tested** — High complexity, but tests verify the contract | **Q4 Dangerous** — High complexity AND tests don't verify the contract |

**Q4 Dangerous** is the highest-priority remediation target. **Q3** functions are the best candidates for the `add_assertions` [fix strategy](#fix-strategy) — they have line coverage but lack contract-level assertions.

---

## Side Effect

Any observable change that a function produces beyond its return value. In Gaze's taxonomy, side effects include return values, error returns, state mutations (receiver, pointer argument, slice, map, global), I/O operations (file system, database, network, stdout/stderr), concurrency operations (goroutine spawn, channel send/close), and more. Gaze detects 37 side effect types organized into five [tiers](#tier) (P0–P4). Each detected effect is assigned a stable ID, a [classification label](#classification-label), and a [confidence score](#confidence-score).

## SSA

**Static Single Assignment.** An intermediate representation of Go code where every variable is assigned exactly once. Gaze uses SSA (via `golang.org/x/tools/go/ssa`) for mutation tracking — detecting receiver mutations, pointer argument mutations, and other state changes that cannot be reliably identified from the AST alone. SSA construction can fail for some packages (e.g., due to upstream tooling bugs); when this happens, Gaze degrades gracefully, reporting partial results with an `ssa_degraded` flag.

---

## Tier

A priority level (P0–P4) assigned to each [side effect](#side-effect) type in Gaze's taxonomy. Tiers determine detection priority, [confidence score](#confidence-score) boosts, and the order in which Gaze implements detection for new effect types. See [P0 / P1 / P2 / P3 / P4](#p0--p1--p2--p3--p4) for the full breakdown.
