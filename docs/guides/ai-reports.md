# AI Reports

Gaze can pipe its analysis data to an external AI model to add judgment to a human-readable quality report. The `gaze report --ai=<adapter>` command orchestrates four analysis operations (CRAP scoring, quality assessment, classification, and documentation scanning), combines the results into a JSON payload, and sends it to the chosen AI adapter. Gaze renders the report itself; the adapter supplies only a validated summary and prioritized recommendations.

## How It Works

1. Gaze runs its full analysis pipeline on the specified packages
2. The results are combined into a single JSON payload
3. Gaze grades the results and renders the report — tables, metrics, grades, and deterministic recommendations — from its own rubric
4. The payload and the embedded report-mode system prompt are sent to the AI adapter. Gaze never reads the project's interactive `.opencode/agents/gaze-reporter.md`; it prints a notice when that file exists
5. The adapter returns one JSON object holding a plain-prose summary and up to five prioritized recommendations, which Gaze validates before rendering them into the AI slots
6. The report is written to stdout (and optionally to `$GITHUB_STEP_SUMMARY`)

The AI adapter adds judgment only — it does not perform analysis, choose grades, or write the report. All metrics, scores, grade boundaries, and deterministic recommendations are computed by Gaze before the AI sees them, so identical analysis data always produces an identical report outside the AI slots.

## Supported Adapters

Four AI adapters are available:

| Adapter | Binary | Transport | Model Required? |
|---------|--------|-----------|-----------------|
| `claude` | `claude` | Subprocess (stdin/stdout) | No (uses CLI default) |
| `gemini` | `gemini` | Subprocess (stdin/stdout) | No (uses CLI default) |
| `ollama` | — | HTTP API (`/api/generate`) | **Yes** |
| `opencode` | `opencode` | Subprocess (stdin/stdout) | No (uses configured default) |

## Claude

The Claude adapter invokes the `claude` CLI as a subprocess. The system prompt is written to a temporary file and passed via `--system-prompt-file` to avoid OS argument length limits.

### Prerequisites

