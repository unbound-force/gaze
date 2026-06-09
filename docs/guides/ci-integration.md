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

This workflow adds an AI-powered quality report that appears in the GitHub Actions Step Summary:

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
            --max-gaze-crapload=5 \
            --min-contract-coverage=8 \
            > /dev/null

      # AI report — push-only (requires API key secret)
      - name: Install OpenCode
        if: github.event_name == 'push'
        run: npm install -g opencode-ai@latest

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
            --max-gaze-crapload=5 \
            --min-contract-coverage=8
```

This pattern splits the quality gate into two steps:

1. **Threshold check** (`--format=json`) — runs on every PR, no secrets needed, fails the build if thresholds are exceeded
2. **AI report** (`--ai=opencode`) — runs only on push to main, requires an API key secret, produces a human-readable report in the GitHub Step Summary

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

Coverage profiles are not portable across machines. Always generate the coverage profile and run Gaze in the same CI job. Don't cache or upload coverage profiles between jobs.

### AI report step fails with "empty output"

The AI adapter returned no content. Check that:
- The API key secret is configured correctly
- The AI CLI binary is installed (the `Install OpenCode` step ran)
- The model name is valid for your adapter

See [AI Reports](ai-reports.md) for adapter-specific setup.
