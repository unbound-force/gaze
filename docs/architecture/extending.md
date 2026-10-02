# Extending Gaze

This guide explains how to extend Gaze with new capabilities. Each section covers an extension point: the files to modify, the patterns to follow, and the tests to add.

## Adding a New Side Effect Type

The side effect taxonomy is defined in `internal/taxonomy/types.go`. Adding a new type involves four steps.

### 1. Define the Constant

Add a new `SideEffectType` constant in the appropriate tier group in `internal/taxonomy/types.go`:

```go
// P2 — Important.
const (
    FileSystemWrite     SideEffectType = "FileSystemWrite"
    // ... existing P2 constants ...
    NewEffectType       SideEffectType = "NewEffectType"  // add here
)
```

Choose the tier based on the effect's importance:

| Tier | Criteria | Examples |
|------|----------|---------|
| P0 | Must detect — zero false negatives, zero false positives | Return values, error returns, receiver mutations |
| P1 | High value — important observable behavior | Channel sends, writer output, global mutations |
| P2 | Important — external interactions | Filesystem, database, goroutine spawns |
| P3 | Nice to have — environment and synchronization | Stdout/stderr, env vars, mutex operations |
| P4 | Exotic — rare patterns | Reflection, unsafe, CGo, finalizers |

### 2. Register the Tier

If you added a constant to an existing tier group, the `TierOf()` function (in `taxonomy/types.go`) already maps it. If you created a new tier or the mapping function uses a switch statement, add the new constant to the appropriate case.

### 3. Add Detection Logic

Detection logic lives in `internal/analysis/`. The file you modify depends on the tier:

| Tier | File | Function | Analysis Method |
|------|------|----------|----------------|
| P0 (returns) | `returns.go` | `AnalyzeReturns` | AST — inspects function signature result list |
| P0 (mutations) | `mutation.go` | `AnalyzeMutations` | SSA — tracks `*FieldAddr`/`*IndexAddr` → `Store` flows |
| P0 (sentinels) | `sentinel.go` | `AnalyzeSentinels` | AST — finds package-level `var Err* = errors.New(...)` |
| P1 | `p1effects.go` | `AnalyzeP1Effects` | AST — dispatches to per-node-type handlers |
| P2 | `p2effects.go` | `AnalyzeP2Effects` | AST — uses selector-to-effect mapping tables |

**For AST-based detection** (P1/P2 pattern): Most P1 and P2 effects are detected by matching function call selectors against a lookup table. For example, `p2effects.go` uses `p2SelectorEffects`:

```go
var p2SelectorEffects = map[string]map[string]taxonomy.SideEffectType{
    "os": {
        "WriteFile": taxonomy.FileSystemWrite,
        "Create":    taxonomy.FileSystemWrite,
        // Add new mappings here
    },
}
```

To add a new effect detected via function calls, add entries to the appropriate selector map or create a new one.

**For SSA-based detection** (mutations): If the new effect requires data flow analysis (e.g., tracking how a value flows through assignments and stores), add detection logic in `mutation.go` following the existing `AnalyzeMutations` pattern.

**For new detection patterns**: If neither AST selector matching nor SSA data flow fits, create a new file (e.g., `p3effects.go`) with an `AnalyzeP3Effects` function, then wire it into `analyzeFunction` in `analyzer.go`:

```go
func analyzeFunction(fset *token.FileSet, pkg *packages.Package, ssaPkg *ssa.Package, fd *ast.FuncDecl) taxonomy.AnalysisResult {
    // ... existing detection steps ...

    // 5. P3-tier effects (your new detector).
    p3Effects := AnalyzeP3Effects(fset, pkg.TypesInfo, fd, pkgPath, funcName)
    effects = append(effects, p3Effects...)

    // ...
}
```

### 4. Add Tests

- Create test fixtures in `internal/analysis/testdata/src/` — real Go packages that exhibit the new effect
- Add test cases in the corresponding `*_test.go` file (e.g., `p2effects_test.go`)
- Test both positive detection (effect is present) and negative detection (effect is absent)
- If the effect type appears in JSON output, update the JSON Schema in `internal/report/schema.go`

### Files to Modify (Summary)

| File | Change |
|------|--------|
| `internal/taxonomy/types.go` | Add `SideEffectType` constant |
| `internal/analysis/<tier>effects.go` | Add detection logic |
| `internal/analysis/analyzer.go` | Wire new detector into `analyzeFunction` (if new file) |
| `internal/analysis/<tier>effects_test.go` | Add tests |
| `internal/analysis/testdata/src/` | Add test fixture packages |
| `internal/report/schema.go` | Update JSON Schema if needed |

---

## Adding a New Classification Signal

