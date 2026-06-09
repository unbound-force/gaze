# Gaze

**Test quality analysis via side effect detection for Go.**

Line coverage tells you which lines ran. It does not tell you whether your tests actually verified anything.

A function can have 90% line coverage and tests that assert on nothing contractually meaningful — logging calls, goroutine lifecycle, internal stdout writes — while leaving the return values, error paths, and state mutations completely unverified. That function is dangerous to change, and traditional coverage metrics will not warn you.

Gaze fixes this by working from first principles:

1. **Detect** every observable side effect a function produces (return values, error returns, mutations, I/O, channel sends, etc.).
2. **Classify** each effect as *contractual* (part of the function's public obligation), *incidental* (an implementation detail), or *ambiguous*.
3. **Measure** whether your tests actually assert on the contractual effects — and flag the ones they don't.

This produces three actionable metrics: **Contract Coverage** (percentage of contractual effects asserted on), **Over-Specification Score** (assertions on implementation details), and **GazeCRAP** (a risk score combining complexity with contract coverage). For details on each metric, see the [Scoring](docs/concepts/scoring.md) and [Quality Assessment](docs/concepts/quality.md) concept docs.

Gaze requires no annotations, no test framework changes, and no restructuring of your code. It analyzes your existing Go packages as-is.

## Documentation

Full documentation is available in [`docs/`](docs/index.md):

- **[Getting Started](docs/getting-started/quickstart.md)** — Install and produce meaningful output in under 10 minutes
- **[Concepts](docs/concepts/side-effects.md)** — Side effects, classification, scoring, quality metrics
- **[CLI Reference](docs/reference/cli/analyze.md)** — Flags, defaults, and output formats for every command
- **[Guides](docs/guides/ci-integration.md)** — CI integration, AI reports, score improvement strategies
- **[Architecture](docs/architecture/overview.md)** — Package structure, data flow, contributing guide
- **[Porting](docs/porting/contracts.md)** — Language-agnostic contracts for building "Gaze for Python/Rust/etc."

## Installation

### Homebrew (recommended)

```bash
brew install unbound-force/tap/gaze
```

### Go Install

```bash
go install github.com/unbound-force/gaze/cmd/gaze@latest
```

### Build from Source

```bash
git clone https://github.com/unbound-force/gaze.git
cd gaze
go build -o gaze ./cmd/gaze
```

Requires Go 1.25.0 or later. For platform notes and verification steps, see [Installation](docs/getting-started/installation.md).

### macOS Code Signing

Homebrew binaries are code-signed with an Apple Developer ID certificate and notarized by Apple's notary service. macOS Gatekeeper trusts the binary on first run -- no security overrides needed.

**For maintainers**: Signing requires 5 GitHub secrets (Apple Developer ID certificate + App Store Connect API key). See [quickstart guide](specs/014-macos-notarization/quickstart.md) for setup instructions. When secrets are not configured, the release pipeline produces unsigned binaries without error.

## Commands

### `gaze analyze` -- Side Effect Detection

Detect all observable side effects each function produces. Gaze detects [37 effect types across 5 tiers](docs/concepts/side-effects.md) (P0–P4).

```bash
gaze analyze ./internal/analysis                    # All exported functions
gaze analyze -f ParseConfig ./internal/config       # Specific function
gaze analyze --classify ./internal/analysis         # With classification labels
gaze analyze --format=json ./internal/analysis      # JSON output
```

For all flags and options, see [`gaze analyze` reference](docs/reference/cli/analyze.md).

### `gaze crap` -- CRAP Score Analysis

Compute CRAP scores by combining cyclomatic complexity with test coverage.

```bash
gaze crap ./...                                     # Analyze all packages
gaze crap --coverprofile=cover.out ./...            # Use existing coverage
gaze crap --max-crapload=5 ./...                    # CI mode: fail on threshold
```

For the CRAP formula, GazeCRAP, quadrants, and fix strategies, see [Scoring](docs/concepts/scoring.md). For all flags, see [`gaze crap` reference](docs/reference/cli/crap.md).

### `gaze quality` -- Test Quality Assessment

Assess how well a package's tests assert on contractual side effects.

```bash
gaze quality ./internal/analysis                    # Analyze test quality
gaze quality --target=LoadAndAnalyze ./internal/analysis  # Specific function
gaze quality --verbose ./internal/analysis          # Detailed mapping info
```

For all flags, see [`gaze quality` reference](docs/reference/cli/quality.md).

### `gaze report` -- AI-Powered Quality Report

Orchestrate all analysis operations and pipe the results to an AI model for formatting.

```bash
gaze report ./... --ai=claude                       # Claude adapter
gaze report ./... --ai=opencode                     # OpenCode adapter
gaze report ./... --format=json                     # JSON only (no AI needed)
gaze report ./... --ai=claude --coverprofile=coverage.out  # Reuse coverage
```

For adapter setup, CI integration, and all flags, see [`gaze report` reference](docs/reference/cli/report.md) and [AI Reports guide](docs/guides/ai-reports.md).

### Other Commands

| Command | Description | Reference |
|---------|-------------|-----------|
| `gaze self-check` | Run CRAP analysis on Gaze's own source code | [`self-check`](docs/reference/cli/self-check.md) |
| `gaze docscan` | Scan repository for documentation files (JSON output) | [`docscan`](docs/reference/cli/docscan.md) |
| `gaze schema` | Print the JSON Schema for `gaze analyze --format=json` output | [`schema`](docs/reference/cli/schema.md) |
| `gaze init` | Scaffold OpenCode agent and command files | [`init`](docs/reference/cli/init.md) |

## CI Integration

Use threshold flags for CI enforcement. Gaze exits non-zero when limits are exceeded:

```bash
gaze crap --max-crapload=5 --max-gaze-crapload=3 ./...
```

For complete GitHub Actions workflow examples, coverage profile reuse, and threshold selection guidance, see the [CI Integration guide](docs/guides/ci-integration.md).

## Baseline Comparison

Gaze supports per-function CRAP and GazeCRAP regression detection by comparing current scores against a saved baseline. This fulfills the constitutional mandate that "Metrics MUST be comparable across runs so users can measure progress over time."

### Creating a Baseline

Save the current CRAP scores as a baseline:

```bash
mkdir -p .gaze
go test -coverprofile=coverage.out ./...
gaze crap --format=json --coverprofile=coverage.out ./... > .gaze/baseline.json
git add .gaze/baseline.json && git commit -m "chore: add CRAP baseline"
```

### How It Works

When `.gaze/baseline.json` exists, `gaze crap` auto-detects it and activates comparison mode. Each function is matched by `file:function` key, and CRAP/GazeCRAP deltas are computed. Functions are classified as `regression`, `improvement`, `unchanged`, `new`, `new_violation`, or `removed`.

The comparison produces a pass/fail gate: exit code 1 when any regression or new-function violation (CRAP above threshold) is detected. This gate is independent of the `--max-crapload` / `--max-gaze-crapload` threshold gates.

### Overriding and Opting Out

```bash
gaze crap --baseline path/to/other-baseline.json ./...   # Use a custom baseline path
```

### CI Integration with Baseline

Commit `.gaze/baseline.json` to version control so CI can detect regressions on every PR:

```yaml
- name: Run tests with coverage
  run: go test -coverprofile=coverage.out ./...
- name: Check for regressions
  run: gaze crap --coverprofile=coverage.out ./...
```

When no baseline file exists, `gaze crap` behaves identically to before — no comparison, no error, exit 0.

### Configuration

Baseline settings can be customized in `.gaze.yaml`:

```yaml
baseline:
  file: .gaze/baseline.json   # default
  epsilon: 0.5                 # score change tolerance
  new_function_threshold: 30   # max CRAP for new functions
```

For full configuration details, see [Configuration Reference](docs/reference/configuration.md).

## Output Formats

The `analyze`, `crap`, `quality`, and `self-check` commands support `--format=text` (default) and `--format=json`.

JSON output conforms to documented schemas. Use `gaze schema` to print the analysis report schema. See [JSON Schemas](docs/reference/json-schemas.md) for annotated examples.

## OpenCode Integration

After running `gaze init`, use the `/gaze` command in OpenCode for AI-assisted quality reporting:

```text
/gaze ./...                     # Full report: CRAP + quality + classification
/gaze crap ./internal/store     # CRAP scores only
/gaze quality ./pkg/api         # Test quality metrics only
```

For setup details, see the [OpenCode Integration guide](docs/guides/opencode-integration.md).

## Known Limitations

- **Direct function body only.** Gaze analyzes the immediate function body. Transitive side effects (effects produced by called functions) are out of scope for v1.
- **P3-P4 side effects not yet detected.** The taxonomy defines types for stdout/stderr writes, environment mutations, mutex operations, reflection, unsafe, and other P3-P4 effects, but detection logic is not yet implemented for these tiers.
- **GazeCRAP accuracy is limited.** The quality pipeline is wired into the CRAP command and GazeCRAP scores are computed when contract coverage data is available. However, assertion-to-side-effect mapping accuracy is currently ~86% (target: 90%), primarily affecting cross-target assertions and go-cmp patterns (tracked as GitHub Issue #6).
- **No CGo or unsafe analysis.** Functions using `cgo` or `unsafe.Pointer` are not analyzed for their specific side effects.
- **Single package loading.** The `analyze` command processes one package at a time. Use shell loops or scripting for multi-package analysis.

## License

Apache License 2.0. See [LICENSE](LICENSE) for details.
