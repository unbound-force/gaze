# Validate Workflow Floor

## Why

`gaze init --ci` fails closed unless the embedding gaze binary's version (or the
explicit `--gaze-version` override) is at or above the `minWorkflowTag` floor
(`v2.3.0` in `internal/scaffold/scaffold.go`), because the scaffolded workflow's
`uses:` pin must reference a release tag that actually ships
`.github/workflows/reusable_gaze_ai_report.yml`.

The release workflow, however, has no symmetric check. Nothing prevents
maintainers from cutting a release tag *below* the floor constant (a release
whose own binary would reject its own `--ci` validation), or from bumping the
floor constant to a tag that was never released. The v2.3.0 release that
activated this coupling shipped before any gate existed; every subsequent
release should verify the tag/floor contract automatically.

This change adds a release-time validation gate, mirroring the existing
`validate-module-major` job and script, so that the release workflow and the
`--ci` floor constant cannot drift out of sync silently.

## What Changes

### Release workflow gate

- New `.github/scripts/validate-workflow-floor.sh` — a deterministic, offline
  validator invoked as `validate-workflow-floor.sh -- <release-tag> [repo-root]`.
  It extracts `minWorkflowTag` from `internal/scaffold/scaffold.go` and checks
  the requested release tag against it (below-floor fails; at-floor passes;
  above-floor requires the floor tag to exist locally and to contain the
  reusable workflow file).
- New `validate-workflow-floor` job in `.github/workflows/release.yml`, a
  sibling of `validate-module-major`: same pinned checkout (with
  `fetch-depth: 0` so existing tags are available locally), `contents: read`
  permission, 5-minute timeout. The `preflight` job now
  `needs: [validate-module-major, validate-workflow-floor]`.
- The gate is unskippable: no new release skip flag is introduced, matching the
  `validate-module-major` precedent (the existing `skip_*` inputs bypass
  org-infra preflight checks only).

### Test coverage

- New `.github/scripts/test-validate-workflow-floor.sh`, wired into
  `.github/workflows/test.yml` (Go 1.25 matrix leg) beside the existing
  module-major test script. It asserts repository invariants (floor constant
  present and parseable, reusable workflow present in the tree) and runs a
  synthetic-repo matrix covering skip, pass, and fail paths.

### Documentation

- `.github/workflows/release.yml` header comment and `AGENTS.md` CI/CD section
  updated to describe the new gate.

### Out of scope

- Any change to the `--ci` flag's runtime validation semantics, the floor
  constant's value, or the scaffolded workflow template.
- Retroactive validation of the v2.3.0 release itself (the gate lands after
  that release; its above-floor branch validates v2.3.0 on every later
  release).
- New Go code or protocol changes.

## Capabilities

### New Capabilities

- `release-workflow-floor-validation`: release-time consistency check between
  the requested release tag and the `minWorkflowTag` floor embedded in the
  scaffold package.

### Modified Capabilities

- None.

### Removed Capabilities

- None.

## Impact

- **Internal surfaces**: `.github/workflows/release.yml` (new job + preflight
  dependency), `.github/workflows/test.yml` (new test step),
  `internal/scaffold/scaffold.go` (read-only dependency on the existing
  `minWorkflowTag` constant — no code change).
- **Users**: no behavioral change to any released binary. Release operators
  gain an automated guard; a release request with a below-floor tag or an
  unreleased floor now fails before preflight creates the tag.
- **Risks**: false rejection if the floor constant moves to a different file
  without updating the validator. Mitigated by the test script's
  repository-invariant assertions, which fail CI the moment the constant is
  no longer where the validator expects it.

## Constitution Alignment

### I. Accuracy

**Assessment**: PASS. The validator derives the floor from the same source
constant the binary embeds, not a duplicated value, so its verdict matches the
`--ci` runtime behavior by construction. The above-floor existence check
verifies the actual git tag contents, not an assumption.

### II. Minimal Assumptions

**Assessment**: PASS. The gate runs entirely offline against the local clone
(`fetch-depth: 0` provides tags); it makes no network or GitHub API calls and
assumes nothing about the host beyond bash and git. Legacy trees without the
floor constant are explicitly skipped rather than assumed invalid.

### III. Actionable Output

**Assessment**: PASS. Every failure mode emits a specific, actionable message
(violated rule, offending tag, offending constant, remediation hint), following
the `::error::`/`::notice::` GitHub workflow-command conventions used by the
existing module-major validator.

### IV. Testability

**Assessment**: PASS. The validator is a pure CLI script exercised by a
self-contained test script running a synthetic-repo matrix in CI, covering
every branch (skip, pass, each failure mode). No test depends on network
access or a real release.
