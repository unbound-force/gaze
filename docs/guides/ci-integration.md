# CI Integration

Integrate Gaze into your CI pipeline to enforce code quality thresholds on every push and pull request. This guide covers GitHub Actions — the same patterns apply to any CI system that runs shell commands.

## How It Works

Gaze's CI integration follows a three-step pattern:

1. **Run tests** with `-coverprofile` to generate a coverage profile
2. **Run [`gaze report`](../reference/cli/report.md)** with `--coverprofile` to reuse that profile (avoiding a second test run)
3. **Enforce thresholds** with `--max-crapload`, `--max-gaze-crapload`, and `--min-contract-coverage`

When any threshold is exceeded, Gaze exits non-zero and prints a one-line summary to stderr:

```text
CRAPload: 12/10 (FAIL) | GazeCRAPload: 3/5 (PASS) | ContractCoverage: 45.2%/60.0% (FAIL)
```

Without threshold flags, Gaze always exits 0 (report-only mode).

## Minimal Example

This workflow runs tests, generates a coverage profile, and enforces quality thresholds — no AI adapter required:

```yaml
name: Quality Gate

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  quality:
    runs-on: ubuntu-latest
    timeout-minutes: 15
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: "1.25"

      - name: Build Gaze
        run: go build -o gaze ./cmd/gaze

      - name: Test with coverage
        run: go test -race -count=1 -short -coverprofile=coverage.out ./...

      - name: Gaze threshold check
        run: |
          ./gaze report ./... \
            --format=json \
            --coverprofile=coverage.out \
            --max-crapload=10 \
            --max-gaze-crapload=5 \
            --min-contract-coverage=60
```

Key points:

- **`--format=json`** skips the AI formatting step entirely — no API keys needed
- **`--coverprofile=coverage.out`** reuses the profile from the test step, so tests run only once
- **Threshold flags** cause a non-zero exit when limits are exceeded, failing the CI step

## Full Example with AI Reports

