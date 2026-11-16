# Configuration Reference

Gaze is configured via a `.gaze.yaml` file in the project root. All settings are optional — Gaze ships with sensible defaults and works without a config file.

## File Location

Gaze searches for `.gaze.yaml` in the current working directory. You can override this with the `--config` flag on commands that support it ([`analyze`](cli/analyze.md), [`quality`](cli/quality.md), [`docscan`](cli/docscan.md)).

If no config file is found, Gaze uses the default configuration silently (no error).

## Complete Example

```yaml
# .gaze.yaml — Gaze configuration
classification:
  thresholds:
    contractual: 80    # Confidence >= 80 → contractual
    incidental: 50     # Confidence < 50 → incidental
                       # 50–79 → ambiguous
  doc_scan:
    exclude:
      - "vendor/**"
      - "node_modules/**"
      - ".git/**"
      - "testdata/**"
      - "CHANGELOG.md"
      - "CONTRIBUTING.md"
      - "CODE_OF_CONDUCT.md"
      - "LICENSE"
      - "LICENSE.md"
    include: []        # Empty = scan all non-excluded files
    timeout: "30s"

baseline:
  file: .gaze/baseline.json   # Path to baseline file
  epsilon: 0.5                 # Minimum score delta for regression/improvement
  new_function_threshold: 30   # CRAP score above which a new function is a violation
```

## Configuration Keys

### `classification`

Top-level section for all [classification](../concepts/classification.md)-related settings.

---

### `classification.thresholds`

Confidence score boundaries that determine how side effects are labeled.

| Key | Type | Default | Valid Range | Description |
|-----|------|---------|-------------|-------------|
| `contractual` | `int` | `80` | 1–99 | Minimum confidence for the **contractual** label. Side effects with confidence scores at or above this value are classified as contractual. |
| `incidental` | `int` | `50` | 1–99 | Upper bound for the **incidental** label. Side effects with confidence scores below this value are classified as incidental. |

**Constraint**: `contractual` must be strictly greater than `incidental`. If this constraint is violated (either in the config file or via CLI flag overrides), Gaze exits with an error explaining which source caused the invalid configuration.

**Label assignment logic**:

- Confidence >= `contractual` → **contractual**
- Confidence < `incidental` → **incidental**
- Otherwise → **ambiguous**

**Tier-based boosts**: Before threshold comparison, P0 effects (ReturnValue, ErrorReturn, SentinelError, ReceiverMutation, PointerArgMutation) receive a +25 confidence boost (base 75), and P1 effects receive a +10 boost (base 60). This means P0 effects trend toward contractual by default.

---

### `classification.doc_scan`

Controls which documentation files are scanned for classification signals.

#### `classification.doc_scan.exclude`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `exclude` | `[]string` | See below | Glob patterns for files to exclude from document scanning |

**Default exclude patterns**:

```yaml
exclude:
  - "vendor/**"
  - "node_modules/**"
  - ".git/**"
  - "testdata/**"
  - "CHANGELOG.md"
  - "CONTRIBUTING.md"
  - "CODE_OF_CONDUCT.md"
  - "LICENSE"
  - "LICENSE.md"
```

These defaults prevent scanning dependency directories and boilerplate files that rarely contain classification-relevant information.

#### `classification.doc_scan.include`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `include` | `[]string` | `null` (scan all) | Glob patterns for files to include. When set, **only** matching files are processed, overriding the default full-repo scan. |

When `include` is set, only files matching at least one include pattern (and not matching any exclude pattern) are scanned. When `include` is empty or null, all non-excluded Markdown files are scanned.

#### `classification.doc_scan.timeout`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `timeout` | `string` | `"30s"` | Maximum duration for document scanning. Uses Go duration format (e.g., `"30s"`, `"1m"`, `"2m30s"`). |

### `baseline`

