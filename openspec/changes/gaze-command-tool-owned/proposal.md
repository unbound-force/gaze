## Why

`gaze init` treats `commands/gaze.md` as user-owned: written once on the
first run, skipped on every run after. Every other scaffolded file is
tool-owned and refreshed by overwrite-on-diff when the embedded asset
changes. The result is that `/gaze` — the flagship interactive command —
goes stale in every consumer project on every gaze release.

The evidence (issue #353 survey of 10 repos running gaze): every local
`commands/gaze.md` diff against the v2.4.0 embedded asset is
**undelivered upstream change**, not local customization — missing the
`<protect>` block from #221, the spec 040 rendered-report description,
and analyzer/language detection. Several copies carry duplicate version
markers (`dev` and `v1.8.0`), the exact defect class issue #279 fixed
for the write path but which persists forever in a file that is never
rewritten. The only delivery path today is `gaze init --force`, which
clobbers all user-owned files to update one tool asset.

Spec 040 already flipped `agents/gaze-reporter.md` to tool-owned while
deliberately leaving `commands/gaze.md` user-owned. The survey shows
that reservation preserves staleness without preserving customizations:
across 10 repos, zero local `/gaze` customizations existed and all drift
was upstream. The reservation's cost is now measurable; this change
removes it.

## What Changes

### Ownership Flip

`commands/gaze.md` is added to the `isToolOwned` exact-match list in
`internal/scaffold/scaffold.go`, making it tool-owned like
`commands/gaze-fix.md`: overwrite-on-diff without `--force`, reported as
`updated` in the init summary. The first `gaze init` after upgrading
gaze delivers the accumulated upstream changes to consumer projects and
— because tool-owned updates rewrite the file from the embedded asset
with a freshly inserted marker — heals the duplicate-marker copies found
in the survey.

### Custom `/gaze` Escape Hatch

A documented convention, not new code: users who want a customized
command name it `commands/gaze-local.md` (any path outside the nine
embedded assets). The scaffold's asset walk only ever writes embedded
paths, so such a file is structurally untouchable — even by
`--force`. OpenCode exposes `.opencode/commands/*.md` as slash commands
by filename, so `/gaze-local` works with zero gaze changes. The
nine-file embedded manifest is unchanged; no new asset is scaffolded.

### Update-Time Escape-Hatch Tip

When `gaze init` updates `commands/gaze.md` (not when it creates or
skips it), the summary prints a one-line tip naming
`commands/gaze-local.md` as the preservation path. The overwrite moment
is exactly when a customizing user needs to learn the hatch exists;
git recovers the prior version, and the tip teaches the permanent
opt-out.

### Documentation

The ownership table and ownership-model section in
`docs/guides/opencode-integration.md` and the ownership semantics in
`docs/reference/cli/init.md` are updated: all nine `.opencode/` assets
are tool-owned; the CI workflow remains the only user-owned scaffold
output; the escape hatch and the migration path for customized copies
are documented.

## Capabilities

### New Capabilities

- `scaffold-file-ownership` — the `gaze init` file ownership model: all
  nine embedded `.opencode/` assets are tool-owned with overwrite-on-diff
  semantics, the CI workflow is user-owned, non-embedded command paths
  (the `commands/gaze-local.md` escape hatch) are never touched, and
  updates to `commands/gaze.md` print escape-hatch guidance

### Modified Capabilities

- None

### Removed Capabilities

- None

## Impact

- **Modified files**: `internal/scaffold/scaffold.go` (one
  `isToolOwned` case constant, comment rewrite, tip in `printSummary`),
  `internal/scaffold/scaffold_test.go` (flipped expectations, new
  regression tests), `cmd/gaze/main_test.go` (init summary assertions,
  if gaze.md-specific), `docs/guides/opencode-integration.md`,
  `docs/reference/cli/init.md`, `AGENTS.md` (Recent Changes entry)
- **Unchanged**: embedded asset content (`.opencode/` and
  `internal/scaffold/assets/` byte-identical pairs, 9-file manifest,
  `TestEmbeddedAssetsMatchSource`), the CI workflow's user-owned
  skip-if-present semantics, `--force` behavior, marker insertion rules
- **User-facing behavior change**: consumer projects' `commands/gaze.md`
  now auto-updates on `gaze init`. Genuinely customized copies are
  overwritten (git-recoverable); the escape hatch and update-time tip
  are the preservation path. A plain `gaze init` re-run on an up-to-date
  project no longer prints the `--force` hint (no user-owned
  `.opencode/` file can be skipped anymore)
- **Risk**: moderate. Breaking only for customized copies, which the
  survey found zero of; mitigated by git recoverability, the documented
  escape hatch, and the update-time tip
- **Gate constraints**: `scaffold.Run` sits at the dogfood GazeCRAPload
  gate ceiling; the change is structured so `Run` gains no branches and
  the dogfood thresholds (38/6/8) must be verified green
- **Website documentation gate**: user-facing `gaze init` behavior
  change — a `docs:` issue in `unbound-force/website` is filed during
  implementation, before or alongside the review council, per AGENTS.md
- **PR targeting**: the PR targets upstream `unbound-force/gaze`
  (origin is the jflowers fork)

## Constitution Alignment

Assessed against the Gaze constitution v1.3.0
(`.specify/memory/constitution.md`).

### I. Accuracy

**Assessment**: PASS

The scaffolded `/gaze` command now matches the shipped asset exactly
instead of silently drifting from it. Version-marker provenance becomes
truthful again: the duplicate `dev`/`v1.8.0` markers found in the survey
are healed to a single current marker on the first update. The init
summary reports the update, so the delivered change is observable.

### II. Minimal Assumptions

**Assessment**: PASS

No new host assumptions. The escape hatch is a filename convention
requiring no configuration, flag, or marker — it works because the asset
walk structurally never writes non-embedded paths. Users who do nothing
receive correct, current behavior; users who customize opt out with a
rename.

### III. Actionable Output

**Assessment**: PASS

Every observable outcome guides the user: the `updated:` summary line
names the file that changed, and the update-time tip names the exact
path (`commands/gaze-local.md`) that preserves customizations. The
stale-`/gaze` failure mode — silently missing report-mode features — is
eliminated.

### IV. Testability

**Assessment**: PASS

The ownership decision is a pure function (`isToolOwned`) with
table-driven tests. The update, marker-healing, escape-hatch, and tip
behaviors are all covered by isolated `t.TempDir` end-to-end tests that
run under `-race -count=1` with no external services.
