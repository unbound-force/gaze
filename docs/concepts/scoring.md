# Scoring

Gaze computes two risk scores for every function: **CRAP** (using line coverage) and **[GazeCRAP](../reference/glossary.md#gazecrap)** (using [contract coverage](../reference/glossary.md#contract-coverage)). Together, they classify functions into four [quadrants](../reference/glossary.md#q1-safe--q2-complex-but-tested--q3-simple-but-underspecified--q4-dangerous) and assign [fix strategies](../reference/glossary.md#fix-strategy) that tell you exactly what to do about risky code.

## The CRAP Formula

CRAP (Change Risk Anti-Patterns) measures the risk of changing a function by combining cyclomatic complexity with test coverage:

```text
CRAP(m) = comp^2 * (1 - cov/100)^3 + comp
```

Where:

- **comp** = cyclomatic complexity (minimum 1)
- **cov** = line coverage percentage (0–100)

### Key Properties

- At **100% coverage**, CRAP equals the complexity itself (`comp^2 * 0 + comp = comp`). A function with complexity 5 and full coverage has CRAP = 5.
- At **0% coverage**, CRAP equals `comp^2 + comp`. A function with complexity 5 and no coverage has CRAP = 30.
- The cubic exponent on the uncovered fraction means coverage has a strong effect — going from 0% to 50% coverage dramatically reduces CRAP.

### CRAPload

The [CRAPload](../reference/glossary.md#crapload) is the count of functions with a CRAP score at or above the threshold (default: 15). It's the single number that answers "how many functions are risky to change?"

## The GazeCRAP Formula

GazeCRAP uses the same formula as CRAP but substitutes [contract coverage](../reference/glossary.md#contract-coverage) for line coverage:

```text
GazeCRAP(m) = comp^2 * (1 - contract_cov/100)^3 + comp
```

Where:

- **comp** = cyclomatic complexity
- **contract_cov** = contract coverage percentage (0–100), the ratio of [contractual](../reference/glossary.md#contractual) side effects that are asserted on by tests

### Why GazeCRAP Matters

A function can have 100% line coverage but 0% contract coverage — every line executes during tests, but no test actually verifies the function's observable behavior. CRAP would say this function is safe. GazeCRAP reveals the truth: the tests are executing code without asserting on anything meaningful.

### GazeCRAPload

The [GazeCRAPload](../reference/glossary.md#gazecrapload) is the count of functions with a GazeCRAP score at or above the GazeCRAP threshold (default: 15). It's available only when contract coverage data is computed.

## The Four Quadrants

Every function with both CRAP and GazeCRAP scores is classified into one of four quadrants based on whether each score exceeds its respective threshold:

```text
                    GazeCRAP < threshold    GazeCRAP >= threshold
                  ┌─────────────────────┬─────────────────────────────┐
CRAP < threshold  │  Q1 Safe            │  Q3 Simple But              │
                  │                     │     Underspecified          │
                  ├─────────────────────┼─────────────────────────────┤
CRAP >= threshold │  Q2 Complex But     │  Q4 Dangerous               │
                  │     Tested          │                             │
                  └─────────────────────┴─────────────────────────────┘
```

| Quadrant | CRAP | GazeCRAP | Meaning |
|---|---|---|---|
| **Q1 Safe** | Below threshold | Below threshold | Low complexity, well-tested with meaningful assertions. No action needed. |
| **Q2 Complex But Tested** | At/above threshold | Below threshold | High complexity but tests verify the contract. Consider decomposing for maintainability, but tests are solid. |
| **Q3 Simple But Underspecified** | Below threshold | At/above threshold | Low complexity but tests don't verify observable behavior. Tests execute code without asserting on contractual effects. Add assertions. |
| **Q4 Dangerous** | At/above threshold | At/above threshold | High complexity AND poor contract coverage. The riskiest code to change. Needs both decomposition and better tests. |

## Fix Strategies

Every function in the [CRAPload](../reference/glossary.md#crapload) (CRAP score at or above the threshold) receives a deterministic [fix strategy](../reference/glossary.md#fix-strategy) label that tells you the recommended remediation action:

| Strategy | When It Applies | What To Do |
|---|---|---|
| `decompose` | Complexity >= threshold (even 100% coverage can't bring CRAP below threshold, since CRAP at full coverage = complexity) | Split the function into smaller units. Reduce cyclomatic complexity. |
| `add_tests` | Zero line coverage and complexity below threshold | Write tests for this function. Coverage alone will bring CRAP below threshold. |
| `add_assertions` | Function is in Q3 (Simple But Underspecified) — has line coverage but lacks contract-level assertions | Add assertions to existing tests. The tests execute the code but don't verify observable behavior. |
| `decompose_and_test` | Complexity >= threshold AND zero line coverage | Both decompose the function AND write tests. The function is too complex and completely untested. |

### Strategy Assignment Logic

The fix strategy is assigned by evaluating conditions in this order:

1. If CRAP < threshold: no strategy (function is healthy)
2. If complexity >= threshold AND zero coverage: `decompose_and_test`
3. If complexity >= threshold: `decompose`
4. If quadrant is Q3 (Simple But Underspecified): `add_assertions`
5. Otherwise: `add_tests`

### Recommended Actions

The CRAP report includes a prioritized `recommended_actions` list (top 20 functions) sorted by fix strategy priority, then CRAP score descending. The priority order is:

1. `add_tests` — easiest wins first (just add tests)
2. `add_assertions` — existing tests need assertions
3. `decompose_and_test` — needs both work
4. `decompose` — structural change required

## Worked Examples

### Example 1: Simple Function, No Tests

```go
func Add(a, b int) int {  // complexity: 1, line coverage: 0%
    return a + b
}
```

- **CRAP** = 1^2 \* (1 - 0/100)^3 + 1 = 1 \* 1 + 1 = **2**
- Below threshold (15). No fix strategy needed.

### Example 2: Complex Function, Good Coverage

```go
func ProcessOrder(order *Order) error {  // complexity: 12, line coverage: 85%
    // 12 branches handling various order states...
}
```

- **CRAP** = 12^2 \* (1 - 85/100)^3 + 12 = 144 \* 0.003375 + 12 = 0.486 + 12 = **12.5**
- Below threshold (15). Safe despite high complexity because coverage is strong.

### Example 3: Complex Function, No Coverage

```go
func ValidateConfig(cfg *Config) error {  // complexity: 8, line coverage: 0%
    // 8 validation checks...
}
```

- **CRAP** = 8^2 \* (1 - 0/100)^3 + 8 = 64 \* 1 + 8 = **72**
- Above threshold (15). Fix strategy: `add_tests` (complexity 8 < threshold 15, so tests alone will fix it).

### Example 4: The Q3 Problem

```go
func SaveUser(db *DB, user *User) error {  // complexity: 3, line coverage: 95%
    // Simple function, tests execute it but don't check the error or mutation
}
```

- **CRAP** = 3^2 \* (1 - 95/100)^3 + 3 = 9 \* 0.000125 + 3 = **3.0**
- **GazeCRAP** (contract coverage: 0%) = 3^2 * (1 - 0/100)^3 + 3 = 9 + 3 = **12**
- CRAP says safe. GazeCRAP says safe too (12 < 15). But if the GazeCRAP threshold were set to 10, this would be Q3 — tests run the code but don't verify the `ReceiverMutation` or `ErrorReturn`.

### Example 5: Q4 Dangerous

```go
func ReconcileState(ctx context.Context, state *State) error {
    // complexity: 18, line coverage: 20%, contract coverage: 0%
}
```

- **CRAP** = 18^2 \* (1 - 20/100)^3 + 18 = 324 \* 0.512 + 18 = 165.9 + 18 = **183.9**
- **GazeCRAP** = 18^2 * (1 - 0/100)^3 + 18 = 324 + 18 = **342**
- Both above threshold. Quadrant: Q4 Dangerous. Fix strategy: `decompose` (complexity 18 >= threshold 15).

## What's Next

- [Quality Assessment](quality.md) — how contract coverage is computed from assertion mapping
- [Classification](classification.md) — how effects are labeled contractual, ambiguous, or incidental
- [Side Effects](side-effects.md) — the 37 effect types that feed into scoring