Top-level section for [baseline comparison](../../README.md#baseline-comparison) settings. Controls how `gaze crap` detects regressions by comparing current CRAP/GazeCRAP scores against a saved baseline.

All fields are optional — when omitted, sensible defaults are used.

| Key | Type | Default | Valid Range | Description |
|-----|------|---------|-------------|-------------|
| `file` | `string` | `.gaze/baseline.json` | Any valid file path | Path to the baseline JSON file. This is the auto-detection path — `gaze crap` loads this file automatically when it exists. |
| `epsilon` | `float64` | `0.5` | >= 0 | Minimum score delta to trigger a regression or improvement. Score changes within epsilon are classified as `unchanged`. The default (0.5) absorbs platform/toolchain noise without masking real regressions. |
| `new_function_threshold` | `float64` | `30` | > 0 | CRAP score above which a new function (not present in the baseline) is flagged as a violation. New functions below this threshold are reported as informational. |

**Validation rules**:

- `epsilon` must be >= 0. A negative epsilon produces an error.
- `new_function_threshold` must be > 0. A zero or negative value produces an error.

**Interaction with CLI flags**: The `--baseline` flag overrides the `file` path. There are no CLI flags for `epsilon` or `new_function_threshold` — these are configured only via `.gaze.yaml` since they rarely change.

---

## Doc-Contract Sidecar File

For [external-analyzer](../porting/contracts.md) projects, Gaze reads an optional sidecar file to derive [classification](../concepts/classification.md) signals from explicit declarations. This file is **not** part of `.gaze.yaml` — it lives at `.uf/gaze/contracts.yaml` (primary) or `.uf/gaze/contracts.json` (alternative, for tool-generated sidecars).

### Schema

```yaml
# .uf/gaze/contracts.yaml
version: 1
contracts:
  - package: "mypackage"
    function: "process_items"
    side_effect_type: "ContainerMutation"
    label: contractual
    reasoning: "public API mutation per design doc"
```

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `version` | `int` | No | Schema version (reserved) |
| `contracts` | `array` | Yes | List of contract declarations |
| `contracts[].package` | `string` | Yes | Package the function belongs to |
| `contracts[].function` | `string` | Yes | The analyzer's verbatim function name (bare name, not receiver-qualified) |
| `contracts[].side_effect_type` | `string` | Yes | A known effect type name (from the [taxonomy](../porting/taxonomy-reference.md)) |
| `contracts[].label` | `string` | Yes | `contractual` or `incidental` (case-insensitive) |
| `contracts[].reasoning` | `string` | No | Human-readable justification; defaults to a generated message when omitted |

### Precedence

Sidecar declarations carry the highest signal weight (`±30`) and override grammar-derived signals from Markdown docs for the same `(package, function, side_effect_type)` tuple. When both `.uf/gaze/contracts.yaml` and `.uf/gaze/contracts.json` exist, YAML takes precedence.

### Graceful Degradation

The sidecar file is opt-in and degrades gracefully:

- **Absent file** → no sidecar signals, no error, no warning. Grammar-derived signals (if any) are still used.
- **Malformed file** (invalid YAML/JSON) → a warning is written to stderr and the sidecar is ignored; grammar-derived signals are still used.
- **Semantically invalid entry** (unknown `label`, empty `package` or `function`, or unknown `side_effect_type`) → that entry is skipped with a warning to stderr; remaining valid entries are still loaded.

A failing sidecar step never blocks analysis — classification simply produces more `ambiguous` results in its absence.

### Scope

The sidecar file is only read for external-analyzer projects. Go-native analysis ignores it and uses the GoDoc analyzer instead. See [Document Annotations](../guides/doc-annotations.md) for the full grammar and examples.

---

## CLI Flag Overrides

Several CLI flags override config file values. The CLI flag always takes precedence when explicitly set.

| CLI Flag | Config Key | Commands |
|----------|-----------|----------|
| `--contractual-threshold` | `classification.thresholds.contractual` | [`analyze`](cli/analyze.md), [`quality`](cli/quality.md) |
| `--incidental-threshold` | `classification.thresholds.incidental` | [`analyze`](cli/analyze.md), [`quality`](cli/quality.md) |
| `--config` | — (specifies file path) | [`analyze`](cli/analyze.md), [`quality`](cli/quality.md), [`docscan`](cli/docscan.md) |
| `--baseline` | `baseline.file` | [`crap`](cli/crap.md) |

**Override semantics**: A CLI flag value of `-1` (the default) means "use the config file value." Any other value in the valid range (1–99) overrides the config. The threshold coherence constraint (`contractual > incidental`) is validated after merging CLI and config values.

## Validation Rules

1. **Threshold range**: Both `contractual` and `incidental` must be integers in [1, 99].
2. **Threshold ordering**: `contractual` must be strictly greater than `incidental`.
3. **Timeout format**: Must be a valid Go duration string (parsed by `time.ParseDuration`).
4. **Glob patterns**: Must be valid glob patterns (parsed by Go's `filepath.Match`).
5. **YAML syntax**: The file must be valid YAML. Parse errors produce a descriptive error message with the file path.
6. **Baseline epsilon**: Must be >= 0.
7. **Baseline new_function_threshold**: Must be > 0.

## Error Messages

When configuration is invalid, Gaze produces actionable error messages that identify the source:

```text
# Invalid threshold from config file
contractual threshold (40) must be greater than incidental threshold (50); check config file .gaze.yaml

# Invalid threshold from CLI flag
--contractual-threshold=30 is invalid: must be in [1, 99]

# Conflict between CLI flag and config
contractual threshold (50) must be greater than incidental threshold (50); check --contractual-threshold flag

# Invalid baseline config
baseline epsilon (-1) must be >= 0; check config file .gaze.yaml

# Invalid baseline threshold
baseline new_function_threshold (0) must be > 0; check config file .gaze.yaml
```

## See Also

- [Classification](../concepts/classification.md) — how thresholds affect classification labels
- [`gaze analyze`](cli/analyze.md) — uses config for classification
- [`gaze crap`](cli/crap.md) — uses config for baseline comparison settings
- [`gaze quality`](cli/quality.md) — uses config for classification and contract coverage
- [`gaze docscan`](cli/docscan.md) — uses config for document scanning settings
- [Glossary](glossary.md) — definitions of contractual, incidental, and ambiguous
