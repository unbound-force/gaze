# gaze analyze

Analyze one or more Go packages (or a specific function) and report all observable side effects each function produces.

Side effects are categorized into priority tiers (P0–P4) from the [side effect taxonomy](../../concepts/side-effects.md). Use `--classify` to attach [contractual classification](../../concepts/classification.md) labels (mechanical signals only). For document-enhanced classification, use the `/gaze` command in OpenCode (full mode).

## Synopsis

```
gaze analyze [packages...] [flags]
```

## Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| `packages...` | Yes (at least one) | One or more Go package import paths, relative paths, or patterns (e.g., `./internal/crap`, `./...`, `github.com/foo/bar`) |

At least one package argument is required. Wildcard patterns like `./...` are expanded to all matching packages.

## Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--format` | | `string` | `text` | Output format: `text`, `json`, or `html` |
| `--function` | `-f` | `string` | `""` (all exported) | Analyze a specific function by name |
| `--include-unexported` | | `bool` | `false` | Include unexported (lowercase) functions in analysis |
| `--interactive` | `-i` | `bool` | `false` | Launch interactive TUI (Bubble Tea) for browsing results |
| `--classify` | | `bool` | `false` | Classify side effects as contractual, incidental, or ambiguous |
| `--verbose` | `-v` | `bool` | `false` | Print full signal breakdown (implies `--classify`) |
| `--config` | | `string` | `""` (search CWD) | Path to `.gaze.yaml` config file |
| `--contractual-threshold` | | `int` | `-1` (from config or 80) | Override contractual confidence threshold |
| `--incidental-threshold` | | `int` | `-1` (from config or 50) | Override incidental confidence threshold |

## Configuration Interaction

The following flags interact with `.gaze.yaml`:

| Flag | Config Key | Behavior |
|------|-----------|----------|
| `--config` | — | Specifies the config file path. If omitted, Gaze searches for `.gaze.yaml` in the current working directory. |
| `--contractual-threshold` | `classification.thresholds.contractual` | Overrides the config value when set. Valid range: 1–99. Must be greater than the incidental threshold. |
| `--incidental-threshold` | `classification.thresholds.incidental` | Overrides the config value when set. Valid range: 1–99. Must be less than the contractual threshold. |

The `--classify` and `--verbose` flags trigger classification, which loads the config file. Without these flags, the config file is not read.

See [Configuration Reference](../configuration.md) for all `.gaze.yaml` options.

## Examples

### Analyze a package (text output)

```bash
gaze analyze ./internal/crap
```

```
Function: Formula
  Package: github.com/unbound-force/gaze/v2/internal/crap
  Signature: func Formula(complexity int, coveragePct float64) float64
  Location: internal/crap/crap.go:152:1
  Side Effects:
    [P0] ReturnValue: returns float64 value (se-a1b2c3d4)
```

### Analyze with classification

```bash
gaze analyze ./internal/crap --classify
```

Side effects are labeled `contractual`, `incidental`, or `ambiguous` based on mechanical signal analysis.

### Analyze a single function with verbose output

```bash
gaze analyze ./internal/crap -f Formula --verbose
```

Shows the full signal breakdown for each side effect, including individual signal sources (interface, visibility, caller, naming, godoc) and their weight contributions.

### Analyze all packages in a module

```bash
gaze analyze ./...
```

Expands `./...` to all packages in the module and reports side effects for every exported function across all packages.

### Analyze multiple specific packages

```bash
gaze analyze ./internal/crap ./internal/loader
```

### JSON output for machine consumption

```bash
gaze analyze ./internal/crap --format=json | jq '.results[0].side_effects'
```

The JSON output conforms to the [Analysis JSON Schema](../json-schemas.md). Use `gaze schema` to print the full schema.

### HTML report saved to a file

```bash
gaze analyze ./internal/crap --format=html > report.html
```

Produces a self-contained HTML document with all styling inline. The report uses native `<details>` and `<summary>` elements for collapsible function sections, requires no JavaScript, and loads no external resources — it works offline as a single file. All source-derived content is contextually escaped via Go's `html/template` package. Open `report.html` directly in any browser to browse the results.

> **Note:** HTML output is currently supported only by `gaze analyze`. Other commands (`gaze quality`, `gaze crap`, `gaze report`) continue to accept `text` and `json` only.

## See Also

- [Side Effects](../../concepts/side-effects.md) — the 37 effect types and 5 priority tiers
- [Classification](../../concepts/classification.md) — how contractual/incidental labels are computed
- [JSON Schemas](../json-schemas.md) — schema reference for `--format=json` output
- [Configuration](../configuration.md) — `.gaze.yaml` options
- [`gaze schema`](schema.md) — print the embedded JSON Schema
- [`gaze quality`](quality.md) — assess how well tests cover the detected side effects
