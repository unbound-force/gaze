## Context

`gaze init` ships nine `.opencode/` assets and classifies each as tool-owned or
user-owned via `isToolOwned` (`internal/scaffold/scaffold.go`): everything under
`references/` plus six exact matches (`commands/speckit.testreview.md`,
`agents/gaze-reporter.md`, `agents/gaze-test-generator.md`,
`agents/divisor-gaze-testing.md`, `commands/gaze-fix.md`, `dcp.jsonc`).
`commands/gaze.md` is the only user-owned `.opencode/` asset — written once on
first scaffold, skipped forever after.

The update path already exists and is content-driven: `processAssetFile`
computes the candidate content (embedded asset bytes + current-version marker
via `insertMarkerAfterFrontmatter`, which internally strips any existing marker
per issue #279), then `handleToolOwnedFile` compares it byte-for-byte with the
on-disk copy — equal means `skipped`, different means wholesale rewrite
reported as `updated`. `printSummary` prints the
`N file(s) skipped (use --force to overwrite)` hint only for skipped files
whose path is **not** tool-owned.

Issue #353's audit of the 10 repos running gaze found zero local customization
of `commands/gaze.md` — every difference from the current asset is undelivered
upstream change (the #221 `<protect>` block, the spec 040 description,
analyzer/language detection), and four repos carry duplicate scaffold markers
(`dev` + `v1.8.0`) because the #279 marker dedup only runs when gaze rewrites
a file, and this file is never rewritten. The embedded asset and its
`.opencode/` mirror are already current and byte-identical
(`TestEmbeddedAssetsMatchSource` green), so no asset content change is needed.

## Goals / Non-Goals

### Goals

- `commands/gaze.md` classified tool-owned, so a plain `gaze init` re-run
  overwrites a stale or locally edited copy with the embedded content and
  reports it as `updated: .opencode/commands/gaze.md (content changed)`.
- Every rewrite of the file — tool-owned update or `--force` — leaves exactly
  one current version marker, collapsing the duplicates observed in #353.
- The `--force` skip hint fires exactly when a genuinely user-owned file was
  skipped (the CI workflow), and no longer on plain identical re-runs.
- Separately-named command files (the customization escape hatch) are
  structurally untouched, with or without `--force`.
- Documentation describes the new ownership and the escape hatch; every
  scenario in the delta spec is covered by a named test.

### Non-Goals

- No embedded asset content changes — `internal/scaffold/assets/commands/gaze.md`
  and `.opencode/commands/gaze.md` stay byte-identical; the 9-file manifest is
  unchanged.
- No changes to marker insertion/dedup logic, the CI workflow's ownership
  (user-owned forever), CLI flags, exit codes, or JSON output.
- No migration or backfill tooling — consumers self-heal on their next plain
  `gaze init` after upgrading.
- No changes to `.uf` registration, orphan cleanup, or anything outside
  `internal/scaffold/` and the two documentation files.
- No in-place customization support for the shipped `/gaze` — the escape hatch
  is a separately-named file, deliberately.

## Decisions

### D1: One-line classification flip in the existing `isToolOwned` switch

Add `"commands/gaze.md"` to the exact-match case list, exactly as
`commands/gaze-fix.md` and the agent files already are — and exactly as spec
040 (FR-034) did for `agents/gaze-reporter.md`. No new mechanism, no new code
path. The classification stays a pure function of the path, independent of
file content, marker state, or local edits, as the delta spec requires.

The `isToolOwned` GoDoc is refreshed: the "user-owned files (only
`commands/gaze.md` today)" note and the mixed-directory rationale are
obsolete. The new rationale keeps the explicit-list design: the switch is an
auditable declaration of which shipped files auto-update, and **unlisted
paths still default to user-owned/skip-if-present**, so a future asset
missing from the list degrades conservatively (skipped, reported) rather than
surprise-overwriting. `Run`'s doc-comment ownership sentences are refreshed
to match (no `.opencode/` asset remains user-owned; the CI workflow is the
user-owned file).