This workflow adds an AI-powered quality report that appears in the GitHub Actions Step Summary. This is the manual path; for less maintenance, prefer the [Reusable Workflow](#reusable-workflow) below, which centralizes the opencode installation and provider configuration:

```yaml
name: Test & Quality

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    timeout-minutes: 30
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: "1.25"

      - name: Build
        run: go build -o gaze ./cmd/gaze

      - name: Test
        run: go test -race -count=1 -short -timeout 15m -coverprofile=coverage.out ./...

      # PR-safe threshold check (no secrets required)
      - name: Gaze threshold check
        run: |
          ./gaze report ./... \
            --format=json \
            --coverprofile=coverage.out \
            --max-crapload=38 \
            --max-gaze-crapload=6 \
            --min-contract-coverage=8 \
            > /dev/null

      # AI report — push-only in this example; the reusable workflow
      # also runs it on same-repo PRs
      - name: Install OpenCode
        if: github.event_name == 'push'
        run: npm install -g opencode-ai@1.19.0

      - name: Gaze quality report
        if: github.event_name == 'push'
        env:
          OPENCODE_API_KEY: ${{ secrets.OPENCODE_API_KEY }}
        run: |
          ./gaze report ./... \
            --ai=opencode \
            --model=opencode/claude-sonnet-4-6 \
            --coverprofile=coverage.out \
            --max-crapload=38 \
            --max-gaze-crapload=6 \
            --min-contract-coverage=8
```

This pattern splits the quality gate into two steps:

1. **Threshold check** (`--format=json`) — runs on every PR, no secrets needed, fails the build if thresholds are exceeded
2. **AI report** (`--ai=opencode`) — runs on push to main only in this example (gated on `github.event_name == 'push'`); the [Reusable Workflow](#reusable-workflow) also runs it on same-repo PRs, where org secrets resolve. Requires an API key secret. The adapter supplies only the summary and recommendations, which Gaze validates and renders into the deterministic report, producing a human-readable Step Summary

## Deterministic Reports and AI Judgment

Gaze renders every report section itself: the tables, metrics, grades, and
deterministic recommendations come from the analysis data, so identical data
always produces identical output. In text mode the AI adapter adds only two
things — a plain-prose summary and a prioritized recommendation list — as strict
JSON that Gaze validates before rendering.

That gives CI two useful properties:

- **No secret needed for a full report.** `--format=markdown` produces the
  complete report, including the Overall Health scorecard, without installing or
  calling an AI provider:

  ```yaml
  - name: Gaze report
    run: ./gaze report ./... --format=markdown --coverprofile=coverage.out
  ```

- **Invalid judgment fails loudly.** Gaze validates the adapter's answer against
  a fixed contract: one JSON object with an English summary of 1–1,200
  characters and 1–5 recommendations of at most 400 characters each. A rejected
  answer exits non-zero even when every threshold passes, and the deterministic
  report plus a `> ⚠️ AI narrative rejected:` callout are still written to
  stdout and the step summary. A run never passes on narration alone.

`gaze report` always uses its built-in report-mode prompt. A local
`.opencode/agents/gaze-reporter.md` is ignored — Gaze prints a one-line notice
to stderr — so a project-customized agent cannot change CI output.

The [Reusable Workflow](#reusable-workflow) has this behavior from v2.4.0: its
`gaze-version` default is `v2.4.0`, the first release that contains it. A
caller that pins `uses:` below `@v2.4.0`, or passes an older `gaze-version`,
installs a binary whose adapter output is not validated. (Gaze's own CI
dogfoods the reusable workflow with `build-from-source: true`, so it always
reports with the commit under test.)

## Reusable Workflow

Gaze publishes a provider-agnostic reusable workflow at
[`.github/workflows/reusable_gaze_ai_report.yml`](https://github.com/unbound-force/gaze/blob/main/.github/workflows/reusable_gaze_ai_report.yml)
so consumers don't have to duplicate the opencode installation and provider
configuration machinery. The caller supplies a coverage profile artifact (plus
optional model and thresholds); the entire opencode configuration — provider,
model, permissions, and API key — arrives through a single secret, so the
consumer controls the whole configuration. The workflow installs pinned
toolchains, runs `gaze report --ai=opencode`, and enforces the supplied
thresholds.

### Inputs and secrets

| Input | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `model` | string | no | — | Optional model override; when unset, opencode uses the config's top-level `model`. |
| `coverage-artifact` | string | yes | — | Artifact name (uploaded earlier in the same run) containing the coverprofile. |
| `coverprofile` | string | no | `coverage.out` | Path to the coverprofile after artifact extraction, relative to the workspace root; must match the artifact's internal layout. |
| `max-crapload` | number | no | — | Fail when the CRAPload exceeds this value. |
| `max-gaze-crapload` | number | no | — | Fail when the GazeCRAPload exceeds this value. |
| `min-contract-coverage` | number | no | — | Fail when average contract coverage falls below this percentage. |
| `opencode-version` | string | no | `1.19.0` | opencode-ai npm version to install. |
| `gaze-version` | string | no | `v2.4.0` | gaze release tag to install. Ignored when `build-from-source` is `true`. |
| `build-from-source` | boolean | no | `false` | Build gaze from the checked-out tree instead of installing `gaze-version`. Intended only for the gaze repository's own dogfood job; fails unless the checked-out module is `github.com/unbound-force/gaze/v2`. |
| `go-version` | string | no | `1.25` | Go toolchain version for the report job. |

Secret `opencode-config` (optional): the consumer's entire strict-JSON
opencode configuration — provider, model, permissions, and the literal API
key. The workflow exports it verbatim to the report step as
`OPENCODE_CONFIG_CONTENT`. Because the config carries key material, keep it
in an org- or repo-level secret (never in a workflow file) and include
permission denials so the report step cannot edit files or reach the network
beyond the provider call:

```json
{
  "model": "enmaas/rits/zai-org/glm-5-3",
  "permission": { "edit": "deny", "bash": "deny", "webfetch": "deny", "websearch": "deny" },
  "provider": {
    "enmaas": {
      "npm": "@ai-sdk/openai-compatible",
      "name": "EnMaaS",
      "options": {
        "baseURL": "https://api.enmaas.devshift.net/v1",
        "apiKey": "<LITERAL API KEY>"
      },
      "models": {
        "rits/zai-org/glm-5-3": {
          "name": "GLM 5.3",
          "limit": { "context": 262144, "output": 65536 }
        }
      }
    }
  }
}
```

The value must be strict JSON (no comments). The `apiKey` is the literal key:
the config is injected as one unit, so there is no separate env-var secret to
resolve. Any custom opencode provider works; the workflow does not hardcode
one. The trade-off is that the config leaves code review — it is a write-only
secret controlled by the org or repo administrator.

When `opencode-config` is unavailable (for example on fork PRs, where secrets
are never exposed), the workflow downloads the artifact and installs gaze,
then skips the opencode install and report with a notice — so every run that
reaches the job still validates the artifact handoff, including that the
coverprofile file exists inside the downloaded artifact. On non-fork runs
where the secret is missing, the skip step emits a warning annotation
pointing at the secret name and scope.

### Secret hygiene and stderr redaction

The `opencode-config` secret embeds the literal API key inside a larger JSON
document. GitHub's automatic masking hides the full secret value in job logs,
but a bare key fragment echoed on its own — for example, an AI CLI printing a
config parse error — would appear unmasked. Two defense-in-depth layers
address this: gaze scrubs adapter stderr before surfacing it, and you can
register the bare key for platform-side masking in the calling job.

When `gaze report` surfaces the AI CLI's stderr in an error message, gaze
first replaces every sensitive inherited env-var value with a
`[REDACTED:<VAR_NAME>]` marker — for example,
`[REDACTED:OPENCODE_CONFIG_CONTENT]`. Both surfacing paths are covered:
adapter-failure errors and empty-output errors each embed up to 512 bytes of
child stderr. A value is scrubbed when its variable name contains `KEY`,
`TOKEN`, `SECRET`, or `CONFIG_CONTENT` (case-insensitive), or when the value
is at least 32 bytes long regardless of name — so the whole
`OPENCODE_CONFIG_CONTENT` document is caught by both rules. The marker names
the variable — env-var names are not secret — keeping the error diagnosable.
The scrub runs on the full stderr buffer before the 512-byte truncation, so a
value straddling the truncation boundary cannot leak a prefix.

The scrub is verbatim-value replacement, so it has known limits. Values
shorter than 4 bytes are never replaced — replacing them would garble the
surrounding error text — and oddly-named secrets below the 32-byte threshold
are not caught. The bare `apiKey` inside the config is not an env-var value
by itself, so an echo of that fragment alone evades the scrub, as does a
transformed value — a child process that base64-encodes or rewraps the
secret. The second-secret pattern below closes the fragment gap at the
platform layer; transformed values remain a residual risk.

Register the bare API key as a second secret — for example, `OPENCODE_API_KEY`
— and reference it in an unused env var on any step of the calling job.
GitHub then masks verbatim occurrences of the bare key anywhere in that job's
logs, covering fragment echoes the scrub cannot see:

```yaml
- name: Test with coverage
  run: go test -race -count=1 -short -coverprofile=coverage.out ./...
  env:
    # Unused by the step: referencing the bare key registers it
    # with GitHub's log masking for the job.
    OPENCODE_API_KEY_MASK: ${{ secrets.OPENCODE_API_KEY }}
```

Prefer a single-line strict-JSON value for the `opencode-config` secret.
GitHub's multiline secret masking is unreliable, so a pretty-printed config
echoed into a log may appear unmasked; gaze's byte-exact scrub is unaffected
by line breaks, so the single-line shape costs nothing. Minify the JSON
before storing it — `jq -c .` emits the single-line form. The pretty-printed
example above is for readability; the actual secret value you store should be
the same JSON minified to one line. See
[Troubleshooting](#troubleshooting) for validating the config with `jq`
before storing it.

### Scaffolded install: `gaze init --ci`

The preferred adoption path is to let gaze generate the caller workflow for
you:

```bash
gaze init --ci
```

In a Go module rooted at the repository root, this writes
`.github/workflows/gaze-ai-report.yml` — a self-contained caller that owns
its own `test` job (checkout → setup-go → `go test -race -count=1 -short
-coverprofile=coverage.out ./...` → upload-artifact) and calls the reusable
workflow with the coverage artifact and the `OPENCODE_CONFIG_CONTENT`
secret. The `uses:` pin is resolved from the running gaze binary's version
and validated (`>= v2.3.0`); dev builds fail closed rather than writing an
unusable pin, and `--gaze-version` overrides the pin with the same
validation. The `setup-go` version is read from the module's `go`
directive. When no `go.mod` exists at the repository root, CI scaffolding
is skipped with a warning and only the `.opencode/` scaffolding runs.

Two ownership notes:

- The scaffolded workflow is **report-only** until you uncomment its
  threshold inputs — it exits 0 and appends the AI report to the step
  summary without enforcing anything. Choose values with the guidance in
  [Threshold Flags](#threshold-flags), then uncomment the lines.
- The file is **user-owned** after first write: re-running `gaze init --ci`
  skips it (threshold edits survive); only `--force` regenerates it.

See [`gaze init`](../reference/cli/init.md) for the full flag reference.
The hand-written example below shows what the scaffolded file contains,
for consumers who prefer to own the YAML from the start or need a shape
the scaffold does not emit.

### Consumer example

```yaml
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.25"
      - name: Test with coverage
        run: go test -race -count=1 -short -coverprofile=coverage.out ./...
      - name: Upload coverage
        uses: actions/upload-artifact@v4
        with:
          name: coverage-out
          path: coverage.out

  gaze-ai-report:
    needs: test
    permissions:
      contents: read
    uses: unbound-force/gaze/.github/workflows/reusable_gaze_ai_report.yml@v2.4.0
    with:
      coverage-artifact: coverage-out
      gaze-version: v2.4.0
      max-crapload: 38
      max-gaze-crapload: 6
      min-contract-coverage: 8
    secrets:
      opencode-config: ${{ secrets.OPENCODE_CONFIG_CONTENT }}
```

Pin `uses:` to a release tag that actually contains the workflow — it
first ships in `v2.3.0`; older tags predate the file and fail to
resolve. Pin the latest release tag when adopting, and set `gaze-version`
to the same tag so the installed binary stays explicit when you bump the pin
(`gaze init --ci` scaffolds it this way).

The coverage profile must come from the same run and commit: both jobs
check out the same SHA, so the profile's import paths map onto identical
source files. See [Troubleshooting](#troubleshooting) for the limits of
profile portability.

### Concurrency

The reusable workflow defines no concurrency group of its own — a called
workflow joins the caller's run, so the supersede/cancel policy is
caller-owned. Callers SHOULD define a workflow-level `concurrency:` group
so rapid pushes don't stack AI-report runs (each report is a paid LLM
call). The pattern gaze's own CI uses:

```yaml
concurrency:
  group: ${{ github.ref }}-${{ github.workflow }}
  cancel-in-progress: ${{ github.ref != 'refs/heads/main' }}
```

### Requirements and limitations

- The calling job must grant at least `contents: read` — a called
  workflow can only downgrade permissions, never elevate them.
- The workflow analyzes `./...` from the repository root: the caller
  must be a Go module rooted at the repo root.
- Providers that authenticate via the GitHub token (for example GitHub
  Models) cannot work — the report step unsets `GITHUB_TOKEN` and
  `GH_TOKEN` before invoking the AI CLI.
- Supply-chain-sensitive consumers may pin `uses:` to the release tag's
  commit SHA instead of the mutable tag.

## Coverage Profile Reuse

The `--coverprofile` flag is the key to avoiding double test runs. Without it, [`gaze report`](../reference/cli/report.md) spawns its own `go test -coverprofile` internally — meaning your tests run twice per CI job.

```bash
# Step 1: Run tests once with coverage
go test -race -count=1 -coverprofile=coverage.out ./...

# Step 2: Pass the profile to Gaze
gaze report ./... --coverprofile=coverage.out --format=json
```

The coverage data used for [CRAP](../concepts/scoring.md) scores is the same high-quality profile produced by the race-detecting test run. No data is lost.

If you omit `--coverprofile`, Gaze generates its own profile by running `go test -short -coverprofile=<tmpfile> ./...` internally. This is convenient for local use but wasteful in CI where you've already run tests.

## Threshold Flags

Three threshold flags control CI enforcement:

| Flag | Type | Description |
|------|------|-------------|
| `--max-crapload` | int | Maximum number of functions at or above the [CRAP threshold](../reference/glossary.md) (default threshold: 15). Zero is a live threshold — it means "no crappy functions allowed." |
| `--max-gaze-crapload` | int | Maximum number of functions at or above the [GazeCRAP](../reference/glossary.md) threshold. Zero is a live threshold. |
| `--min-contract-coverage` | int | Minimum average [contract coverage](../reference/glossary.md) percentage across all analyzed functions. |

When a threshold is exceeded, Gaze:

1. Prints a summary line to stderr showing each threshold's pass/fail status
2. Exits with a non-zero exit code, failing the CI step

When no threshold flags are provided, Gaze operates in report-only mode and always exits 0.

### Choosing Thresholds

Start with permissive thresholds and tighten over time:

```yaml
# Week 1: Establish baseline
--max-crapload=50 --max-gaze-crapload=20 --min-contract-coverage=5

# Month 1: Prevent regression
--max-crapload=40 --max-gaze-crapload=15 --min-contract-coverage=10

# Steady state: Ratchet toward quality
--max-crapload=20 --max-gaze-crapload=5 --min-contract-coverage=30
```

The goal is to prevent regression first, then gradually improve. See [Improving Scores](improving-scores.md) for strategies to reduce [CRAPload](../reference/glossary.md) and increase contract coverage.

## GitHub Step Summary

When the `$GITHUB_STEP_SUMMARY` environment variable is set (as it is automatically in GitHub Actions), [`gaze report`](../reference/cli/report.md) appends the formatted AI report to the workflow step summary. This makes the report visible directly in the GitHub Actions UI without opening logs.

The step summary write is non-fatal — if it fails (e.g., permissions issue), Gaze prints a warning to stderr and exits 0. The report is still written to stdout.

The step summary uses symlink-safe writes (`O_NOFOLLOW`) to prevent symlink attacks in shared runner environments.

## Using [`gaze crap`](../reference/cli/crap.md) Instead of [`gaze report`](../reference/cli/report.md)

If you only need CRAP scores without the full analysis pipeline (no quality assessment, no classification, no docscan), you can use [`gaze crap`](../reference/cli/crap.md) directly:

```yaml
- name: CRAP check
  run: |
    gaze crap ./... \
      --coverprofile=coverage.out \
      --max-crapload=10 \
      --max-gaze-crapload=5
```

The [`gaze crap`](../reference/cli/crap.md) command supports the same `--max-crapload` and `--max-gaze-crapload` threshold flags. It does not support `--min-contract-coverage` (that requires the full quality pipeline via [`gaze report`](../reference/cli/report.md)).

## Baseline Comparison

Baseline comparison detects per-function CRAP and GazeCRAP regressions by comparing the current analysis against a saved baseline. When `.gaze/baseline.json` exists and is non-empty, [`gaze crap`](../reference/cli/crap.md) auto-detects it and activates comparison mode. No flags required.

### Creating a Baseline

```bash
mkdir -p .gaze
go test -coverprofile=coverage.out -count=1 ./...
gaze crap --format=json --coverprofile=coverage.out ./... > .gaze/baseline.json
git add .gaze/baseline.json && git commit -m "chore: add CRAP baseline"
```

Commit the baseline to version control so CI can compare every PR against it.

### How Comparison Works

Each function is matched between baseline and current results by its `file:function` key. For each matched function, CRAP and GazeCRAP deltas are computed:

| Status | Condition |
|--------|-----------|
| `regression` | CRAP or GazeCRAP delta exceeds epsilon (default 0.5) |
| `improvement` | CRAP or GazeCRAP delta exceeds negative epsilon |
| `unchanged` | Delta within epsilon tolerance |
| `new` | Function not in baseline, CRAP below threshold |
| `new_violation` | Function not in baseline, CRAP above threshold (default 30) |
| `removed` | Function in baseline but not in current results |

The comparison passes when there are zero regressions and zero new-function violations. `gaze crap` exits with code 1 when the comparison fails -- independently of any threshold flags (`--max-crapload`, etc.).

### CI Workflow with Baseline

```yaml
- name: Test with coverage
  run: go test -race -count=1 -coverprofile=coverage.out ./...

- name: Check for regressions
  run: gaze crap --coverprofile=coverage.out ./...
```

That's it. If `.gaze/baseline.json` is committed, the comparison runs automatically. If the file doesn't exist, `gaze crap` behaves as normal.

### Overriding the Baseline Path

Use `--baseline` to point to a baseline file at a non-default location:

```bash
gaze crap --baseline path/to/other-baseline.json --coverprofile=coverage.out ./...
```

When `--baseline` is specified explicitly, a missing or empty file is an error.

### Tuning Comparison Sensitivity

Add a `baseline` section to `.gaze.yaml`:

```yaml
baseline:
  epsilon: 0.5               # score change tolerance (default)
  new_function_threshold: 30  # max CRAP for new functions (default)
```

These defaults are production-validated. Increase epsilon if platform-induced score jitter causes false regressions. Lower the new-function threshold to enforce stricter standards on new code.

### Refreshing the Baseline

The baseline is a snapshot that drifts over time. Refresh it periodically:

```bash
gaze crap --format=json --coverprofile=coverage.out ./... > .gaze/baseline.json
```

A good cadence is after each release or when you intentionally accept score changes. Functions added to main after the baseline was created appear as "new" until the baseline is refreshed -- this is informational, not a failure (unless they exceed the new-function threshold).

## Troubleshooting

### Tests run twice

You're not using `--coverprofile`. Add `-coverprofile=coverage.out` to your test step and `--coverprofile=coverage.out` to your [`gaze report`](../reference/cli/report.md) step.

### Threshold check passes locally but fails in CI

Coverage profiles are not portable across machines or runs. Generate the profile from the same commit you analyze, and don't cache profiles across runs. Uploading a profile for a later job in the same run — as the [reusable workflow](#reusable-workflow) does — is safe: both jobs check out the same commit, so the profile's import paths map to identical source files.

### AI report step fails with "empty output"

The AI adapter returned no content. Check that:

- The API key secret is configured correctly
- The AI CLI binary is installed (the `Install OpenCode` step ran)
- The model name is valid for your adapter
- The `opencode-config` secret (reusable workflow) parses as strict JSON — a malformed config surfaces as "empty output"; validate it locally with `jq` before storing it

See [AI Reports](ai-reports.md) for adapter-specific setup.
