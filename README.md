# Gaze

Test quality analysis via side effect detection for Go.

Gaze statically analyzes Go functions to detect all observable side effects and computes CRAP (Change Risk Anti-Patterns) scores by combining cyclomatic complexity with test coverage. It helps you find functions that are complex and under-tested -- the riskiest code to change.

## Installation

```bash
go install github.com/jflowers/gaze/cmd/gaze@latest
```

Or build from source:

```bash
git clone https://github.com/jflowers/gaze.git
cd gaze
go build -o gaze ./cmd/gaze
```

Requires Go 1.24.2 or later.

## Commands

### `gaze analyze` -- Side Effect Detection

Analyze a Go package to detect all observable side effects each function produces.

```bash
# Analyze all exported functions in a package
gaze analyze ./internal/analysis

# Analyze a specific function
gaze analyze -f ParseConfig ./internal/config

# Include unexported functions
gaze analyze --include-unexported ./internal/loader

# JSON output
gaze analyze --format=json ./internal/analysis

# Classify side effects as contractual, incidental, or ambiguous
gaze analyze --classify ./internal/analysis

# Verbose classification with full signal breakdown
gaze analyze --verbose ./internal/analysis

# Interactive TUI for browsing results
gaze analyze -i ./internal/analysis

# Use a config file with custom thresholds
gaze analyze --classify --config=.gaze.yaml --contractual-threshold=90 ./internal/analysis
```

**Flags:**

| Flag | Short | Description |
|------|-------|-------------|
| `--function` | `-f` | Analyze a specific function (default: all exported) |
| `--format` | | Output format: `text` or `json` (default: `text`) |
| `--include-unexported` | | Include unexported functions |
| `--interactive` | `-i` | Launch interactive TUI for browsing results |
| `--classify` | | Classify side effects as contractual, incidental, or ambiguous |
| `--verbose` | `-v` | Print full signal breakdown (implies `--classify`) |
| `--config` | | Path to `.gaze.yaml` config file (default: search CWD) |
| `--contractual-threshold` | | Override contractual confidence threshold (default: from config or 80) |
| `--incidental-threshold` | | Override incidental confidence threshold (default: from config or 50) |

**Detected side effect types:**

| Tier | Effects |
|------|---------|
| P0 | `ReturnValue`, `ErrorReturn`, `SentinelError`, `ReceiverMutation`, `PointerArgMutation` |
| P1 | `SliceMutation`, `MapMutation`, `GlobalMutation`, `WriterOutput`, `HTTPResponseWrite`, `ChannelSend`, `ChannelClose`, `DeferredReturnMutation` |
| P2 | `FileSystemWrite`, `FileSystemDelete`, `FileSystemMeta`, `DatabaseWrite`, `DatabaseTransaction`, `GoroutineSpawn`, `Panic`, `CallbackInvocation`, `LogWrite`, `ContextCancellation` |
| P3* | `StdoutWrite`, `StderrWrite`, `EnvVarMutation`, `MutexOp`, `WaitGroupOp`, `AtomicOp`, `TimeDependency`, `ProcessExit`, `RecoverBehavior` |
| P4* | `ReflectionMutation`, `UnsafeMutation`, `CgoCall`, `FinalizerRegistration`, `SyncPoolOp`, `ClosureCaptureMutation` |

*P3 and P4 types are defined in the taxonomy but detection is not yet implemented.*

Example output:

```
=== ParseConfig ===
    func ParseConfig(path string) (*Config, error)
    internal/config/config.go:15:1

    TIER  TYPE         DESCRIPTION
    ----  ----         -----------
    P0    ReturnValue  returns *Config at position 0
    P0    ErrorReturn  returns error at position 1

    Summary: P0: 2
```

### `gaze crap` -- CRAP Score Analysis

Compute CRAP scores by combining cyclomatic complexity with test coverage.

```bash
# Analyze all packages
gaze crap ./...

# Use an existing coverage profile
gaze crap --coverprofile=cover.out ./...

# Custom thresholds
gaze crap --crap-threshold=20 ./...
gaze crap --gaze-crap-threshold=20 ./...

# CI mode: fail if too many crappy functions
gaze crap --max-crapload=5 ./...

# JSON output
gaze crap --format=json ./...
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--format` | Output format: `text` or `json` (default: `text`) |
| `--coverprofile` | Path to existing coverage profile (default: generate one) |
| `--crap-threshold` | CRAP score threshold (default: 15) |
| `--gaze-crap-threshold` | GazeCRAP score threshold, used when contract coverage is available (default: 15) |
| `--max-crapload` | Fail if CRAPload exceeds this count (0 = no limit) |
| `--max-gaze-crapload` | Fail if GazeCRAPload exceeds this count (0 = no limit) |

**CRAP formula:**

```
CRAP(m) = complexity^2 * (1 - coverage/100)^3 + complexity
```

A function with complexity 5 and 0% coverage has CRAP = 30. The same function with 100% coverage has CRAP = 5. The default threshold is 15.

Example output:

```
CRAP    COMPLEXITY  COVERAGE  FUNCTION       FILE
----    ----------  --------  --------       ----
30.0 *  5           0.0%      ParseConfig    internal/config/config.go:15
5.0     5           100.0%    FormatOutput   internal/report/text.go:20

--- Summary ---
Functions analyzed:  2
Avg complexity:     5.0
Avg line coverage:  50.0%
Avg CRAP score:     17.5
CRAP threshold:     15
CRAPload:           1 (functions at or above threshold)
```

