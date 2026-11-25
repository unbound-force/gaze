## Context

`gaze init` scaffolds nine files into a consumer project's `.opencode/`
tree. `processAssetFile` (`internal/scaffold/scaffold.go`) classifies each
existing file via `isToolOwned(relPath)`: tool-owned files go to
`handleToolOwnedFile` (overwrite-on-diff via `bytes.Equal`), everything
else is skipped unless `--force`. `commands/gaze.md` is the only embedded
asset on the user-owned side — the `isToolOwned` doc comment names it
explicitly — so it is written once and never refreshed. Spec 040 flipped
`agents/gaze-reporter.md` to tool-owned and deliberately kept
`commands/gaze.md` user-owned; the issue #353 survey of 10 consumer repos
shows that reservation now only preserves staleness: every observed diff
against the v2.4.0 asset is undelivered upstream change, and several
copies carry duplicate `dev`/`v1.8.0` version markers.

Mechanics that this design relies on, verified in the current source:

- The marker is inserted into the *embedded* content before comparison
  (`insertMarkerAfterFrontmatter`, which already strips existing markers
  per issue #279), so a tool-owned update rewrites the file wholesale as
  embedded asset + exactly one current marker.
- The asset walk (`fs.WalkDir` over the embedded FS) only ever visits the
  nine embedded paths plus the excluded `ci/` template subtree; no other
  `.opencode/` path is read, written, or removed — including under
  `--force`.
- `printSummary` calls `isToolOwned` a second time, on `Skipped` display
  paths, to count user-owned skips for the `use --force to overwrite`
  hint; the CI workflow (`.github/workflows/gaze-ai-report.yml`) is the
  remaining user-owned scaffold output and still feeds that hint.
- `scaffold.Run` sits at the dogfood GazeCRAPload gate ceiling
  (max-gaze-crapload 6, zero headroom) on gaze's own CI.

## Goals / Non-Goals

### Goals

- `commands/gaze.md` becomes tool-owned: overwrite-on-diff without
  `--force`, reported as `updated`
- A documented, structurally guaranteed escape hatch for customized
  `/gaze` commands (`commands/gaze-local.md` or any non-embedded path)
- Escape-hatch discoverability at the moment of update (one summary tip
  line)
- Duplicate-marker copies healed by the first update
- Documentation matching the new ownership model everywhere it is stated

### Non-Goals

- Changing any embedded asset content (the nine-file byte-identical
  mirror manifest is untouched)
- Changing the CI workflow's user-owned skip-if-present semantics
- Changing `--force` semantics or marker insertion/placement rules
- Scaffolding a `gaze-local.md` template or adding any tenth embedded
  asset
- Detecting whether an existing `gaze.md` is "stale" versus
  "customized" (not reliably possible without shipping historical asset
  versions; see D3)

## Decisions

### D1: Flip via one `isToolOwned` exact-match entry

Add `"commands/gaze.md"` to the existing exact-match `switch` in
`isToolOwned`, and rewrite the function's doc comment: the "user-owned
files (only commands/gaze.md today)" sentence and the "commands/
directory contains both user-owned (gaze.md) and tool-owned files"
rationale no longer hold — after the flip, all nine embedded `.opencode/`
assets are tool-owned and the only user-owned scaffold output is the CI
workflow (classified by its dedicated write path, not this list).

Alternatives considered:

- **Invert the default** (all embedded assets tool-owned, maintain a
  user-owned exception list): smaller future diffs, but it changes
  `isToolOwned`'s contract for arbitrary paths — `printSummary` relies
  on it returning `false` for the CI workflow path and for anything
  outside the embedded set, and an open-ended default weakens that. The
  explicit list keeps each embedded file's classification a deliberate,
  reviewed act.
- **Directory-prefix convention for `commands/`**: rejected — users add
  their own files under `.opencode/commands/` (this repo itself has 30+),
  and a prefix rule would misclassify them if the function is ever
  consulted for non-embedded paths.

### D2: Escape hatch is a convention, not code

The escape hatch needs no production change: the asset walk structurally
never writes a non-embedded path, so `commands/gaze-local.md` (or any
other name) is user-owned by construction — untouched even by `--force`.
OpenCode maps `.opencode/commands/*.md` to slash commands by filename, so
`/gaze-local` works with zero gaze involvement. The nine-file manifest
stays nine; no template is scaffolded that nobody asked for.

Alternative considered: scaffolding a commented-out `gaze-local.md`
template — rejected as manifest growth and noise (Minimal Assumptions:
users who customize already have a file to rename; users who don't
should not get one).

### D3: Update-time tip in `printSummary`, gated on `Updated` only

When `commands/gaze.md` lands in `Result.Updated`, `printSummary` prints
one deterministic line after the per-file lists, naming
`commands/gaze-local.md` as the preservation path (exact wording fixed
in tasks; tests assert a stable `gaze-local.md` substring). Gating:

- **Updated** — yes. This is the surprise moment: a diff (stale or
  customized) was just overwritten. The prior version is in git, so the
  tip is actionable after the fact: recover, rename, keep.
- **Created** — no. Nothing was lost; a fresh adopter does not need an
  opt-out lesson.
- **Overwritten via `--force`** — no. The user explicitly requested the
  overwrite.
- **Skipped identical** — no. Nothing changed.

The tip lives in `printSummary`, not `Run`, because `Run` is the function
at the GazeCRAPload dogfood gate ceiling (D6) and `printSummary` already
owns summary-adjacent guidance (the `--force` hint). Implementation sets
a flag inside the existing `Updated` loop — `Run` gains no branches.

Alternatives considered:

- **Docs-only discoverability**: rejected — the overwrite moment is the
  only time a customizing user is guaranteed to be looking; a guide they
  have not read does not help then.
- **One-time notice** (print only on the first update ever): rejected —
  requires persisted state (a marker or config key), violating Minimal
  Assumptions for a marginal noise reduction. The recurring cost is one
  line, only on runs where the command file actually changed.
- **Pre-overwrite warning with a backup**: rejected — gaze cannot
  distinguish stale from customized copies without shipping historical
  asset versions, and interactive confirmation does not fit `init`'s
  non-interactive model. The survey found zero customized copies; the
  escape hatch plus git recoverability covers the residual risk.

### D4: Marker healing rides the existing update path

`handleToolOwnedFile` writes the freshly rendered content (embedded
asset + single current marker, via `insertMarkerAfterFrontmatter` /
`removeExistingMarker` from issue #279). The first tool-owned update of a
duplicate-marker copy therefore rewrites it into exactly one current
marker with no new code. A regression test pins this (the survey found
real copies with `dev` + `v1.8.0` markers).

### D5: Byte-identity invariant untouched

No file under `.opencode/` or `internal/scaffold/assets/` changes. The
ownership flip is metadata about the write path (`isToolOwned`), not
asset content. `TestEmbeddedAssetsMatchSource`, the AGENTS.md File Sync
Invariants table, and the nine-file manifest are all unaffected —
implementation must confirm this (a dedicated task) so a well-meaning
"also update the asset" edit cannot slip in.

### D6: GazeCRAPload gate discipline

The dogfood gate (max-gaze-crapload 6) has zero headroom on
`scaffold.Run`. The change is shaped to respect it: `isToolOwned` gains a
case constant (no branching), and the tip adds one flag + one
conditional to `printSummary` while `Run` stays branch-free. The
implementation MUST verify the dogfood gate stays green (CI's
`gaze-ai-report` job, thresholds 38/6/8). If any scaffold function trips
a gate, the fix is decomposition (extract a helper), never touching the
gate — per the AGENTS.md gatekeeping-integrity rule.

### D7: Documentation sweep derived from full-text grep

Per the documented repo learning, the sweep is derived from grepping the
docs tree for `gaze.md` / `user-owned`, not from the obvious tables.
Confirmed touch points:

- `docs/guides/opencode-integration.md` — Commands ownership table row
  (`commands/gaze.md` → Tool-owned), the "Tool-Owned vs User-Owned
  Files" section (user-owned paragraph becomes the CI workflow + escape
  hatch; tool-owned enumeration adds `commands/gaze.md`), the Upgrading
  section (sample output now shows the `updated:` line and tip;
  migration note for customized copies: rename to `gaze-local.md`
  before upgrading, or recover from git and rename after)
- `docs/reference/cli/init.md` — `--force` flag row ("existing
  user-owned files are skipped" wording), scaffolded-files tree comment
  (`# /gaze command definition (user-owned)`), the user-owned bullet,
  the "On subsequent runs" paragraph, and any sample init output
- `docs/guides/ci-integration.md` — the CI workflow "user-owned after
  first write" statement stays true; verified, no change
- `README.md` — verified to contain no ownership claims (command table
  row only); no change

### D8: Website documentation issue sequencing

The change is user-facing (`gaze init` behavior), so the AGENTS.md
Website Documentation Gate applies: a `docs:` issue in
`unbound-force/website` must exist before the PR merges. Per the
documented org learning, it is filed before or alongside the review
council dispatch — filing it after leaves a blocking HIGH finding at
council time for a five-minute task. The issue states what changed, why
it matters, and which pages need updating.

### D9: PR targets upstream

`origin` is the jflowers fork; the PR targets `unbound-force/gaze`.
Recorded here so the apply/finale phases route correctly; no code
impact.

## Risks / Trade-offs

- **Customized `/gaze` copies are overwritten on the first post-upgrade
  init.** Accepted: the survey found zero customized copies across 10
  repos; the prior version is git-recoverable; the escape hatch plus
  update-time tip is the permanent opt-out; and the alternative
  (continued staleness) is the defect being fixed.
- **The tip recurs on every release that changes the command asset.**
  Accepted: one line, only when the file actually changed, and genuinely
  relevant each time it prints.
- **`--force` hint disappears from plain-init up-to-date runs** (no
  user-owned `.opencode/` file can be skipped). This is a correctness
  improvement — the hint now only fires for the CI workflow, the one
  file it still describes — but it is an observable output change that
  tests and docs must reflect.
- **Gate headroom.** If `printSummary`'s added branch trips a dogfood
  gate, the mitigation is decomposition (D6), which risks review churn
  but not design change.

## Constitution Alignment

This design implements the proposal's Constitution Alignment
(Gaze constitution v1.3.0): the flip restores Accuracy (scaffolded
command matches the shipped asset; markers become truthful), keeps
Minimal Assumptions (escape hatch is a filename convention, no new
state or configuration), improves Actionable Output (the `updated:`
line and the escape-hatch tip name the exact file and the exact
opt-out path), and preserves Testability (pure-function ownership
decision plus `t.TempDir` end-to-end tests, no external services).