### D2: Update-on-diff and marker machinery inherited unchanged

`processAssetFile` and `handleToolOwnedFile` already implement the required
semantics; the flip routes `commands/gaze.md` into them. Walking the delta
spec's scenarios through the existing mechanics:

- **Stale content / local edits**: on-disk bytes ≠ embedded-plus-current-marker
  → wholesale rewrite → `Result.Updated` → summary line
  `updated: .opencode/commands/gaze.md (content changed)`.
- **Identical copy**: `bytes.Equal` → `skipped` (the file still appears in
  `Result.Skipped` and the `skipped:` summary lines; only the hint changes,
  see D3).
- **Marker-only drift**: a copy whose body matches the asset but whose marker
  carries a different version is not byte-equal to the candidate (which
  carries the current run's marker) → rewritten with the current marker and
  reported as updated.
- **Duplicate markers**: the old copy — one marker or two — differs from the
  candidate, which carries exactly one current marker, so the rewrite
  collapses the duplicates. `insertMarkerAfterFrontmatter`'s internal
  `removeExistingMarker` (#279) additionally guarantees single-marker output
  even if a future embedded asset regressed to shipping a marker.
- **`--force`**: the `exists && !opts.Force` gate bypasses ownership
  entirely (`writeNewFile` → `overwritten`), unconditionally, like every
  other scaffolded file.

The candidate content is derived solely from the embedded asset, so every
rewrite is deterministic, version-pinned, and independent of what was on disk.

### D3: Skip-hint behavior changes by classification, not by code

`printSummary`'s hint counter re-consults `isToolOwned` over `Result.Skipped`,
so no `printSummary` change is needed:

- Plain identical re-run: all nine assets are tool-owned → the counter is
  zero → the hint is absent. (The `Gaze OpenCode integration already up to
  date:` header remains the "everything is current" signal.)
- `gaze init --ci` re-run with an existing workflow: the CI workflow's
  display path (`.github/workflows/gaze-ai-report.yml`) has no `.opencode/`
  prefix, so the path normalization leaves it intact and `isToolOwned`
  returns false → it counts → the hint fires as
  `1 file skipped (use --force to overwrite)`.

Post-change, the hint's semantics become exactly "a file you could only
replace with `--force` was skipped" — which is what it always claimed.

### D4: The escape hatch is structural, not a feature

`Run`'s asset walk iterates only the embedded assets (`fs.WalkDir` over
`assets/`), so any filename gaze does not ship — `commands/gaze-local.md`,
say — is never read, written, or removed. Customizing `/gaze` means keeping
the customization in a separately-named command file; no allowlist, blocklist,
or configuration is introduced. This replaces the current "customize this
file" advice in the docs.

### D5: Test strategy — every delta-spec scenario gets a named test

Existing tests updated:

- `TestIsToolOwned`: flip `commands/gaze.md` to `true`; negative probes for
  unshipped paths (`commands/gaze-local.md`, `commands/speckit.analyze.md`,
  `agents/custom.md`) keep the default-deny branch verified.
- `TestRun_SkipsExisting`: the hint assertion flips to **absence** on the
  plain identical re-run; the skip-breakdown comment is corrected (it
  currently claims "2 user-owned + 7 tool-owned" — the pre-change reality is
  1 user-owned + 8 tool-owned, and post-change 9 tool-owned).
- `TestRun_ReScaffoldNonForceSingleMarker`: the user-owned `else` branch is
  dead post-change (all nine assets are tool-owned); restructure so the
  tool-owned assertions cover `commands/gaze.md` (current marker present, old
  marker replaced).
- `TestRun_OverwriteOnDiff_ToolOwned`: arithmetic is unchanged (2 updated +
  7 skipped) — only the stale count comments are corrected; optionally add
  `commands/gaze.md` to `TestRun_OverwriteOnDiff_SkipsIdentical`'s
  tool-owned enumeration.

New tests (all `Run()`-level with `t.TempDir()`, per the existing patterns):

- `TestRun_UpdatesStaleGazeCommand` — stale content and local-edit variants
  (scenarios 1 and 3 of the first requirement).
- `TestRun_GazeCommandMarkerOnlyDriftRewritten` — marker-only drift rewrites.
- `TestRun_GazeCommandDuplicateMarkersCollapsed` — the #353 shape: two
  markers plus old content, run at `v2.5.0`, exactly one
  `<!-- scaffolded by gaze v2.5.0 -->` marker and content equal to the
  embedded asset with that marker.
- `TestRun_SeparatelyNamedCommandsUntouched` — a `gaze-local.md` survives
  runs with and without `--force` byte-for-byte.
- `TestProcessAssetFile_UserOwnedSkipRetained` — direct unit test of the
  retained user-owned skip branch, which is no longer reachable through
  `Run()` with the current manifest. Without this, the ownership model's
  other half loses coverage the moment the flip lands.
- `TestRun_CI_SkipIfPresent` (in `ci_test.go`) extended with the
  hint assertion for the `--ci` re-run.

Unchanged and required to stay green without modification:
`TestEmbeddedAssetsMatchSource`, `TestAssetPaths_Returns9Files`,
`TestRun_ForceOverwrites` (its 9 overwritten already includes `gaze.md`),
`TestRun_ReScaffoldSingleMarker` (the `--force` marker path now covers
`gaze.md` automatically), `TestProtectTagPlacement`, and the `cmd/gaze`
init tests (verified: they assert first-run creation only).

### D6: Documentation updates

- `docs/guides/opencode-integration.md`: Commands table row flips to
  Tool-owned; the "Tool-Owned vs User-Owned Files" section is rewritten (the
  user-owned `.opencode/` set is empty, the tool-owned list gains
  `commands/gaze.md`, and the escape-hatch paragraph replaces the customize
  advice); the Upgrading walkthrough's prose and sample output move
  `commands/gaze.md` from the `skipped:` line to the `updated:` lines.
- `docs/reference/cli/init.md`: `--force` flag row, directory-tree
  annotation, ownership bullet, and re-initialize prose.
- `README.md` needs no change (verified: it contains no ownership references
  for this file).
- A website documentation issue in `unbound-force/website` is required before
  merge (user-facing `gaze init` behavior change — Website Documentation
  Gate).

### Constitution alignment

The proposal assesses all four Gaze constitution principles as PASS; this
design implements that assessment and adds nothing that revisits it. Against
the org constitution: **Autonomous Collaboration** — the shipped `/gaze`
command is the collaboration artifact between gaze and the host agent, and a
command that disagrees with the agent it delegates to (spec 040) is a
degraded artifact contract; D1/D2 keep it current. **Composability First** —
D4 keeps `gaze init` independently usable with zero consumer configuration.
**Observable Quality** — summary lines and version markers are provenance
output, and D3 makes the hint truthful. **Testability** — D5 exercises the
flip through `Run()`-level tests plus one direct unit test for the branch
`Run()` can no longer reach, with no external services.

## Risks / Trade-offs

- **Local customizations are overwritten, irreversibly.** Accepted: the
  10-repo audit found none; the identical trade-off was accepted for
  `agents/gaze-reporter.md` in spec 040 (FR-034); the escape hatch
  (separately-named command) is documented. No backup mechanism is
  introduced — that would be new scope for a case with zero observed
  instances.
- **The user-owned branch in `processAssetFile` becomes unreachable via
  `Run()`.** Retained deliberately as the default-deny safety default (D1)
  and kept verified by direct unit test (D5) rather than deleted — deleting
  it would remove the conservative failure mode for any future asset missing
  from the list.
- **One-time observable output change.** Consumers' first post-upgrade
  `gaze init` reports `updated:` lines where they previously saw `skipped:`,
  and plain re-runs drop the `--force` hint. Expected and desired; no exit
  code or machine-readable contract changes.
- **Dev-build marker churn.** A dev-built gaze rewrites the marker to
  `dev` on update — identical to every other tool-owned file today, not new
  behavior.