Classification signals determine whether a side effect is contractual, incidental, or ambiguous. Each signal analyzer is a function that returns a `taxonomy.Signal` with a source name and weight.

### 1. Create the Analyzer

Create a new file in `internal/classify/` (e.g., `doccomment.go`):

```go
package classify

import (
    "github.com/unbound-force/gaze/v2/internal/taxonomy"
)

// maxDocCommentWeight is the maximum weight for doc comment signals.
const maxDocCommentWeight = 15

// AnalyzeDocCommentSignal checks for documentation patterns that
// indicate contractual intent. Returns a zero signal when no
// evidence is found.
func AnalyzeDocCommentSignal(/* params */) taxonomy.Signal {
    // Analyze and return a Signal with:
    //   Source:    "doc_comment"
    //   Weight:    positive (contractual evidence) or negative (incidental evidence)
    //   Reasoning: human-readable explanation
    return taxonomy.Signal{}
}
```

Follow the existing patterns:

| Existing Analyzer | File | Signal Source | Max Weight |
|-------------------|------|-------------|------------|
| Interface satisfaction | `interface.go` | `"interface"` | 30 |
| API surface visibility | `visibility.go` | `"visibility"` | 20 |
| Caller dependency | `callers.go` | `"caller"` | 15 |
| Naming convention | `naming.go` | `"naming"` | 10 |
| GoDoc comment | `godoc.go` | `"godoc"` | 15 |

### 2. Register in Signal Accumulation

Wire the new analyzer into `classifySideEffect` in `internal/classify/classify.go`:

```go
func classifySideEffect(
    funcName string,
    funcDecl *ast.FuncDecl,
    funcObj types.Object,
    receiverType types.Type,
    effectType taxonomy.SideEffectType,
    namingName string,
    ifaces []namedInterface,
    opts Options,
) []taxonomy.Signal {
    var signals []taxonomy.Signal

    // ... existing 5 analyzers ...

    // 6. Doc comment analysis (your new signal).
    if s := AnalyzeDocCommentSignal(/* params */); s.Source != "" {
        signals = append(signals, s)
    }

    return signals
}
```

The signal is automatically included in `ComputeScore` (in `score.go`), which sums all signal weights starting from `baseConfidence` (50) plus the tier boost. No changes to the scoring logic are needed.

### 3. Add Tests

- Create `internal/classify/doccomment_test.go` with test cases
- Test positive signals (evidence found, positive weight)
- Test negative signals (counter-evidence found, negative weight)
- Test zero signals (no evidence, zero weight returned)
- Add test fixtures in `internal/classify/testdata/` if needed

### Files to Modify (Summary)

| File | Change |
|------|--------|
| `internal/classify/doccomment.go` | New analyzer function |
| `internal/classify/classify.go` | Wire into `classifySideEffect` |
| `internal/classify/doccomment_test.go` | Tests for the new analyzer |
| `internal/classify/testdata/` | Test fixtures (if needed) |

---

## Adding a New Output Format

Output formatters live with the command's existing report model. Analyze formatters live in `internal/report/`; CRAP formatters in `internal/crap/`; quality formatters in `internal/quality/`; and combined report formatters in `internal/aireport/`. Each family supports its existing text or JSON paths plus self-contained HTML where the command explicitly accepts it.

### 1. Create the Formatter

Create a new file in the package that owns the command's report model. For example, an analyze CSV formatter belongs in `internal/report/csv.go`:

```go
package report

import (
    "io"

    "github.com/unbound-force/gaze/v2/internal/taxonomy"
)

// WriteCSV writes analysis results as CSV to the writer.
func WriteCSV(w io.Writer, results []taxonomy.AnalysisResult) error {
    // Format and write results
    return nil
}
```

Follow the existing pattern:

- Accept `io.Writer` as the first parameter (enables testing with `bytes.Buffer`)
- Accept the command's existing typed report model as the data source (`[]taxonomy.AnalysisResult` in this example)
- Return `error`

### 2. Wire into the CLI

In `cmd/gaze/main.go`, pass command-specific formats to `cliutil.ValidateFormat` as additional allowed values, then add the format to the output switch. Commands that do not pass the additional value continue to reject it:

```go
if err := cliutil.ValidateFormat(p.format, "csv"); err != nil {
    return err
}
```

```go
case "csv":
    return report.WriteCSV(p.stdout, results)
```

### 3. Add Tests

- Create `internal/report/csv_test.go`
- Test output structure, edge cases (empty results, special characters)
- Verify output fits within 80-column terminals if applicable

### Files to Modify (Summary)

| File | Change |
|------|--------|
| `internal/report/csv.go` | New formatter function |
| `internal/report/csv_test.go` | Tests |
| `cmd/gaze/main.go` | Wire format into CLI flag handling |

