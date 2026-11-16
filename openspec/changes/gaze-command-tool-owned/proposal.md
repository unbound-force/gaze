## Why

Issue #353: `gaze init` treats `commands/gaze.md` as user-owned — written
once, skipped forever after. Every other file gaze ships is tool-owned and
updated when its content changes on a new release. That split leaves the
shipped `/gaze` command frozen at whatever release first scaffolded it.

The issue's audit of the 10 repos running gaze found no local customization
of this file anywhere — every difference from the v2.4.0 asset is
undelivered upstream change:

| Repo | Lines differing from v2.4.0 | Markers carried |
|---|---|---|
| unbound-force/unbound-force | 15 | v1.2.7 |
| unbound-force/website | 15 | v1.4.9 |
| unbound-force/dewey | 13 | v1.8.0 |
| unbound-force/replicator | 13 | v1.8.0-rc.1 |
| zero-dot-force/vibe-check | 13 | dev |
| zero-dot-force/reading-stone | 2 | dev and v1.8.0 |
| zero-dot-force/snake-eyes | 2 | dev and v1.8.0 |
| fullsend-ai/.uf/agents | 2 | dev and v1.8.0 |
| fullsend-ai/.uf/fullsend | 2 | dev and v1.8.0 |

The stale copies are missing behavior the current agent already has:

- The `<protect>` block from #221, so DCP can compress the command away.
- The spec 040 description — the command still says the agent "interprets
  the JSON output" while v2.4.0's gaze-reporter shows the rendered report
  and adds judgment. The command and the agent it delegates to now
  disagree.
- Analyzer and language detection (`.gaze.yaml`, then heuristic) in the
  description.

Several copies also carry two scaffold markers (`dev` and `v1.8.0`)
because the #279 marker-dedup fix only strips old markers when gaze
rewrites a file, and this file is never rewritten.

Today the only way to get the current `/gaze` is `gaze init --force`,
which also clobbers every other user-owned file (the CI workflow) — a
blunt instrument for a routine upgrade.

## What Changes

### `commands/gaze.md` becomes tool-owned

`isToolOwned` (`internal/scaffold/scaffold.go`) gains
`commands/gaze.md`. On a plain `gaze init` re-run, an existing
`commands/gaze.md` whose content differs from the embedded asset is
overwritten and reported as `updated: ... (content changed)`, with
version-marker dedup applied (exactly one marker, carrying the current
version). An identical copy is skipped. This is the same
overwrite-on-diff behavior `commands/gaze-fix.md` and the agents already
have — no new mechanism is introduced.

### Documented trade-off and escape hatch

Local customizations of the shipped `/gaze` are overwritten. Evidence
says nobody has any; users who genuinely need a custom `/gaze` should
keep it in a separately-named command (e.g. `commands/gaze-local.md`),
which gaze never touches. This guidance replaces the current
"customize this file" advice in the docs.

### Observable output change

With no user-owned `.opencode/` asset remaining, the
`N file(s) skipped (use --force to overwrite)` hint no longer appears on
plain `gaze init` re-runs with identical content. The hint's semantics
are preserved — it still fires when the genuinely user-owned CI workflow
(`gaze init --ci`) is skipped — it just no longer fires on plain runs.

### No asset content changes

The embedded `internal/scaffold/assets/commands/gaze.md` and its
`.opencode/commands/gaze.md` mirror are already current (byte-identical,
`TestEmbeddedAssetsMatchSource` green). The drift lives in consumer
repos and self-heals on their next plain `gaze init` after this ships —
including the duplicate-marker collapse, since the rewritten file passes
through `removeExistingMarker`.

## Capabilities

### New Capabilities

- `scaffold-file-ownership`: the `.opencode/` asset ownership model —
  which scaffolded files gaze auto-updates on re-run (tool-owned) versus
  leaves untouched (user-owned). Currently implicit in code; this change
  gives it an explicit spec with `commands/gaze.md` moved to tool-owned.

### Modified Capabilities

