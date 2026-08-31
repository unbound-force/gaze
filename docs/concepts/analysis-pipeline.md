# Analysis Pipeline

Gaze detects [side effects](side-effects.md) through a multi-phase static analysis pipeline that combines two complementary techniques: **AST (Abstract Syntax Tree) analysis** for syntactic patterns and **SSA (Static Single Assignment) analysis** for data flow tracking. Each technique has strengths the other lacks, and together they cover the full range of detectable effects.

## Pipeline Overview

```
┌──────────┐    ┌──────────────┐    ┌──────────────┐    ┌─────────┐
│  Load    │───>│ AST Analysis │───>│ SSA Analysis │───>│ Combine │──> AnalysisResult
│ Packages │    │              │    │              │    │ Results │
└──────────┘    └──────────────┘    └──────────────┘    └─────────┘
```

For each function in the loaded package, Gaze runs four analysis phases in sequence:

1. **Return value analysis** (AST) — detects `ReturnValue`, `ErrorReturn`, `SentinelError`, `DeferredReturnMutation`
2. **Mutation analysis** (SSA, with AST fallback) — detects `ReceiverMutation`, `PointerArgMutation`
3. **P1 effect analysis** (AST) — detects `SliceMutation`, `MapMutation`, `GlobalMutation`, `WriterOutput`, `HTTPResponseWrite`, `ChannelSend`, `ChannelClose`
4. **P2 effect analysis** (AST) — detects `FileSystemWrite`, `FileSystemDelete`, `FileSystemMeta`, `DatabaseWrite`, `DatabaseTransaction`, `GoroutineSpawn`, `Panic`, `CallbackInvocation`, `LogWrite`, `ContextCancellation`

The results from all four phases are combined into a single `AnalysisResult` per function, containing the function's identity (`FunctionTarget`) and its complete list of detected side effects.

## Phase 0: Package Loading

Before analysis begins, Gaze loads the target package using `go/packages` with full type information. The load mode includes:

- Package name, files, and compiled Go files
- Import graph and dependencies
- Type information (`go/types`)
- Syntax trees (`go/ast`)
- Type-checked information (`TypesInfo`)
- Type sizes

This produces a `*packages.Package` with everything needed for both AST and SSA analysis. The `loader.Load` function handles single-package loading, while `loader.LoadModule` loads all packages in a module (used for cross-package analysis like caller counting in [classification](classification.md)).

## Phase 1: Return Value Analysis (AST)

**File:** `internal/analysis/returns.go`

Return value analysis inspects the function signature's result list to detect:

### ReturnValue (P0)

Every non-error return position produces a `ReturnValue` effect. For `func Divide(a, b int) (float64, error)`, position 0 (`float64`) generates a `ReturnValue` effect.

### ErrorReturn (P0)

Every error-typed return position produces an `ErrorReturn` effect. The error type is detected using `go/types` information (with a fallback to AST name matching for the `error` identifier).

### SentinelError (P0)

Package-level sentinel error variables (`var ErrNotFound = errors.New("not found")`) are detected by scanning file-level declarations. Sentinels are attached to a synthetic `<package>` function target since they are package-level, not function-level.

### DeferredReturnMutation (P1)

When a function uses named returns and a `defer` statement modifies one of those named return variables, a `DeferredReturnMutation` effect is generated. This is a common Go pattern for error wrapping:

```go
func Process() (err error) {
    defer func() {
        if err != nil {
            err = fmt.Errorf("process: %w", err)  // modifies named return
        }
    }()
    // ...
}
```

## Phase 2: Mutation Analysis (SSA with AST Fallback)

**File:** `internal/analysis/mutation.go`

Mutation analysis detects when a function modifies state through its receiver or pointer parameters. This is the only phase that uses SSA, because tracking data flow through pointer indirection requires the precision of SSA's value graph.

### How SSA Detection Works

1. **Build SSA**: The SSA representation is built once per package using `ssautil.AllPackages` with `ssa.InstantiateGenerics` (for generic type support) and `ssa.BuildSerially` (to keep construction on the calling goroutine for panic recovery).

2. **Find the SSA function**: The target function is located in the SSA package by matching its `types.Func` object (for precise lookup) or by name-based fallback.

3. **Walk SSA instructions**: Every `*ssa.Store` instruction is examined:
   - **Receiver mutation**: If the store's address traces through `FieldAddr` instructions back to the receiver parameter, it's a `ReceiverMutation`. The top-level field name (closest to the receiver) is reported.
   - **Pointer argument mutation**: If the store's address traces back to a pointer-typed parameter (through `FieldAddr`, `IndexAddr`, or `UnOp` dereference), it's a `PointerArgMutation`.

