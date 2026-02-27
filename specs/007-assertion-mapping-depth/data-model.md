# Data Model: Assertion Mapping Depth

**Date**: 2026-02-27
**Feature**: [spec.md](spec.md) | [plan.md](plan.md)

## Entities

### Modified Entity: AssertionMapping

**Location**: `internal/taxonomy/types.go`
**Change**: No structural change. The `Confidence` field already
exists. Values are:

| Confidence Value | Meaning | When Used |
|-----------------|---------|-----------|
| 75 | Direct identity match | `types.Object` of assertion ident is directly in `objToEffectID` |
| 65 | Indirect match (NEW) | Root identifier of a selector/index/builtin expression matches |
| 0 | Unmapped | No match found; `UnmappedReason` populated |

No new fields, types, or structs are introduced.

### New Internal Function: resolveExprRoot

**Location**: `internal/quality/mapping.go` (unexported)

**Signature**:
```
resolveExprRoot(expr ast.Expr, info *types.Info) *ast.Ident
```

**Input**: An AST expression node and the package's type information.
**Output**: The root `*ast.Ident` reached by recursively unwinding
expression wrappers, or `nil` if the expression cannot be unwound.

**Resolution rules**:

| AST Node Type | Action | Example |
|---------------|--------|---------|
| `*ast.Ident` | Return directly (base case) | `got` |
| `*ast.SelectorExpr` | Recurse on `.X` | `result.Field` -> `result` |
| `*ast.IndexExpr` | Recurse on `.X` | `results[0]` -> `results` |
| `*ast.CallExpr` | If `Fun` is a `*types.Builtin` with name `len` or `cap` and exactly 1 argument, recurse on `Args[0]` | `len(docs)` -> `docs` |
| All other types | Return `nil` | — |

**Invariants**:
- Never returns the `Sel` field of a `SelectorExpr` — always resolves
  to the root identifier before the first selector
- Recursion depth bounded by Go source expression nesting (typically <= 5)
- Returns `nil` for any expression that cannot be unwound to an identifier

### Modified Function: matchAssertionToEffect

**Location**: `internal/quality/mapping.go`
**Change**: Two-pass matching strategy.

**Pass 1 (direct)**: Existing behavior unchanged. Walk expression
tree with `ast.Inspect` looking for `*ast.Ident` nodes whose
`types.Object` is directly in `objToEffectID`. Match confidence: 75.

**Pass 2 (indirect)**: If Pass 1 found no match, walk expression
tree again. For each node that is a `SelectorExpr`, `IndexExpr`, or
`CallExpr`, call `resolveExprRoot` to find the root identifier. If
the root's `types.Object` is in `objToEffectID`, produce a match at
confidence 65.

**State transitions**: None. `AssertionMapping` is stateless — each
mapping is computed independently and immutably.

### Modified Function: traceReturnValues (helper fallback)

**Location**: `internal/quality/mapping.go`
**Change**: When `findAssignLHS` fails (returns nil) for the target
call position, fall back to helper return tracing:

1. Search the test function's AST for all `AssignStmt` nodes
2. For each assignment whose RHS is a `CallExpr` to a function that
   (via SSA call graph at depth 1) invokes the target function:
   - Map the LHS variables to the corresponding return effects
   - Use the same positional index mapping as direct tracing
3. The traced objects receive indirect confidence (65) in
   `matchAssertionToEffect` because the match goes through a helper

**Constraints**:
- Only depth-1 helpers (direct callers of the target)
- SSA call graph verification required before tracing
- Fallback only — never activates when direct tracing succeeds

## Data Flow

```
traceReturnValues      builds    objToEffectID: map[types.Object]string
                                  key = variable's types.Object
                                  val = side effect ID
                       path A:   findAssignLHS succeeds (direct call)
                                  -> map LHS to effects by position
                       path B:   findAssignLHS fails (helper indirection)
                                  -> search test AST for helper calls
                                  -> verify helper calls target (SSA)
                                  -> map helper assignment LHS to effects

matchAssertionToEffect reads     objToEffectID
                       input     site.Expr (AST expression)
                       output    *AssertionMapping (confidence 75 or 65)
                                 or nil (unmapped)

  Pass 1: ast.Inspect -> *ast.Ident -> obj = info.Uses[ident]
           if obj in objToEffectID -> match at 75

  Pass 2: ast.Inspect -> SelectorExpr/IndexExpr/CallExpr
           -> resolveExprRoot -> root *ast.Ident
           -> obj = info.Uses[root]
           if obj in objToEffectID -> match at 65
```

## Relationships

```
AssertionSite --[mapped by]--> matchAssertionToEffect --[produces]--> AssertionMapping
                                    |
                                    +--> resolveExprRoot (indirect resolution)
                                    |
                                    +--> objToEffectID (lookup)
                                            |
                                            +--> traceReturnValues (builds map)
                                            +--> traceMutations (builds map)
```

## Validation Rules

- `resolveExprRoot` MUST return `nil` for expressions that cannot
  be unwound — never return a fabricated or incorrect identifier
- Only `types.Builtin` objects with name `len` or `cap` trigger
  call expression unwinding — user-defined functions are NOT unwound
- Pass 1 (direct) MUST execute before Pass 2 (indirect) — direct
  matches are always preferred
- If both Pass 1 and Pass 2 could match the same assertion to
  different effects, Pass 1's match wins (this cannot happen in
  practice since Pass 2 only runs when Pass 1 finds nothing)