- None. (`scaffold-file-ownership` has no prior spec to modify; the
  capability is newly recorded with its post-change end state.)

### Removed Capabilities

- None.

## Impact

### Files Modified

- `internal/scaffold/scaffold.go` — `isToolOwned` switch gains
  `commands/gaze.md`; GoDoc on `isToolOwned` and the ownership
  description updated (no user-owned `.opencode/` asset remains)
- `internal/scaffold/scaffold_test.go` — `TestIsToolOwned` expectation
  flip; `TestRun_SkipsExisting` `--force`-hint assertion flips (hint
  absent on plain identical re-run); `TestRun_ReScaffoldNonForceSingleMarker`
  dead user-owned branch restructured; new tests for update-on-diff,
  marker dedup, and retained user-owned skip path
- `docs/guides/opencode-integration.md` — ownership table row, ownership
  prose, escape-hatch guidance, Upgrading walkthrough sample output
- `docs/reference/cli/init.md` — `--force` flag description, directory
  tree annotation, ownership bullet, re-initialize prose
- Website documentation issue in `unbound-force/website` (user-facing
  `gaze init` behavior change affecting the shipped `/gaze` command)

### External Impact

- Consumers running any gaze release get the current `/gaze` on their
  next plain `gaze init` after upgrading — no `--force`, no clobbering
  of the CI workflow. Copies carrying duplicate scaffold markers are
  collapsed to one.
- Consumers with a genuinely customized `commands/gaze.md` (none found
  in the audit, but possible) have it overwritten; the escape hatch is a
  separately-named command file. This is the same trade-off already
  accepted for `agents/gaze-reporter.md` in spec 040.

### Backward Compatibility

- No CLI flags, exit codes, or JSON output change. The scaffold summary
  gains `updated:` lines it previously reported as `skipped:` for this
  one file, and plain-run identical re-runs drop the `--force` hint
  line. The `.opencode/` ↔ embedded-asset byte-identity invariant
  (`TestEmbeddedAssetsMatchSource`) and the 9-file manifest are
  unchanged.
- The 9-file embedded manifest in AGENTS.md File Sync Invariants is
  unchanged — ownership is not part of that invariant.

## Constitution Alignment

Assessed against the Gaze constitution v1.3.0
(`.specify/memory/constitution.md`).

### I. Accuracy

**Assessment**: PASS

The scaffold's version markers and summary lines become accurate
reports of what gaze shipped: a `/gaze` command that disagrees with the
agent it delegates to (spec 040) is a false signal to users and to DCP
(the missing `<protect>` block from #221 means the command cannot be
compressed away, degrading context accuracy). After this change, the
scaffolded command matches the shipped agent on every release. The
duplicate-marker case (two markers claiming two versions) is collapsed
to one truthful marker.

### II. Minimal Assumptions

**Assessment**: PASS

The change removes an assumption rather than adding one: gaze currently
assumes users want to customize `commands/gaze.md`, and the 10-repo
audit in #353 found zero evidence for that assumption. Consumers need
no annotation or restructuring — a plain `gaze init` delivers the
current command. The escape hatch for real customization
(separately-named command file) requires nothing from gaze and is
documented. No supported environment changes behavior beyond receiving
current content.

### III. Actionable Output

**Assessment**: PASS

The scaffold summary remains actionable and becomes more truthful:
stale command files are reported as `updated: ... (content changed)`
instead of silently `skipped (already exists)`, and the
`use --force to overwrite` hint now appears exactly when a genuinely
user-owned file (the CI workflow) is skipped — no longer nagging about
a file that auto-updates. Markers remain provenance metadata
comparable across runs, now with exactly one per file.

### IV. Testability

**Assessment**: PASS

The ownership flip is a pure classification change exercised through
the existing `Run()`-level scaffold tests with `t.TempDir()` targets —
no external services. The retained user-owned skip path, no longer
reachable through `Run()` with the current manifest, gets direct
`processAssetFile` unit-test coverage so the ownership model's other
half stays verified. Coverage strategy is specified in `tasks.md` with
named tests for every scenario in the delta spec.