### `gaze quality` -- Test Quality Assessment

Assess how well a package's tests assert on the contractual side effects of the functions they test. Reports Contract Coverage (ratio of contractual effects that are asserted on) and Over-Specification Score (assertions on incidental implementation details).

```bash
# Analyze test quality for a package
gaze quality ./internal/analysis

# Target a specific function
gaze quality --target=LoadAndAnalyze ./internal/analysis

# Verbose output with detailed assertion and mapping information
gaze quality --verbose ./internal/analysis

# JSON output
gaze quality --format=json ./internal/analysis

# CI mode: enforce minimum contract coverage
gaze quality --min-contract-coverage=80 --max-over-specification=3 ./internal/analysis

# Custom classification thresholds
gaze quality --config=.gaze.yaml --contractual-threshold=90 ./internal/analysis
```

**Flags:**

| Flag | Short | Description |
|------|-------|-------------|
| `--format` | | Output format: `text` or `json` (default: `text`) |
| `--target` | | Restrict analysis to tests that exercise this function |
| `--verbose` | `-v` | Show detailed assertion and mapping information |
| `--config` | | Path to `.gaze.yaml` config file (default: search CWD) |
| `--contractual-threshold` | | Override contractual confidence threshold (default: from config or 80) |
| `--incidental-threshold` | | Override incidental confidence threshold (default: from config or 50) |
| `--min-contract-coverage` | | Fail if contract coverage is below this percentage (0 = no limit) |
| `--max-over-specification` | | Fail if over-specification count exceeds this (0 = no limit) |

### `gaze schema` -- JSON Schema Output

Print the JSON Schema (Draft 2020-12) that documents the structure of `gaze analyze --format=json` output. Useful for validating output or generating client types.

```bash
gaze schema
```

### `gaze docscan` -- Documentation Scanner

Scan the repository for Markdown documentation files and output a prioritized list as JSON. Useful as input to the `/classify-docs` OpenCode command for document-enhanced classification.

Files are prioritized by proximity to the target package:
1. Same directory as the target package (highest relevance)
2. Module root
3. Other locations

```bash
# Scan from current directory
gaze docscan

# Scan for a specific package
gaze docscan ./internal/analysis

# Use a config file
gaze docscan --config=.gaze.yaml ./internal/analysis
```

### `gaze self-check` -- Self-Analysis

Run CRAP analysis on Gaze's own source code, serving as both a dogfooding exercise and a code quality gate.

```bash
# Run self-check
gaze self-check

# JSON output
gaze self-check --format=json

# CI mode: enforce limits
gaze self-check --max-crapload=5 --max-gaze-crapload=3
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--format` | Output format: `text` or `json` (default: `text`) |
| `--max-crapload` | Fail if CRAPload exceeds this count (0 = no limit) |
| `--max-gaze-crapload` | Fail if GazeCRAPload exceeds this count (0 = no limit) |

### CI Integration

Use threshold flags for CI enforcement. Gaze exits non-zero when limits are exceeded and prints a one-line summary to stderr:

```bash
gaze crap --max-crapload=5 --max-gaze-crapload=3 ./...
# stderr: CRAPload: 2/5 (PASS) | GazeCRAPload: 1/3 (PASS)
```

Without threshold flags, Gaze always exits 0 (report-only mode).

## Output Formats

The `analyze`, `crap`, `quality`, and `self-check` commands support `--format=text` (default) and `--format=json`.

JSON output conforms to documented schemas. Use `gaze schema` to print the analysis report schema. The schemas are embedded in the binary at `internal/report/schema.go`.

## Architecture

```
cmd/gaze/              CLI entry point (cobra)
internal/
  analysis/            Side effect detection engine
    analyzer.go        Main analysis orchestrator
    returns.go         Return value analysis (AST)
    sentinel.go        Sentinel error detection (AST)
    mutation.go        Receiver/pointer mutation (SSA)
    p1effects.go       P1-tier effects (AST)
    p2effects.go       P2-tier effects (AST)
  taxonomy/            Side effect type system and stable IDs
  classify/            Contractual classification engine
  config/              Configuration file handling (.gaze.yaml)
  loader/              Go package loading wrapper
  report/              JSON and text formatters for analysis output
  crap/                CRAP score computation and reporting
  quality/             Test quality assessment (contract coverage)
  docscan/             Documentation file scanner
```

## Known Limitations

- **Direct function body only.** Gaze analyzes the immediate function body. Transitive side effects (effects produced by called functions) are out of scope for v1.
- **P3-P4 side effects not yet detected.** The taxonomy defines types for stdout/stderr writes, environment mutations, mutex operations, reflection, unsafe, and other P3-P4 effects, but detection logic is not yet implemented for these tiers.
- **GazeCRAP accuracy is limited.** The quality pipeline is wired into the CRAP command and GazeCRAP scores are computed when contract coverage data is available. However, assertion-to-side-effect mapping accuracy is currently ~74% (target: 90%), primarily affecting helper function assertions and testify field-access patterns (tracked as GitHub Issue #6).
- **No CGo or unsafe analysis.** Functions using `cgo` or `unsafe.Pointer` are not analyzed for their specific side effects.
- **Single package loading.** The `analyze` command processes one package at a time. Use shell loops or scripting for multi-package analysis.

## License

Apache License 2.0. See [LICENSE](LICENSE) for details.
