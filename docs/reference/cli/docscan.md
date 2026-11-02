# gaze docscan

Scan the repository for Markdown documentation files and output a prioritized list as JSON. This command also feeds the full-mode documentation section of [`gaze report`](report.md) for document-enhanced [classification](../../concepts/classification.md).

Documents are prioritized by proximity to the target package:

| Priority | Location |
|----------|----------|
| 1 (highest) | Same directory as the target package |
| 2 | Module root |
| 3 | Other locations |

## Synopsis

```
gaze docscan [package] [flags]
```

## Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| `package` | No | Go package import path or relative path. Defaults to `.` (current directory). |

When a relative path is provided (starting with `./` or `../`), it is resolved to an absolute path for directory-level priority matching.

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--config` | `string` | `""` (search CWD) | Path to `.gaze.yaml` config file |
| `--analyzer` | `string` | `""` | External analyzer binary (e.g., `snake-eyes`) |
| `--language` | `string` | `""` | Target language for analyzer discovery (e.g., `python`) |

## Configuration Interaction

The `--config` flag loads `.gaze.yaml` for document scanning settings:

| Config Key | Type | Default | Description |
|-----------|------|---------|-------------|
| `classification.doc_scan.exclude` | `[]string` | See below | Glob patterns for files to exclude from scanning |
| `classification.doc_scan.include` | `[]string` | `null` (scan all) | Glob patterns for files to include. When set, only matching files are processed. |
| `classification.doc_scan.timeout` | `string` | `"30s"` | Maximum duration for document scanning (Go duration format) |

**Default exclude patterns**:
- `vendor/**`
- `node_modules/**`
- `.git/**`
- `testdata/**`
- `CHANGELOG.md`
- `CONTRIBUTING.md`
- `CODE_OF_CONDUCT.md`
- `LICENSE`
- `LICENSE.md`

See [Configuration Reference](../configuration.md) for all `.gaze.yaml` options.

## Output Format

The output is always JSON (there is no `--format` flag). Documents are wrapped in an envelope with an optional API coverage report:

```json
{
  "documents": [
    {
      "path": "internal/crap/README.md",
      "priority": 1
    },
    {
      "path": "README.md",
      "priority": 2
    },
    {
      "path": "docs/concepts/scoring.md",
      "priority": 3
    }
  ],
  "api_coverage": null
}
```

When `--analyzer` or `--language` is provided, `api_coverage` holds the documentation coverage report (`total_symbols`, `documented_symbols`, `coverage_percent`, `undocumented`, `stale_references`, `code_block_issues`); otherwise it is `null`.

## Examples

### Scan for a specific package

```bash
gaze docscan ./internal/crap
```

Returns all Markdown files in the repository, prioritized by proximity to the `internal/crap` directory.

### Scan with custom config

```bash
gaze docscan ./internal/crap --config=/path/to/.gaze.yaml
```

### Pipe to the AI report pipeline

The docscan output is typically consumed by `gaze report` internally, but can be used standalone for debugging or custom pipelines:

```bash
gaze docscan ./internal/crap | jq '.documents[].path'
```

## See Also

- [Classification](../../concepts/classification.md) — how document signals contribute to classification
- [Configuration](../configuration.md) — `.gaze.yaml` doc_scan settings
- [`gaze report`](report.md) — the AI report pipeline that uses docscan internally
- [`gaze analyze`](analyze.md) — side effect detection (step 1 before classification)