- Install the Claude CLI: [claude.ai/cli](https://claude.ai/cli)
- Set the `ANTHROPIC_API_KEY` environment variable

### Usage

```bash
# Use Claude's default model
gaze report ./... --ai=claude

# Specify a model
gaze report ./... --ai=claude --model=claude-sonnet-4-20250514
```

### CI Configuration

```yaml
- name: Gaze quality report
  run: gaze report ./... --ai=claude --coverprofile=coverage.out
  env:
    ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
```

### How It Works Internally

1. System prompt is written to a temp file (`gaze-claude-prompt-*/prompt.md`)
2. Claude is invoked: `claude -p "" --system-prompt-file <path> [--model <name>]`
3. The JSON payload is piped to stdin
4. Formatted report is read from stdout
5. Temp file is cleaned up

## Gemini

The Gemini adapter invokes the `gemini` CLI as a subprocess. The system prompt is written as `GEMINI.md` in a temporary directory, because the Gemini CLI reads its system prompt from that file in the working directory.

### Prerequisites

- Install the Gemini CLI: [github.com/google-gemini/gemini-cli](https://github.com/google-gemini/gemini-cli)
- Authenticate via `gemini auth` or set the appropriate API key

### Usage

```bash
# Use Gemini's default model
gaze report ./... --ai=gemini

# Specify a model
gaze report ./... --ai=gemini --model=gemini-2.5-pro
```

### CI Configuration

```yaml
- name: Gaze quality report
  run: gaze report ./... --ai=gemini --coverprofile=coverage.out
  env:
    GOOGLE_API_KEY: ${{ secrets.GOOGLE_API_KEY }}
```

### How It Works Internally

1. A temp directory is created (`gaze-gemini-*/`)
2. System prompt is written as `GEMINI.md` in that directory
3. Gemini is invoked with `cmd.Dir` set to the temp directory: `gemini -p "" --output-format json [-m <name>]`
4. The JSON payload is piped to stdin
5. Response is parsed from Gemini's JSON output (`{"response": "..."}`)
6. Temp directory is cleaned up

## Ollama

The Ollama adapter uses the HTTP API (`/api/generate`) rather than a CLI subprocess, because the Ollama CLI has no system prompt flag.

### Prerequisites

- Install Ollama: [ollama.com](https://ollama.com)
- Pull a model: `ollama pull llama3.2`
- Ensure the Ollama server is running: `ollama serve`

### Usage

```bash
# Model is required for Ollama
gaze report ./... --ai=ollama --model=llama3.2

# Use a different model
gaze report ./... --ai=ollama --model=deepseek-r1:70b
```

The `--model` flag is **required** when using the Ollama adapter. Omitting it produces an error.

### Custom Host

By default, Ollama connects to `http://localhost:11434`. Override this with the `OLLAMA_HOST` environment variable:

```bash
OLLAMA_HOST=http://my-gpu-server:11434 gaze report ./... --ai=ollama --model=llama3.2
```

The host must be an absolute HTTP or HTTPS URL with a valid host component.

### How It Works Internally

1. The payload is read from the input reader
2. A JSON request is built: `{"model": "<name>", "system": "<prompt>", "prompt": "<payload>", "stream": false}`
3. The request is POSTed to `<host>/api/generate`
4. The response JSON is parsed (`{"response": "..."}`)
5. The response text is returned as the formatted report

### Limitations

- Ollama does not implement the `AdapterValidator` interface — there is no pre-flight binary check. Errors are reported at request time.
- Large payloads may exceed the context window of smaller models. Use a model with at least 32K context for full-module analysis.

## OpenCode

The OpenCode adapter invokes the `opencode` CLI as a subprocess. The system prompt is written as `.opencode/agents/gaze-headless.md` in a temporary directory, because OpenCode reads agent definitions from that path. That agent denies every tool and runs as a primary agent, so the model can only answer.

### Prerequisites

- Install OpenCode: `npm install -g opencode-ai`
- Set the `OPENCODE_API_KEY` environment variable (or configure OpenCode's default provider)

### Usage

```bash
# Use OpenCode's configured default model
gaze report ./... --ai=opencode

# Specify a model
gaze report ./... --ai=opencode --model=opencode/claude-sonnet-4-6
```

### CI Configuration

```yaml
- name: Install OpenCode
  run: npm install -g opencode-ai@latest

- name: Gaze quality report
  run: |
    gaze report ./... \
      --ai=opencode \
      --model=opencode/claude-sonnet-4-6 \
      --coverprofile=coverage.out
  env:
    OPENCODE_API_KEY: ${{ secrets.OPENCODE_API_KEY }}
```

### How It Works Internally

1. A temp directory is created (`gaze-opencode-*/`)
2. `.opencode/agents/gaze-headless.md` is written inside it, with frontmatter that sets `mode: primary` and denies every tool, followed by the embedded report-mode prompt
3. OpenCode is invoked: `opencode run --dir <tmpDir> --agent gaze-headless --format json [-m <name>] ""`
4. The JSON payload is piped to stdin
5. Only the final assistant text is extracted from the JSON event stream and validated against the answer contract
6. Temp directory is cleaned up

## Common Options

### `--model`

Override the default model for any adapter:

```bash
gaze report ./... --ai=claude --model=claude-sonnet-4-20250514
gaze report ./... --ai=gemini --model=gemini-2.5-pro
gaze report ./... --ai=ollama --model=llama3.2        # Required for Ollama
gaze report ./... --ai=opencode --model=opencode/claude-sonnet-4-6
```

When omitted, Claude, Gemini, and OpenCode use their CLI's configured default model. Ollama requires `--model` to be specified explicitly.

### `--ai-timeout`

Set the maximum time to wait for the AI adapter to respond. Default: 10 minutes.

```bash
gaze report ./... --ai=claude --ai-timeout=5m
gaze report ./... --ai=ollama --model=llama3.2 --ai-timeout=20m
```

The timeout is applied to the subprocess execution or HTTP request context. If the adapter does not respond within the timeout, Gaze cancels the request and returns an error.

### `--format=markdown`

Render the report without any AI adapter:

```bash
gaze report ./... --format=markdown
gaze report ./... --format=markdown --sections=crap,quality
```

`markdown` is deterministic: Gaze renders every section, grade, and recommendation itself, with no AI slots and no API key. Use it for a complete human-readable report in CI when no AI provider is available, or when you want byte-identical output for the same analysis data. `--sections` limits which analyses run and render (`crap`, `quality`, `classification`, `health`; default all).

### `--format=json`

Skip the AI formatting step entirely and output the raw JSON payload:

```bash
gaze report ./... --format=json > report.json
```

This is useful for CI threshold checks where you don't need a human-readable report, or for piping the data to your own formatting tool. No AI adapter or API key is required.

## Answer Validation

Every adapter's output passes the same validator before Gaze renders it, so a malformed or off-contract answer can never become the report. The answer must be exactly one JSON object:

```json
{"summary": "Plain-prose narrative.", "recommendations": ["First action.", "Second action."]}
```

- `summary` is 1–1,200 characters of plain English prose with no headings, lists, tables, links, or HTML
- `recommendations` holds 1–5 single-line items of at most 400 characters each
- non-Latin letters may not exceed 5% of the prose

A rejected answer fails the run with a non-zero exit, and the error names the rule that rejected it (for example `summary_length` or `non_english`). The deterministic report is still written to stdout and to the step summary, with a `> ⚠️ AI narrative rejected:` callout in place of the AI slots. An adapter failure renders `> ⚠️ AI narrative unavailable:` instead. Gaze never echoes the raw adapter output into the report, and a run never passes on narration alone.

## GitHub Step Summary

When the `$GITHUB_STEP_SUMMARY` environment variable is set (as it is automatically in GitHub Actions), the formatted report is appended to the workflow step summary. This makes the report visible directly in the GitHub Actions UI.

The step summary write is non-fatal — if it fails, Gaze prints a warning to stderr and continues. The report is still written to stdout.

## Pre-Flight Validation

The Claude, Gemini, and OpenCode adapters implement the `AdapterValidator` interface, which checks that the required CLI binary is on `PATH` before the analysis pipeline runs. This gives users an immediate error message rather than failing after several minutes of analysis work.

```text
Error: claude not found on PATH (FR-012): exec: "claude": executable file not found in $PATH
```

The Ollama adapter does not perform pre-flight validation because it uses HTTP rather than a subprocess.

## Troubleshooting

### "returned empty output" or a rejected answer

The AI adapter ran successfully (exit code 0) but produced no output, or its output failed the answer contract. The error names the failing validation rule (`empty_output`, `not_json`, `summary_length`, `non_english`, and others). Common causes:

- **Invalid API key** — the CLI may write the error to stderr while exiting 0. Gaze includes truncated stderr in the error message to help diagnose this. Sensitive environment-variable values in that stderr are replaced with `[REDACTED:<VAR>]` markers before the snippet is included (see [Secret hygiene and stderr redaction](ci-integration.md#secret-hygiene-and-stderr-redaction)).
- **Model not found** — verify the model name is correct for your adapter
- **Rate limiting** — wait and retry
- **Narration instead of JSON** — the model wrote prose around the answer or tried to use tools; the report-mode prompt forbids both, and the OpenCode adapter fails closed when a tool completes

### "not found on PATH"

The required CLI binary is not installed or not in your `$PATH`. Install the binary for your chosen adapter (see Prerequisites above).

### Large payloads

Full-module analysis (`./...`) on large codebases produces large JSON payloads. If the AI model truncates or refuses the input:

- Analyze specific packages instead of `./...`
- Use a model with a larger context window
- Use `--format=json` and process the data with your own tooling
