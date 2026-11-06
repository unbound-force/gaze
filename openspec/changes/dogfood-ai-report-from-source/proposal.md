# Dogfood AI Report From Source

## Why

The `gaze-ai-report` job in `.github/workflows/test.yml` calls the reusable
workflow without a `gaze-version`, so it installs the input's default release
(`v2.2.0`) rather than the commit under test. Test run 41698377720 (push of
`fe28ce6` to `main`) shows the consequence:

- `Install gaze` logged `GAZE_RELEASE: v2.2.0` and
  `go: downloading github.com/unbound-force/gaze/v2 v2.2.0`.
- The v2.2.0 opencode adapter (`--format default`, tool-enabled agent) wrote a
  single line of agent narration ("The `gaze` binary is on PATH … Let me
  inspect the repo's opencode configuration …") to the step summary instead
  of a quality report, and the job passed.

This is the same failure spec 040 (deterministic AI report rendering) was
written to fix (run 41309423609). Spec 040 is on `main` but in no release
(neither v2.2.0 nor v2.3.0 contains it), so the dogfood job has never
exercised it. A dogfood job that tests a pinned old release cannot catch
regressions in, or validate fixes to, the code being changed.

## What Changes

- New optional boolean input `build-from-source` (default `false`) on
  `.github/workflows/reusable_gaze_ai_report.yml`. When `true`, the workflow
  builds gaze from the checked-out tree (`go install ./cmd/gaze`) instead of
  installing the `gaze-version` release, after verifying that the checked-out
  module is `github.com/unbound-force/gaze/v2`; any other module fails the
  job with an actionable `::error::`.
- Both install paths print `gaze --version` so the job log records which
  binary produced the report; the source build stamps the checked-out commit
  into it.
- `.github/workflows/test.yml` `gaze-ai-report` job passes
  `build-from-source: true`, so every push and same-repo PR dogfoods the
  exact commit under test (the PR merge commit on `pull_request`).
- The `gaze-version` input description states it is ignored when
  `build-from-source` is `true`.

### Out of scope

- Bumping the `gaze-version` default. It must change in the same commit that
  is tagged as the release containing spec 040 (callers pinned to `@vX.Y.Z`
  receive the default baked into that tag), and the release is cut only after
  this change produces a correct report on `main`. That is a separate,
  follow-up change.
- Any change to the scaffolded consumer workflow (`gaze init --ci`), which
  already passes `gaze-version` equal to its `uses:` pin.
- Go code changes.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `ci-workflow`: the reusable AI-report workflow can build gaze from the
  checked-out source, and the repository's own dogfood job uses it.

### Removed Capabilities

- None.

## Impact

- **Internal surfaces**: `.github/workflows/reusable_gaze_ai_report.yml`,
  `.github/workflows/test.yml`, `docs/guides/ci-integration.md`, `AGENTS.md`.
- **Consumers**: none by default — `build-from-source` defaults to `false`
  and the existing install path is unchanged. The input is documented as
  intended for the gaze repository itself; on any other repository it fails
  fast.
- **Behavior on main**: the dogfood job starts running spec 040 for the first
  time. If the configured model cannot satisfy the strict answer contract,
  the job now fails (spec 040's validation) instead of passing on narration.
  That is the intended signal and must be resolved before releasing.
- **Trust boundary**: on `pull_request`, the PR's own gaze binary runs with
  the `opencode-config` secret. This adds no new exposure: secrets reach only
  same-repo PRs (fork PRs skip the report), whose authors can already edit
  `test.yml` itself in the same PR.

## Constitution Alignment

### I. Accuracy

**Assessment**: PASS. The dogfood report is produced by the binary built from
the commit under test, so it reflects the code being changed rather than an
unrelated release. `gaze --version` in the log makes the provenance explicit.

### II. Minimal Assumptions

**Assessment**: PASS. Building from source needs only the checkout and Go
toolchain the job already sets up; no module proxy, checksum database, or
pseudo-version resolution is involved, so it works for ephemeral PR merge
commits. The module-path guard refuses to guess when the caller is not gaze.

### III. Actionable Output

**Assessment**: PASS. A non-gaze caller enabling the input receives an
`::error::` naming the expected and actual module paths and the remedy (use
`gaze-version`).

### IV. Testability

**Assessment**: PASS. The change is exercised on every push and same-repo PR
by the dogfood job itself; `actionlint` and `yamllint` validate the workflow
syntax and expressions locally and in CI.
