# Quality Assessment

Quality assessment answers the question: *do your tests actually verify the things that matter?* Gaze goes beyond line coverage to measure whether test assertions target a function's [contractual](../reference/glossary.md#contractual) [side effects](side-effects.md) — the observable behaviors that callers depend on.

The quality pipeline produces two metrics per test-target pair:

- **[Contract Coverage](../reference/glossary.md#contract-coverage)**: the ratio of contractual side effects that are asserted on
- **[Over-Specification](../reference/glossary.md#over-specification-score)**: the ratio of assertions that target incidental (implementation detail) side effects

## How It Works: The Four Steps

### Step 1: Test-Target Pairing

Gaze needs to know which function each test exercises. It infers this automatically using SSA (Static Single Assignment) call graph analysis.

For each test function (any function matching `Test*(*testing.T)`), Gaze:

1. Builds the SSA representation of the test package
2. Walks the SSA call graph from the test function, following calls up to a configurable depth (default: 3)
3. Identifies calls to functions in the package under test (excluding test functions, init functions, and closures)
4. Follows `MakeClosure` instructions to find target calls inside `t.Run` sub-tests — critical for table-driven test patterns

If multiple target functions are found, Gaze produces a quality report for each test-target pair and emits a warning. If no target is found, the test is skipped with a warning.

The `--target` flag can restrict analysis to a specific function name when automatic inference is ambiguous.

### Step 2: Assertion Detection

Gaze walks the test function's AST to find assertion patterns. It recognizes seven kinds of assertions:

| Kind | Pattern | Example |
|---|---|---|
| `stdlib_comparison` | `if got != want { t.Errorf(...) }` | Standard library comparison with test failure |
| `stdlib_error_check` | `if err != nil { t.Fatal(err) }` | Error nil check with test failure |
| `testify_equal` | `assert.Equal(t, got, want)` | Testify equality assertions (and variants: `NotEqual`, `Contains`, `Len`, `True`, `False`, etc.) |
| `testify_noerror` | `require.NoError(t, err)` | Testify error assertions (`NoError`, `Error`, `ErrorIs`, `ErrorAs`, etc.) |
| `testify_nil_check` | `assert.Nil(t, obj)` | Testify nil/not-nil assertions |
| `gocmp_diff` | `diff := cmp.Diff(want, got)` | go-cmp diff checks |
| `unknown` | Unrecognized pattern | Detected but not classified |

Assertion detection recurses into:

- **`t.Run` sub-tests**: Assertions inside sub-test closures are inlined at depth 0 (they're logically part of the parent test)
- **Helper functions**: Functions that accept `*testing.T` (or `*testing.TB`, `*testing.B`) are followed up to `MaxHelperDepth` levels (default: 3). Caller arguments are tracked so assertions inside helpers can be bridged back to the test's variables.

The **assertion detection confidence** metric reports what fraction of detected assertion sites were successfully pattern-matched (recognized as a known kind vs. `unknown`).

### Step 3: Assertion Mapping

This is the core of quality assessment: linking each detected assertion to the specific [side effect](side-effects.md) it verifies. Gaze uses a multi-pass strategy with decreasing confidence levels.

#### The Mapping Pipeline

For each assertion site, Gaze tries these passes in order, stopping at the first match:

#### Pass 1 — Direct Identity (confidence 75)

Walk the assertion expression's AST looking for identifiers whose `types.Object` matches a traced side effect variable. This handles the common case where a test assigns the target's return value and asserts on it directly:

```go
got, err := Divide(10, 2)
if got != 5 { t.Errorf(...) }  // "got" maps to ReturnValue
```

The bridge between SSA and AST works by finding the AST assignment statement that contains the target call, then mapping each left-hand-side identifier to the corresponding return value side effect by position.

#### Pass 1b — Helper Bridge (confidence 70)

When an assertion is inside a helper function (depth > 0), the helper's parameter objects are bridged back to the caller's argument objects. For example:

```go
func assertEqual(t *testing.T, got, want int) {
    if got != want { t.Errorf(...) }
}
// In test:
assertEqual(t, result, 42)  // "result" bridges to helper's "got"
```

#### Pass 2 — Indirect Root Resolution (confidence 65)

If Pass 1 finds no match, resolve composite expressions (selectors, index expressions, built-in calls) to their root identifier using `resolveExprRoot`. This handles patterns like:

```go
if result.Name != "expected" { ... }  // resolves result.Name → result
if len(results) != 3 { ... }          // resolves len(results) → results
```

#### Pass 3 — Inline Call (confidence 60)

Detects assertions where the target function is called inline without assigning its return value:

```go
if c.Value() != 5 { t.Errorf(...) }  // Value() is the target, called inline
```

#### Pass 4 — Container Unwrap (confidence 55)

Traces data flow forward from the return value through field access, type assertions, and transformation calls (like `json.Unmarshal`) to the assertion expression. This handles patterns where a test unpacks a complex return value:

```go
result := ProcessRequest(req)
body := result.Body
var data map[string]any
json.Unmarshal(body, &data)
if data["key"] != "value" { ... }  // traces back through unmarshal to result
```

#### Pass 5 — AI-Assisted Mapping (confidence 50)

When all mechanical passes fail, an optional AI mapper evaluates the semantic relationship between the assertion and the target function's side effects. AI mappings receive the lowest confidence (50) and are only used when configured.

#### Unmapped Assertions

Assertions that cannot be linked to any side effect are reported separately with a reason:

| Reason | Description |
|---|---|
| `helper_param` | Assertion is inside a helper function body and parameters couldn't be traced back |
| `inline_call` | Target function was called inline without assigning its return value |
| `no_effect_match` | No side effect object matched the assertion expression |

Unmapped assertions are excluded from both contract coverage and over-specification metrics.

### Step 4: Metric Computation

#### Contract Coverage

[Contract coverage](../reference/glossary.md#contract-coverage) is the ratio of contractual side effects that are asserted on:

```text
contract_coverage = covered_contractual / total_contractual * 100
```

Where:

- **total_contractual** = count of side effects classified as [contractual](../reference/glossary.md#contractual) (effects with no classification are conservatively treated as contractual)
- **covered_contractual** = count of contractual effects that have at least one assertion mapping

[Ambiguous](../reference/glossary.md#ambiguous) and [incidental](../reference/glossary.md#incidental) effects are excluded from both numerator and denominator.

**Gaps** are contractual effects with no assertion mapping. Each gap includes a hint — a Go code snippet suggesting how to write the missing assertion.

**Discarded returns** are a special case: when a test explicitly discards a return value (`_ = target()`), the corresponding return/error effects are flagged as definitively unasserted. Gaze detects this via SSA by checking which tuple indices have no `Extract` referrers.

#### Over-Specification

[Over-specification](../reference/glossary.md#over-specification-score) measures how many assertions target [incidental](../reference/glossary.md#incidental) side effects:

```text
over_specification_ratio = incidental_assertions / total_mapped_assertions
```

A high over-specification score means the test is fragile — it asserts on implementation details that may change without affecting the function's contract. Each incidental assertion includes a suggestion for how to remove the over-specification.

## Graceful Degradation

When SSA construction fails (e.g., due to upstream `x/tools` bugs with certain generic types), quality assessment degrades gracefully instead of returning an error:

- **Available (AST-only)**: Test function enumeration, assertion detection, assertion detection confidence
- **Zero-valued (requires SSA)**: Contract coverage, over-specification, assertion mapping, target inference

The `SSADegraded` flag in the package summary indicates when results are partial. The `SSADegradedPackages` field lists which packages failed SSA construction.

## What's Next

- [Classification](classification.md) — how effects are labeled contractual, ambiguous, or incidental
- [Scoring](scoring.md) — how contract coverage feeds into GazeCRAP scores
- [Analysis Pipeline](analysis-pipeline.md) — how side effects are detected in the first place