---

## Adding a New AI Adapter

AI adapters format analysis payloads using external AI CLIs or APIs. The adapter interface is defined in `internal/aireport/adapter.go`.

### 1. Implement the Interface

The `AIAdapter` interface has a single method:

```go
type AIAdapter interface {
    Format(ctx context.Context, systemPrompt string, payload io.Reader) (string, error)
}
```

Create a new file (e.g., `adapter_mistral.go`):

```go
package aireport

import (
    "context"
    "io"
)

// MistralAdapter formats payloads using the Mistral CLI.
type MistralAdapter struct {
    config AdapterConfig
}

// Format implements AIAdapter.
func (a *MistralAdapter) Format(ctx context.Context, systemPrompt string, payload io.Reader) (string, error) {
    // For subprocess-based adapters, use the shared runSubprocess helper:
    args := []string{"--system-prompt", systemPrompt}
    if a.config.Model != "" {
        args = append(args, "--model", a.config.Model)
    }

    stdout, stderr, err := runSubprocess(ctx, "mistral", args, "", payload)
    if err != nil {
        return "", err
    }

    output := strings.TrimSpace(string(stdout))
    if output == "" {
        return "", fmt.Errorf("mistral returned empty output%s", formatStderrSuffix(stderr))
    }
    return output, nil
}
```

**Key helpers available:**

- `runSubprocess(ctx, binaryName, args, cmdDir, payload)` — shared subprocess execution with stdout/stderr capture, output size limiting, and error formatting. Used by Claude, Gemini, and OpenCode adapters.
- `formatStderrSuffix(stderrBytes)` — formats stderr for inclusion in error messages, with truncation at 512 bytes.

### 2. Implement AdapterValidator (Optional)

If the adapter uses a CLI binary, implement `AdapterValidator` for pre-flight binary checks:

```go
// ValidateBinary checks that the mistral CLI is on PATH.
func (a *MistralAdapter) ValidateBinary() error {
    _, err := exec.LookPath("mistral")
    if err != nil {
        return fmt.Errorf("mistral CLI not found on PATH: %w", err)
    }
    return nil
}
```

This is called before the analysis pipeline runs, providing a fast failure if the binary is missing.

### 3. Register in the Factory

In `internal/aireport/adapter.go`, add the adapter to the allowlist and factory:

```go
var validAdapters = map[string]bool{
    "claude":   true,
    "gemini":   true,
    "ollama":   true,
    "opencode": true,
    "mistral":  true,  // add here
}

func NewAdapter(cfg AdapterConfig) (AIAdapter, error) {
    // ... existing validation ...
    switch cfg.Name {
    // ... existing cases ...
    case "mistral":
        return &MistralAdapter{config: cfg}, nil
    }
}
```

### 4. Update CLI Help Text

In `cmd/gaze/main.go`, update the `--ai` flag description to include the new adapter name.

### 5. Add Tests

Create `internal/aireport/adapter_mistral_test.go`:

- Test `Format` with a fake binary (create a test fixture in `testdata/fake_mistral/main.go`)
- Test error cases: binary not found, non-zero exit, empty output
- Test `ValidateBinary` if implemented
- Follow the existing adapter test patterns (see `adapter_claude_test.go` for examples)

### Files to Modify (Summary)

| File | Change |
|------|--------|
| `internal/aireport/adapter_mistral.go` | New adapter implementation |
| `internal/aireport/adapter.go` | Add to `validAdapters` map and `NewAdapter` factory |
| `internal/aireport/adapter_mistral_test.go` | Tests |
| `internal/aireport/testdata/fake_mistral/main.go` | Fake binary for testing |
| `cmd/gaze/main.go` | Update `--ai` flag help text and error messages |

---

## General Extension Guidelines

1. **Follow the options struct pattern**: If your extension needs configuration, add fields to the relevant `Options` struct rather than adding function parameters.

2. **Keep `taxonomy` as a leaf**: The `taxonomy` package defines domain types only. Never add dependencies from `taxonomy` to other internal packages.

3. **Test with real Go packages**: For analysis and classification tests, create real Go source files in `testdata/src/` directories. Load them with `go/packages` in tests — this ensures detection works on actual compiled code, not synthetic AST nodes.

4. **Use the standard library for testing**: No testify, gomega, or other assertion libraries. Use `t.Errorf` and `t.Fatalf` directly.

5. **Wrap errors with context**: Every error return should include the operation context via `fmt.Errorf("operation: %w", err)`.

6. **Write a spec first**: Non-trivial extensions require a spec (see [Contributing Guide](contributing.md) for the spec-first workflow). This ensures the design is reviewed before implementation begins.
