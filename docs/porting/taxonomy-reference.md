# Taxonomy Reference

Canonical reference tables for the Gaze effect taxonomy, scoring formulas, and classification thresholds. This is the spec sheet — use it while implementing.

For explanations and rationale, see [contracts.md](contracts.md). For capability requirements, see [requirements.md](requirements.md).

---

## Effect Types

48 types across 5 priority tiers.

**Status key**: Implemented = detected by the reference Go implementation. Defined = specified in the taxonomy but detection not yet implemented.

| Name | Tier | Category | Status |
|------|------|----------|--------|
| ReturnValue | P0 | Return | Implemented |
| ErrorReturn | P0 | Return | Implemented |
| SentinelError | P0 | Return | Implemented |
| ReceiverMutation | P0 | Mutation | Implemented |
| PointerArgMutation | P0 | Mutation | Implemented |
| ErrorSignal | P0 | Return | Defined |
| SliceMutation | P1 | Mutation | Implemented |
| MapMutation | P1 | Mutation | Implemented |
| GlobalMutation | P1 | Mutation | Implemented |
| WriterOutput | P1 | I/O | Implemented |
| HTTPResponseWrite | P1 | I/O | Implemented |
| ChannelSend | P1 | Concurrency | Implemented |
| ChannelClose | P1 | Concurrency | Implemented |
| DeferredReturnMutation | P1 | Mutation | Implemented |
| GeneratorYield | P1 | Return | Defined |
| ContainerMutation | P1 | Mutation | Defined |
| StreamOutput | P1 | I/O | Defined |
| FileSystemWrite | P2 | I/O | Implemented |
| FileSystemDelete | P2 | I/O | Implemented |
| FileSystemMeta | P2 | I/O | Implemented |
| DatabaseWrite | P2 | I/O | Implemented |
| DatabaseTransaction | P2 | I/O | Implemented |
| GoroutineSpawn | P2 | Concurrency | Implemented |
| Panic | P2 | Control Flow | Implemented |
| CallbackInvocation | P2 | Control Flow | Implemented |
| LogWrite | P2 | I/O | Implemented |
| ContextCancellation | P2 | Concurrency | Implemented |
| AsyncGeneratorYield | P2 | Return | Defined |
| MetaprogrammingMutation | P2 | Mutation | Defined |
| DescriptorEffect | P2 | Mutation | Defined |
| ResourceManagement | P2 | Resource | Defined |
| ImportSideEffect | P2 | Control Flow | Defined |
| MonkeyPatch | P2 | Mutation | Defined |
| StdoutWrite | P3 | I/O | Defined |
| StderrWrite | P3 | I/O | Defined |
| EnvVarMutation | P3 | Mutation | Defined |
| MutexOp | P3 | Concurrency | Defined |
| WaitGroupOp | P3 | Concurrency | Defined |
| AtomicOp | P3 | Concurrency | Defined |
| TimeDependency | P3 | External | Defined |
| ProcessExit | P3 | Control Flow | Defined |
| RecoverBehavior | P3 | Control Flow | Defined |
| ReflectionMutation | P4 | Exotic | Defined |
| UnsafeMutation | P4 | Exotic | Defined |
| CgoCall | P4 | Exotic | Defined |
| FinalizerRegistration | P4 | Exotic | Defined |
| SyncPoolOp | P4 | Exotic | Defined |
| ClosureCaptureMutation | P4 | Exotic | Defined |

### Tier Summary

| Tier | Label | Count | Detection Requirement |
|------|-------|-------|-----------------------|
| P0 | Must Detect | 6 | Zero false negatives, zero false positives |
| P1 | High Value | 11 | Must detect; false positives acceptable if documented |
| P2 | Important | 16 | Should detect; partial detection acceptable |
| P3 | Nice to Have | 9 | May detect |
| P4 | Exotic | 6 | May detect |

---

## Scoring Formulas

| Formula | Definition | Variables |
|---------|-----------|-----------|
| CRAP | `comp² × (1 - cov/100)³ + comp` | `comp` = cyclomatic complexity (int >= 1); `cov` = line coverage % (0–100) |
| GazeCRAP | `comp² × (1 - cc/100)³ + comp` | `comp` = cyclomatic complexity (int >= 1); `cc` = contract coverage % (0–100) |
| CRAPload | `count(functions where CRAP >= threshold)` | `threshold` default = 15 |
| GazeCRAPload | `count(functions where GazeCRAP >= threshold)` | `threshold` default = 15 |

### Reference Values

| Complexity | Line Coverage | CRAP |
|-----------|--------------|------|
| 1 | 100% | 1.0 |
| 1 | 0% | 2.0 |
| 1 | 50% | 1.125 |
| 5 | 100% | 5.0 |
| 5 | 50% | 8.125 |
| 5 | 0% | 30.0 |
| 10 | 100% | 10.0 |
| 10 | 50% | 22.5 |
| 10 | 0% | 110.0 |
| 15 | 100% | 15.0 |
| 15 | 0% | 240.0 |
| 20 | 100% | 20.0 |
| 20 | 50% | 70.0 |

---

## Classification Thresholds

| Label | Condition | Default |
|-------|-----------|---------|
| Contractual | `score >= contractual_threshold` | 80 |
| Ambiguous | `incidental_threshold <= score < contractual_threshold` | [50, 80) |
| Incidental | `score < incidental_threshold` | 50 |

### Confidence Score Formula

```text
score = clamp(base + tier_boost + sum(signal_weights) - contradiction_penalty, 0, 100)
```

