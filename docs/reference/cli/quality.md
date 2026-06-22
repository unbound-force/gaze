# gaze quality

Assess how well a package's tests assert on the [contractual](../glossary.md) side effects of the functions they test. Reports [Contract Coverage](../glossary.md) (ratio of contractual effects that are asserted on) and [Over-Specification](../glossary.md) Score (assertions on incidental implementation details).

The target package must have existing `*_test.go` files.

## Synopsis

```
gaze quality [packages...] [flags]
```

## Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| `packages...` | Yes (at least one) | One or more Go package import paths, relative paths, or patterns (e.g., `./internal/crap`, `./...`) |

At least one package argument is required. Wildcard patterns like `./...` are expanded to all matching packages. Packages without test files are skipped with a warning.

**Auto-detection**: When a target package is `package main`, unexported functions are automatically included (a `main` package has no exported API by definition). This behavior is equivalent to passing `--include-unexported`. Detection is applied per package.

## Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--format` | | `string` | `text` | Output format: `text` or `json` |
| `--target` | | `string` | `""` (all) | Restrict analysis to tests that exercise this specific function |
| `--verbose` | `-v` | `bool` | `false` | Show detailed assertion and mapping information |
| `--include-unexported` | | `bool` | `false` | Include unexported functions (auto-enabled for `package main`) |
| `--config` | | `string` | `""` (search CWD) | Path to `.gaze.yaml` config file |
| `--contractual-threshold` | | `int` | `-1` (from config or 80) | Override contractual confidence threshold |
| `--incidental-threshold` | | `int` | `-1` (from config or 50) | Override incidental confidence threshold |
| `--min-contract-coverage` | | `int` | `0` (no limit) | CI gate: fail if any test's contract coverage is below this percentage |
| `--max-over-specification` | | `int` | `0` (no limit) | CI gate: fail if any test's over-specification count exceeds this value |
| `--ai-mapper` | | `string` | `""` | AI backend for assertion mapping fallback: `claude`, `gemini`, `ollama`, or `opencode` |
| `--ai-mapper-model` | | `string` | `""` | Model name for AI mapper (required for `ollama`) |

## Configuration Interaction

The following flags interact with `.gaze.yaml`:

| Flag | Config Key | Behavior |
|------|-----------|----------|
| `--config` | — | Specifies the config file path. If omitted, Gaze searches for `.gaze.yaml` in the current working directory. |
| `--contractual-threshold` | `classification.thresholds.contractual` | Overrides the config value when set. Valid range: 1–99. |
| `--incidental-threshold` | `classification.thresholds.incidental` | Overrides the config value when set. Valid range: 1–99. |

Classification thresholds directly affect contract coverage: a lower contractual threshold means more effects are classified as contractual, which changes the denominator of the coverage ratio.

See [Configuration Reference](../configuration.md) for all `.gaze.yaml` options.

## CI Threshold Behavior

The `--min-contract-coverage` and `--max-over-specification` thresholds are evaluated **per test-target pair**, not on the package average. This means every individual test must meet the threshold, not just the average across all tests.

When SSA construction fails (degraded mode), CI thresholds are automatically skipped to avoid false-positive failures from zero-valued metrics. A warning is printed to stderr.

## Examples

### Basic quality assessment

```bash
gaze quality ./internal/crap
```

```
Test Quality Report
═══════════════════════════════════════════════════════════════

  Test: TestFormula_ZeroCoverage
  Target: Formula
  Contract Coverage: 100% (1/1 contractual effects asserted)
  Over-Specification: 0 (0.00 ratio)

  ...

Summary
  Total tests: 12
  Average contract coverage: 85.0%
  Total over-specifications: 2
```

### Assess all packages in a module

```bash
gaze quality ./...
```

Expands `./...` to all packages with test files and reports quality metrics for each. Packages without tests are skipped with a warning.

### CI quality gate

```bash
gaze quality ./internal/crap --min-contract-coverage=80
```

Exits with code 1 if any test's contract coverage falls below 80%. All violations are reported at once.

### Verbose output with assertion mapping details

```bash
gaze quality ./internal/crap --verbose
```

Shows which assertions map to which side effects, the mapping confidence level, and the mapping pass that matched (direct identity, indirect root, helper bridge, or inline call).

### JSON output

```bash
gaze quality ./internal/crap --format=json | jq '.quality_summary'
```

See [JSON Schemas](../json-schemas.md) for the full output structure.

## See Also

- [Quality](../../concepts/quality.md) — test-target pairing, assertion mapping, and contract coverage
- [Classification](../../concepts/classification.md) — how contractual/incidental labels affect coverage
- [JSON Schemas](../json-schemas.md) — schema reference for `--format=json` output
- [Configuration](../configuration.md) — `.gaze.yaml` options including classification thresholds
- [`gaze analyze`](analyze.md) — detect side effects (step 1 of the quality pipeline)
- [`gaze crap`](crap.md) — CRAP scoring (uses contract coverage from quality)
