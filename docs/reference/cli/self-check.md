# gaze self-check

Run CRAP analysis on Gaze's own source code. Serves as both a dogfooding exercise and a code quality gate. Reports [CRAPload](../glossary.md) and the worst offenders by CRAP score. [GazeCRAP](../glossary.md) scores are included when contract coverage data is available from the integrated quality pipeline.

## Synopsis

```
gaze self-check [flags]
```

## Arguments

This command takes no positional arguments. It automatically analyzes the `./...` pattern from the module root.

**Module root detection**: `self-check` walks up from the current working directory to find the nearest `go.mod` file. This ensures it always analyzes the full module, even when invoked from a subdirectory.

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--format` | `string` | `text` | Output format: `text` or `json` |
| `--max-crapload` | `int` | `0` (no limit) | CI gate: fail if CRAPload exceeds this count |
| `--max-gaze-crapload` | `int` | `0` (no limit) | CI gate: fail if GazeCRAPload exceeds this count |

## Configuration Interaction

The `gaze self-check` command does not read `.gaze.yaml`. It uses default CRAP options (threshold: 15) and analyzes the entire module.

Internally, `self-check` delegates to the same CRAP pipeline as `gaze crap`, with the package pattern hardcoded to `./...` and the module directory resolved automatically.

Because the pipeline is shared, `self-check` is subject to the nested coverage guard: when it inherits a non-empty `GAZE_COVERAGE_RUN` (i.e., it runs inside a Gaze coverage collection), coverage generation is refused with `ErrNestedCoverageRun` instead of recursing. Gaze's own self-analyzing tests skip under this variable. Normal runs are unaffected.

## Examples

### Basic self-check

```bash
gaze self-check
```

Runs CRAP analysis on Gaze's own source code and prints the report to stdout.

### CI quality gate

```bash
gaze self-check --max-crapload=10 --max-gaze-crapload=5
```

Exits with code 1 if CRAPload exceeds 10 or GazeCRAPload exceeds 5.

### JSON output

```bash
gaze self-check --format=json | jq '.summary.crapload'
```

The JSON output has the same structure as `gaze crap --format=json`.

## See Also

- [Scoring](../../concepts/scoring.md) — CRAP formula and CRAPload definition
- [`gaze crap`](crap.md) — CRAP analysis on any package (self-check uses this internally)
- [Improving Scores](../../guides/improving-scores.md) — how to reduce CRAPload