4. **Trace to parameter**: The `tracesToParam` function walks up the SSA value chain (through `FieldAddr`, `IndexAddr`, `UnOp`, and `Phi` nodes) with cycle detection to determine if a value ultimately derives from a specific parameter.

### ReceiverMutation (P0)

Detected when a method's SSA body contains a `Store` instruction whose address traces through `FieldAddr` back to the receiver parameter. For nested field access like `s.Nested.Value = v`, the top-level field `Nested` is reported.

### PointerArgMutation (P0)

Detected when a function's SSA body contains a `Store` instruction whose address traces back to a pointer-typed parameter. Handles direct field stores, index stores, and dereference stores.

### AST Fallback

When SSA construction fails (returns nil), mutation analysis falls back to AST-based detection. The AST fallback covers the most common patterns:

- **Receiver mutations**: Assignment statements where the left-hand side's root identifier matches the receiver name (e.g., `s.field = value`, `s.count++`)
- **Pointer argument mutations**: Assignment statements where the left-hand side's root identifier matches a pointer parameter name

AST fallback effects include "(AST fallback)" in their description to distinguish them from SSA-detected mutations. The AST approach is lower fidelity — it can miss mutations through complex pointer chains and may produce false positives for shadowed variables — but it ensures Gaze always produces results even when SSA is unavailable.

## Phase 3: P1 Effect Analysis (AST)

**File:** `internal/analysis/p1effects.go`

P1 effects are detected by walking the function body's AST and dispatching to per-node-type handlers:

- **`AssignStmt`**: Detects `GlobalMutation` (assignment to package-level variables using type resolution), `MapMutation` (map index assignment), and `SliceMutation` (slice index assignment)
- **`IncDecStmt`**: Detects `GlobalMutation` via `++`/`--` on package-level variables
- **`SendStmt`**: Detects `ChannelSend` (`ch <- value`)
- **`CallExpr`**: Detects `ChannelClose` (builtin `close(ch)` verified via type resolution), `WriterOutput` (calls to `Write` on `io.Writer` types), and `HTTPResponseWrite` (calls to `http.ResponseWriter` methods)

Global variable detection uses `types.Info` to distinguish package-level variables from locals. A fast-path check against function signature names (parameters, named returns, receiver) avoids expensive type lookups for obvious locals.

## Phase 4: P2 Effect Analysis (AST)

**File:** `internal/analysis/p2effects.go`

P2 effects are detected through two AST node types:

- **`GoStmt`**: Detects `GoroutineSpawn` from `go` statements
- **`CallExpr`**: Detects multiple effect types:
  - `Panic` — builtin `panic()` verified via type resolution
  - `FileSystemWrite/Delete/Meta` — calls to `os.WriteFile`, `os.Remove`, `os.Chmod`, etc., resolved via a lookup table keyed by import path
  - `LogWrite` — calls to `log.Print*`, `log.Fatal*`, `slog.Debug/Info/Warn/Error`
  - `ContextCancellation` — calls to `context.WithCancel`, `WithTimeout`, `WithDeadline`
  - `DatabaseWrite` — `Exec`/`ExecContext` on `*sql.DB`/`*sql.Tx`/`*sql.Stmt`
  - `DatabaseTransaction` — `Begin`/`BeginTx` on `*sql.DB`
  - `CallbackInvocation` — calling a function-typed parameter

Import alias resolution uses `types.Info` to map AST identifiers to their actual import paths, preventing false positives from user packages with the same short name as standard library packages.

## Deduplication

Each analysis phase maintains a `seen` map to prevent reporting the same effect multiple times. For example, if a function assigns to the same global variable in three places, only one `GlobalMutation` effect is reported. The deduplication key varies by effect type (e.g., `"global:" + varName` for globals, `"chsend:" + channelName` for channel sends).

## Graceful Degradation

The pipeline is designed to produce useful results even when parts fail:

| Failure | Impact | Mitigation |
|---|---|---|
| SSA build panics | No mutation detection via SSA | `ssaguard.SafeSSABuild` recovers the panic; AST fallback detects common mutation patterns |
| SSA build returns nil | Same as panic | AST fallback activates automatically |
| Type info unavailable | Reduced precision for global detection, import resolution | Fallback to AST name matching (may produce false positives) |
| Package load errors | No analysis for that package | Error returned to caller; other packages unaffected |

The `ssa.BuildSerially` flag ensures SSA construction runs on the calling goroutine, making Go's goroutine-scoped `recover()` effective. Without this flag, `prog.Build()` spawns child goroutines whose panics cannot be caught.

## What's Next

- [Side Effects](side-effects.md) — the complete taxonomy of 37 effect types
- [Classification](classification.md) — how detected effects are classified as contractual, ambiguous, or incidental
- [Quality Assessment](quality.md) — how test assertions are mapped to detected effects
