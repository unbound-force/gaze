# Release Install Canary

## Why

`dogfood-ai-report-from-source` (PR #317) makes the `gaze-ai-report` job in
`.github/workflows/test.yml` build gaze from the checkout
(`build-from-source: true`). As a consequence, the
`Install gaze (release)` step of
`.github/workflows/reusable_gaze_ai_report.yml` — the path every consumer
takes — no longer runs anywhere in gaze's own CI. That change's design
accepted the gap; a review advisory on #317 suggested a canary to close it.

If the consumer install path breaks (the module proxy cannot resolve the
release, the `/v2` module path drifts, the release's `go.mod` requires a
newer toolchain than the reusable workflow's `go-version` default, a
dependency disappears), the first signal today is a failed consumer run.

## What Changes

- New workflow `.github/workflows/install-canary.yml`, triggered daily by
  `schedule` and on demand by `workflow_dispatch`. It resolves the latest
  GitHub release tag, runs the same commands as `Install gaze (release)`
  (`go install github.com/unbound-force/gaze/v2/cmd/gaze@<tag>` then
  `gaze --version`) on the reusable workflow's default Go toolchain (`1.25`),
  and verifies from the binary's build info that the installed module
  version equals the tag.
- `AGENTS.md`: CI/CD list entry and Recent Changes entry.

### Out of scope

- Running the full reusable AI-report workflow in release mode (needs a
  coverage artifact and the `opencode-config` secret; the install step is the
  only part the dogfood job no longer exercises).
- Triggering or changing the release pipeline.
- Notifications beyond GitHub's default scheduled-workflow failure email.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `ci-workflow`: the consumer install path of the reusable AI-report
  workflow is exercised against the latest release on a schedule.

### Removed Capabilities

- None.

## Impact

- **Internal surfaces**: `.github/workflows/install-canary.yml` (new),
  `AGENTS.md`.
- **Consumers**: none.
- **Cost**: one short `ubuntu-latest` job per day (setup-go + one
  `go install`), `contents: read` only, no secrets beyond `github.token`.
- **No website issue**: CI/CD pipeline changes are exempt.

## Constitution Alignment

### I. Accuracy

**Assessment**: PASS. The canary runs the consumer commands verbatim and
checks the module version the proxy actually served, so a pass means the
published release installs as consumers install it.

### II. Minimal Assumptions

**Assessment**: PASS. No checkout, no secrets, no cache; the job sees the
same environment a consumer's reusable-workflow run sees at the install step.

### III. Actionable Output

**Assessment**: PASS. Failures name the step (resolve / install / version
check), and the version mismatch `::error::` prints both the expected tag
and the installed version.

### IV. Testability

**Assessment**: PASS. `actionlint` and `yamllint` (MegaLinter's config)
validate the workflow; `workflow_dispatch` allows an on-demand run after
merge.
