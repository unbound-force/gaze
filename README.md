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

Requires Go 1.24 or later.

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
```

**Detected side effect types:**

| Tier | Effects |
|------|---------|
| P0 | `ReturnValue`, `ErrorReturn`, `SentinelError`, `ReceiverMutation`, `PointerArgMutation` |
| P1 | `SliceMutation`, `MapMutation`, `GlobalMutation`, `WriterOutput`, `HTTPResponseWrite`, `ChannelSend`, `ChannelClose`, `DeferredReturnMutation` |
| P2 | `FileSystemWrite`, `FileSystemDelete`, `FileSystemMeta`, `DatabaseWrite`, `DatabaseTransaction`, `GoroutineSpawn`, `Panic`, `CallbackInvocation`, `LogWrite`, `ContextCancellation` |

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

# Custom threshold
gaze crap --crap-threshold=20 ./...

# CI mode: fail if too many crappy functions
gaze crap --max-crapload=5 ./...

# JSON output
gaze crap --format=json ./...
```

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

### CI Integration

Use threshold flags for CI enforcement. Gaze exits non-zero when limits are exceeded and prints a one-line summary to stderr:

```bash
gaze crap --max-crapload=5 --max-gaze-crapload=3 ./...
# stderr: CRAPload: 2/5 (PASS) | GazeCRAPload: 1/3 (PASS)
```

Without threshold flags, Gaze always exits 0 (report-only mode).

## Output Formats

Both commands support `--format=text` (default) and `--format=json`.

JSON output conforms to a documented schema. The analysis report schema is embedded in the binary and can be referenced at `internal/report/schema.go`.

## Architecture

```
cmd/gaze/           CLI entry point (cobra)
internal/
  analysis/         Side effect detection engine
    returns.go      Return value analysis (AST)
    sentinel.go     Sentinel error detection (AST)
    mutation.go     Receiver/pointer mutation (SSA)
    p1effects.go    P1-tier effects (AST)
    p2effects.go    P2-tier effects (AST)
  taxonomy/         Side effect type system and stable IDs
  loader/           Go package loading wrapper
  report/           JSON and text formatters for analysis output
  crap/             CRAP score computation and reporting
```

## Known Limitations

- **Direct function body only.** Gaze analyzes the immediate function body. Transitive side effects (effects produced by called functions) are out of scope for v1.
- **P3-P4 side effects not yet detected.** The taxonomy defines types for stdout/stderr writes, environment mutations, mutex operations, reflection, unsafe, and other P3-P4 effects, but detection logic is not yet implemented for these tiers.
- **Contract coverage not yet available.** GazeCRAP currently uses line coverage as a fallback. Full contract-aware coverage requires the contractual classification engine (Spec 002) and test quality metrics (Spec 003), which are planned but not yet built.
- **No CGo or unsafe analysis.** Functions using `cgo` or `unsafe.Pointer` are not analyzed for their specific side effects.
- **Single package loading.** The `analyze` command processes one package at a time. Use shell loops or scripting for multi-package analysis.

## License

See [LICENSE](LICENSE) for details.
