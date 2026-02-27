# Research: Assertion Mapping Depth

**Date**: 2026-02-27
**Feature**: [spec.md](spec.md) | [plan.md](plan.md)

## Research Questions

### R1: Why does ast.Inspect already visit child identifiers but fail to match?

**Question**: The `ast.Inspect` walk in `matchAssertionToEffect`
already recurses into child nodes. For `result.SideEffects`
(`SelectorExpr`), the walk visits both `Sel` (the field name
`SideEffects`) and `X` (the identifier `result`). Why doesn't the
existing walk match `result` against the `objToEffectID` map?

**Finding**: It actually does in some cases. The existing walk visits
`result` as a child ident under `SelectorExpr.X`. If `result` is in
`objToEffectID` (which it is, via `traceReturnValues`), the match
succeeds. However, the walk visits nodes in **preorder** — for a
`SelectorExpr`, `ast.Inspect` visits `SelectorExpr` first, then `X`,
then `Sel`. The `Sel` field name is also an `*ast.Ident`, and
`info.Uses[Sel]` resolves to the **struct field declaration's
`types.Object`** (e.g., `taxonomy.AnalysisResult.SideEffects`), which
is NOT in the `objToEffectID` map.

The critical issue is: `X` (`result`) IS matched when visited, BUT
only if `ast.Inspect` actually reaches it. In practice, the assertion
expression node passed to `matchAssertionToEffect` via `site.Expr` is
the **comparison expression** (e.g., `count != 1`), not the
selector expression itself. The `result.SideEffects` appears as a
deeper sub-expression nested inside a function call argument
(`countEffects(result.SideEffects, ...)`), which is itself inside a
binary comparison. The walk does visit all these nodes, but:

1. The `countEffects` function identifier resolves to a
   `types.Object` that is NOT in the map — walk continues.
2. The `result.SideEffects` selector's `Sel` resolves to the struct
   field — NOT in the map — walk continues.
3. The `result` identifier under `SelectorExpr.X` resolves to the
   local variable — IS in the map — **should match**.

After further investigation, the walk **does** match for simple
`result.Field` patterns in some test fixtures. The actual gap is
in the `analysis` package tests where the pattern is:

```go
result := analyzeFunc(t, "returns", "SingleReturn")
```

Here `analyzeFunc` is a helper function. The SSA target inference
identifies `analysis.AnalyzeFunctionWithSSA` as the target (called
inside the helper). But `traceReturnValues` looks for the AST
assignment containing the **target call** (`AnalyzeFunctionWithSSA`),
which is inside the helper — NOT in the test function's AST. So the
assignment `result := analyzeFunc(...)` is never found by
`findAssignLHS`, and `objToEffectID` is **empty**. This causes
`classifyUnmappedReason` to classify as `inline_call` or
`no_effect_match`.

**Decision**: The primary gap for the `analysis` package is
**helper indirection in return value tracing**, not selector
expression matching. However, selector and index matching are still
the fix needed for the `multilib`, `crap`, `classify`, `report`,
`docscan`, and `config` packages, plus any `analysis` tests that
call target functions directly.

**Revised understanding**: Two distinct problems contribute to low
contract coverage:

1. **Helper indirection** (analysis package): `result := helper(t, ...)`
   where the helper calls the target. `traceReturnValues` cannot trace
   through the helper. This is the bigger problem for the `analysis`
   package specifically. **SCOPE UPDATED**: Originally out of scope
   (tracked as Issue #6), helper return tracing was added to this spec
   as US5 after R3 revealed that the 80% SC-001 target is not
   achievable without it. See R3 decision (below) for details.

2. **Expression wrapping** (multilib, crap, classify, etc.): `user.Name`,
   `len(docs)`, `results[0].Field` — the traced variable IS in the
   `objToEffectID` map, but the assertion expression wraps it. This
   IS the problem this spec solves.

### R2: What is the actual mapping accuracy impact of expression resolution?

**Question**: If selector/index/builtin resolution is implemented,
what percentage of the 42 fixture assertions would newly match?

**Finding**: The test fixtures have these unmapped assertions:

- `welltested`: `c.Value()` — inline call, NOT fixed by expression
  resolution (no assignment to trace)
- `helpers`: 6 assertions inside helpers — depth > 0, classified as
  `helper_param`, NOT fixed by expression resolution
- `multilib`: `user.Name`, `user.Email`, `user.Age` — selector
  expressions on traced `user` variable. These WOULD be fixed.
  ~3-5 new mappings.
- `tabledriven`, `overspecd`, `undertested`: All current mappings are
  already direct identity matches; no selector patterns.

Estimated improvement: 3-5 additional mappings out of 42 total
assertion sites. The current 31/42 (73.8%) would improve to
~34-36/42 (81-86%). This meets the FR-011 target of >= 85% with
the upper bound.

### R3: What is the contract coverage impact across real packages?

**Question**: Which real packages will see the biggest contract
coverage improvement from expression resolution?

**Finding**: Based on the gaze quality analysis performed earlier:

| Package | Current CC | Unmapped Pattern | Expected After |
|---------|-----------|------------------|----------------|
| analysis | 6.9% | Helper indirection (NOT fixed by this spec) | ~6.9% (unchanged) |
| crap | 68.8% | Struct field assertions on Report | ~80-85% |
| classify | 66.7% | Nested slice/struct access | ~80-85% |
| report | 61.1% | Error and buffer assertions | ~70-75% |
| config | 81.2% | Minor field access | ~85-90% |
| docscan | 58.3% | len() and index patterns | ~75-80% |

**Revised weighted average estimate**: Since the `analysis` package
(108 pairs, 51% weight) will NOT improve from expression resolution
alone (its tests use helper indirection), the weighted average
improvement is more modest than initially projected:

- Before: 36.0%
- After expression resolution only: ~45-55% (analysis stays at 6.9%)
- To reach 80%: Would also require helper return tracing (Issue #6)

**Decision**: The 80% SC-001 target may not be achievable with
expression resolution alone. The spec should be updated if this
finding changes the acceptance criteria. However, the mapping
accuracy target (SC-002, >= 85%) IS achievable from fixture
improvements. The contract coverage target may need to be re-scoped
to "significant improvement" or may require a companion fix for
helper return tracing.

**Decision**: Proceed with expression resolution (high value, low
risk) AND add helper return value tracing to scope. The latter
requires `traceReturnValues` to also handle cases where the
assignment is `result := helper(t, ...)` and the helper's return
value flows from the target call. This is a natural extension: when
`findAssignLHS` fails for the target call position, search the test
function's AST for assignments whose RHS is a call to a function that
(at depth 1) calls the target. This was confirmed as in-scope via
user decision during the planning phase.

### R4: Built-in function detection via types.Builtin

**Question**: How to reliably detect built-in functions (len, cap)
vs user-defined functions with the same name?

**Finding**: The `go/types` package provides `types.Builtin` as the
object type for predeclared built-in functions. Using
`testPkg.TypesInfo.Uses[ident]` on the `Fun` identifier of a
`CallExpr` returns a `*types.Builtin` if the function is a genuine
built-in. User-defined functions named `len` would resolve to a
`*types.Func` or `*types.Var`, not `*types.Builtin`.

```go
obj := info.Uses[funIdent]
if builtin, ok := obj.(*types.Builtin); ok {
    switch builtin.Name() {
    case "len", "cap":
        // safe to unwind
    }
}
```

**Decision**: Use `*types.Builtin` type assertion for detection. The
allowlist is `{"len", "cap"}`. This is reliable and handles shadowed
names correctly.

### R5: go/ast expression node structures

**Question**: What are the exact Go AST node types and their field
relationships for the expression patterns we need to handle?

**Finding**: From `go/ast` package:

```go
type SelectorExpr struct {
    X   Expr   // expression before "."
    Sel *Ident // field identifier after "."
}

type IndexExpr struct {
    X     Expr // collection expression
    Index Expr // index expression
}

type CallExpr struct {
    Fun  Expr   // function expression
    Args []Expr // function arguments
}
```

The recursive unwinding algorithm:

```
resolveExprRoot(expr) -> *ast.Ident:
  SelectorExpr  -> resolveExprRoot(expr.X)
  IndexExpr     -> resolveExprRoot(expr.X)
  CallExpr      -> if isBuiltinInspector(expr.Fun) && len(expr.Args)==1:
                     resolveExprRoot(expr.Args[0])
                   else: nil
  Ident         -> return expr (base case)
  *             -> nil
```

**Decision**: Straightforward recursive descent. No edge cases beyond
the allowlist for built-ins.

## Summary of Decisions

| # | Decision | Rationale | Alternatives Rejected |
|---|----------|-----------|----------------------|
| 1 | Two-pass matching (direct then indirect) | Preserves existing behavior; clear confidence separation | Single-pass with priority tracking (more complex) |
| 2 | Confidence 65 for indirect matches | Clear separation from direct (75); above uncertainty threshold (50) | 50 (too low), 70 (too close to direct) |
| 3 | Built-in allowlist: len, cap only | Value-inspecting; side-effecting builtins would create false mappings | Full built-in set (semantically wrong for append, delete, etc.) |
| 4 | Detect builtins via types.Builtin | Handles shadowed names; type-safe | String comparison on name (fragile) |
| 5 | Recursive resolveExprRoot | Handles arbitrary nesting; bounded by Go expression depth | Iterative loop (less readable for same correctness) |
| 6 | Helper return tracing added to scope | Analysis package uses helper indirection, not expression wrapping; 80% target requires both fixes | Defer to separate spec (risks missing 80% target) |
