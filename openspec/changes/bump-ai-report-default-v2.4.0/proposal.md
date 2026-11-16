# Bump AI Report Default to v2.4.0

## Why

`.github/workflows/reusable_gaze_ai_report.yml` defaults `gaze-version` to
`v2.2.0`. A caller pinned to `uses: ...@v2.4.0` that omits `gaze-version`
therefore installs v2.2.0, whose opencode adapter (`--format default`,
tool-enabled agent) can publish agent narration as the report and still pass
(runs 41309423609 and 41698377720). Spec 040 (deterministic AI report
rendering) fixes this but has not shipped; v2.4.0 will be its first release.

A tag bakes in its own copy of the workflow, including the input default, so
the bump must land in the commit that is tagged `v2.4.0`. The precondition in
`dogfood-ai-report-from-source` (task 3.3) is met: run 43251427781 on `main`
built gaze from source (`dev-e53867eda330`) and published the full spec 040
report.

## What Changes

- `gaze-version` default: `v2.2.0` to `v2.4.0`, with a comment beside it
  recording that the default must equal the release tag that ships the file.
- `docs/guides/ci-integration.md`: inputs table default and the spec 040 note.
- New release gate `.github/scripts/validate-ai-report-default.sh`, run by
  a `validate-ai-report-default` job in `release.yml` before preflight. It
  fails the release unless the `gaze-version` default equals the release
  tag. Tests in `.github/scripts/test-validate-ai-report-default.sh` run in
  `test.yml`.
- `AGENTS.md`: Recent Changes entry.

### Out of scope

- Cutting the release (`release.yml`), done separately after merge.
- `test.yml`: the dogfood job uses `build-from-source: true` and ignores
  `gaze-version`.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `ci-workflow`: the reusable AI-report workflow's default gaze release.

### Removed Capabilities

- None.

## Impact

- **Consumers pinned to `@v2.4.0` or later that omit `gaze-version`**: run
  v2.4.0. If their model cannot satisfy spec 040's strict answer contract,
  the job now fails instead of passing on narration. That is spec 040's
  intended behavior.
- **Consumers pinned to older tags**: unchanged.
- **`gaze init --ci` scaffolds**: unchanged; they already pass
  `gaze-version` equal to the `uses:` pin.
- **Between merge and tag**: callers pinned to `@main` would install
  v2.4.0 before it exists and fail at `go install`. The release follows
  this merge directly, and `@main` pins are not a supported consumer path.

## Constitution Alignment

### I. Accuracy

**Assessment**: PASS. Default consumers get the release whose report output
is validated, not one that can publish narration as a passing report.

### II. Minimal Assumptions

**Assessment**: PASS. No new inputs or behavior; one default value.

### III. Actionable Output

**Assessment**: PASS. Spec 040 failures name the invalid answer instead of
silently publishing it.

### IV. Testability

**Assessment**: PASS. The keep-equal rule is enforced by an offline release
gate with its own tests in CI (the dogfood job cannot exercise the default,
since it builds from source). `actionlint` and `yamllint` validate the workflows; the
released behavior was exercised on `main` by the dogfood job (run
43251427781), and the scaffold goldens already expect `v2.4.0`.
