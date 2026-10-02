# gaze report

Run Gaze's CRAP, quality, classification, and docscan operations together to produce a text, JSON, or HTML report. Text output uses an external AI adapter. Gaze generates JSON and HTML directly.

The formatted report is written to stdout and optionally appended to `$GITHUB_STEP_SUMMARY` for GitHub Actions Step Summary integration.

## Synopsis

```text
gaze report [packages] [flags]
```

## Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| `packages` | No | Go package patterns (e.g., `./...`, `./internal/...`). Defaults to `./...` when omitted. |

When no package arguments are provided, `./...` is used automatically (the entire module).

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--format` | `string` | `text` | Output format: `text` (AI-formatted markdown), `json` (raw analysis payload), or `html` (native combined report) |
| `--ai` | `string` | `""` | AI adapter: `claude`, `gemini`, `ollama`, or `opencode`. **Required in text mode.** |
| `--model` | `string` | `""` | Model name for the AI adapter. **Required for `ollama`**; optional for other adapters. |
| `--ai-timeout` | `duration` | `10m` | Maximum time to wait for the AI adapter to respond. Uses Go duration format (e.g., `5m`, `30s`, `2m30s`). |
| `--coverprofile` | `string` | `""` | Path to a pre-generated Go coverage profile. Skips the internal `go test -coverprofile` run. |
| `--max-crapload` | `int` | not set | CI gate: fail if CRAPload exceeds N. Absent by default (no enforcement). |
| `--max-gaze-crapload` | `int` | not set | CI gate: fail if GazeCRAPload exceeds N. Absent by default (no enforcement). |
| `--min-contract-coverage` | `int` | not set | CI gate: fail if average contract coverage is below N%. Absent by default (no enforcement). |

### Threshold Semantics

The threshold flags (`--max-crapload`, `--max-gaze-crapload`, `--min-contract-coverage`) use pointer semantics internally: when a flag is not provided, no threshold is enforced. When explicitly set — even to `0` — the threshold is active. This means `--max-crapload=0` will fail if any function is in the CRAPload (i.e., zero tolerance).

## Configuration Interaction

The `gaze report` command does not directly read `.gaze.yaml`. The underlying analysis pipeline uses default classification thresholds. To customize thresholds, use `gaze crap` and `gaze quality` individually.

The `--coverprofile` flag is the key CI optimization — pass a coverage profile generated during your test step to avoid running tests twice.

## AI Adapter Details

| Adapter | Binary | System Prompt Delivery | Payload Delivery |
|---------|--------|----------------------|-----------------|
| `claude` | `claude` | Temp file via `-p` flag | stdin |
| `gemini` | `gemini` | `GEMINI.md` in temp dir | stdin |
| `ollama` | HTTP API | `system` field in JSON body | `prompt` field in JSON body |
| `opencode` | `opencode` | `.opencode/agents/gaze-reporter.md` in temp dir via `--dir` | stdin |

For text output, Gaze validates that the adapter binary exists on `PATH` (or that the Ollama HTTP API is reachable) before analysis starts. An invalid binary produces a hard exit before any analysis runs.

## Native HTML Output

Select `--format=html` to generate a complete combined report without an AI adapter. In HTML mode, `--ai` is optional: Gaze does not validate or invoke an adapter, and it does not load an AI prompt.

The report includes typed summary metrics and separate CRAP, quality, classification, and documentation sections. Successful steps show their available pipeline data as formatted JSON. If a step fails, its section shows the recorded error while output from successful steps remains available. Metrics that were not computed are shown as unavailable rather than as measured zero values.

HTML output is a self-contained file with inline styling. It does not load scripts, stylesheets, fonts, images, or other network resources. Its content remains readable with JavaScript disabled.

## Environment Variables

| Variable | Description |
|----------|-------------|
| `GITHUB_STEP_SUMMARY` | When set, the formatted report is appended to this file path for GitHub Actions Step Summary display. Symlink protection (`O_NOFOLLOW`) is applied. |

## Examples

### AI-formatted report with Claude

```bash
gaze report ./... --ai=claude
```

### JSON-only output (no AI required)

```bash
gaze report ./... --format=json > report.json
```

In JSON mode, the `--ai` flag is not required. The raw analysis payload is written directly to stdout.

### Native HTML report written to a file

```bash
gaze report ./... --format=html > gaze-report.html
```

No `--ai` flag is needed. Open `gaze-report.html` directly in a browser; the file does not require network access.

### CI integration with thresholds

```bash
# Generate coverage during test step
go test -race -count=1 -coverprofile=coverage.out ./...

# Run report with pre-generated coverage and quality gates
gaze report ./... \
  --ai=opencode \
  --coverprofile=coverage.out \
  --max-crapload=35 \
  --max-gaze-crapload=5 \
  --min-contract-coverage=8
```

### Using Ollama (local model)

```bash
gaze report ./... --ai=ollama --model=llama3.2
```

The `--model` flag is required for Ollama and specifies which model to use for formatting.

## See Also

- [CI Integration](../../guides/ci-integration.md) — full GitHub Actions workflow setup
- [AI Reports](../../guides/ai-reports.md) — adapter setup and configuration
- [Scoring](../../concepts/scoring.md) — understanding the metrics in the report
- [`gaze crap`](crap.md) — standalone CRAP analysis
- [`gaze quality`](quality.md) — standalone quality analysis
- [`gaze docscan`](docscan.md) — standalone document scanning