| Component | Value |
|-----------|-------|
| Base confidence | 50 |
| Tier boost (P0) | +25 |
| Tier boost (P1) | +10 |
| Tier boost (P2–P4) | 0 |
| Contradiction penalty | -20 (applied when both positive and negative signals exist) |

### Effective Starting Scores

| Tier | Starting Score | Distance to Contractual (80) |
|------|---------------|------------------------------|
| P0 | 75 | 5 points (one small positive signal reaches contractual) |
| P1 | 60 | 20 points (needs moderate positive evidence) |
| P2–P4 | 50 | 30 points (needs strong positive evidence) |

---

## Signal Weights

| Signal | Source ID | Max Positive | Max Negative | Notes |
|--------|----------|-------------|-------------|-------|
| Interface Satisfaction | `interface` | +30 | 0 | Method satisfies an interface/trait/protocol |
| API Visibility | `visibility` | +20 | 0 | Sum of: exported function (+8), exported return type (+6), exported receiver type (+6); clamped to 20 |
| Caller Dependency | `caller` | +15 | 0 | 1 caller = +5, 2–3 = +10, 4+ = +15 |
| Naming Convention | `naming` | +10 | -10 | Contractual prefixes vs. incidental prefixes |
| Naming (Sentinel) | `naming` | +30 | — | `Err*` sentinel errors only; exceeds normal max |
| Documentation (direct) | `godoc` | +15 | -15 | Keyword matches the detected effect type |
| Documentation (indirect) | `godoc_keyword_indirect` | +5 | — | Keyword found but effect type doesn't match |
| Documentation (architecture doc) | `architecture_doc` | +25 | -25 | External analyzers only; annotation in non-README Markdown |
| Documentation (README) | `readme` | +15 | -15 | External analyzers only; annotation in a `README*` document |
| Documentation (sidecar) | `sidecar` | +30 | -30 | External analyzers only; entry in `.uf/gaze/contracts.yaml`/`.json` |
| Contradiction | `contradiction` | — | -20 | Auto-applied when positive + negative signals coexist |

---

## Quadrant Classification

| Quadrant | ID | CRAP | GazeCRAP | Meaning |
|----------|----|------|----------|---------|
| Safe | Q1_Safe | < threshold | < threshold | Low risk, well tested |
| Complex But Tested | Q2_ComplexButTested | >= threshold | < threshold | Complex but contracts verified |
| Simple But Underspecified | Q3_SimpleButUnderspecified | < threshold | >= threshold | Simple but contracts not verified |
| Dangerous | Q4_Dangerous | >= threshold | >= threshold | Complex AND contracts not verified |

Default thresholds: CRAP = 15, GazeCRAP = 15 (independently configurable).

---

## Fix Strategies

Assigned to functions in the CRAPload (CRAP >= threshold). Evaluated in order; first matching rule wins.

| Priority | Strategy | Condition | Action |
|----------|----------|-----------|--------|
| 0 | `add_tests` | Default (none of the below) | Add tests to increase coverage |
| 1 | `add_assertions` | Quadrant == Q3 | Add assertions to existing tests |
| 2 | `decompose_and_test` | complexity >= threshold AND coverage == 0 | Split function, then add tests |
| 3 | `decompose` | complexity >= threshold AND coverage > 0 | Split function into smaller units |

**Rule evaluation order in code**: Rules 2 and 3 (complexity >= threshold) are checked first, then Rule 1 (Q3), then Rule 0 (default).

---

## Contract Coverage Reasons

When contract coverage is 0% or otherwise diagnostic, a reason field explains why:

| Reason | Meaning |
|--------|---------|
| `all_effects_ambiguous` | All effects classified as ambiguous; no contractual effects to cover |
| `no_effects_detected` | Function has no detected side effects |
| `no_test_coverage` | Effects detected but no test targets this function |
| `no_assertions_mapped` | Effects exist and tests exist but no assertions mapped to effects |
| *(empty)* | Normal coverage — no special explanation needed |

---

## Assertion Types

| Type | Description |
|------|-------------|
| `equality` | Value equality check (e.g., `assertEqual`, `==`) |
| `error_check` | Error value check (e.g., `if err != nil`) |
| `nil_check` | Nil/null check |
| `diff_check` | Structural diff comparison |
| `custom` | Framework-specific or unrecognized assertion pattern |

---

## JSON Field Name Reference

Canonical `snake_case` field names for cross-implementation compatibility:

| Field | Parent Object | Type |
|-------|--------------|------|
| `side_effects` | AnalysisResult | array |
| `side_effect_id` | AssertionMapping | string |
| `line_coverage` | Score | float |
| `contract_coverage` | Score | float (nullable) |
| `gaze_crap` | Score | float (nullable) |
| `fix_strategy` | Score | string (nullable) |
| `quadrant` | Score | string (nullable) |
| `quadrant_counts` | Summary | object (nullable) |
| `fix_strategy_counts` | Summary | object (nullable) |
| `crap_threshold` | Summary | float |
| `gaze_crap_threshold` | Summary | float (nullable) |
| `gaze_crapload` | Summary | int (nullable) |
| `avg_contract_coverage` | Summary | float (nullable) |
| `recommended_actions` | Summary | array (nullable) |
| `ssa_degraded_packages` | Summary | array (nullable) |
| `contract_coverage_reason` | Score | string (nullable) |
| `effect_confidence_range` | Score | [int, int] (nullable) |
| `assertion_count` | QualityReport | int |
| `unmapped_assertions` | QualityReport | array |
| `assertion_detection_confidence` | QualityReport | int |
| `no_contract_expected` | ContractCoverage | bool (nullable) |
| `reason` | ContractCoverage | string (nullable) |
