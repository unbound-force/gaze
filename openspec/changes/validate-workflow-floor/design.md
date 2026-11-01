# Design: Validate Workflow Floor

## Context

`gaze init --ci` (shipped in v2.3.0) pins the scaffolded workflow's `uses:`
reference to the embedding binary's version and refuses to run when that
version is below `minWorkflowTag` (`v2.3.0`, `internal/scaffold/scaffold.go`),
because only tags ≥ v2.3.0 contain
`.github/workflows/reusable_gaze_ai_report.yml`. The release workflow had no
corresponding check: a below-floor release tag or an unreleased floor bump
would ship a binary that fails its own `--ci` validation. This change adds the
missing release-time gate, modeled on `validate-module-major` (script + release
job + CI test script).

## Goals / Non-Goals

**Goals**

- Fail the release workflow, before preflight creates the tag, when the
  requested tag is below the embedded floor.
- Fail the release workflow when the floor constant references a tag that does
  not exist or does not contain the reusable workflow file.
- Keep the check deterministic, offline, and unskippable.
- Fail CI if the floor constant moves or the reusable workflow file is removed
  from the tree without the validator being updated.

**Non-Goals**

- Validating the v2.3.0 release itself (it predates this gate).
- Changing `--ci` runtime behavior, the floor value, or the scaffold template.
- Supporting floors that are pre-release tags.

## Decisions

### D1: Mirror the `validate-module-major` shape

Script `.github/scripts/validate-workflow-floor.sh` with CLI
`validate-workflow-floor.sh -- <release-tag> [repo-root]` (repo-root defaults
to the working directory), `set -euo pipefail`, `LC_ALL=C`. Exit taxonomy
mirrors the sibling: exit 2 for usage errors, malformed tags, and malformed
repository state (duplicate/malformed/pre-release floor constant,
floor-bearing tree missing the workflow — the same class the sibling assigns
exit 2 for "expected exactly one module directive"); exit 1 for validation
failures. `::error::`/`::notice::` workflow commands print to stdout (the
Actions runner only parses stdout for workflow commands, so redirected
commands never render as annotations); usage and malformed-tag messages are
plain stderr like the sibling; success line to stdout. Release job
`validate-workflow-floor` sits beside `validate-module-major` (same pinned
`actions/checkout` SHA, `contents: read`, 5-minute timeout) and `preflight`
needs both.

*Revision (post-review)*: the original draft redirected `::error::` to stderr
and used exit 1 for repo-state failures; both were corrected after council
review flagged the annotation rendering and the sibling-divergent exit
semantics.

*Rationale*: an established in-repo pattern; reviewers and operators already
know its failure modes.

### D2: Derive the floor from the source constant, never duplicate it

The validator greps `internal/scaffold/scaffold.go` for
`^[[:space:]]*const minWorkflowTag = "vX.Y.Z"`. Zero matches → skip (D4);
exactly one match → use it; multiple matches → fail (ambiguous source of
truth); a value failing the strict tag regex → fail (malformed floor).

*Rationale*: duplicating `v2.3.0` into the script would recreate the exact
drift problem this gate exists to catch. Reading the same constant the binary
embeds makes the verdict correct by construction.

*Post-review revision*: the grep's exit status is now captured explicitly
(matches are written to a temporary file). A grep failure other than "no
match" (exit 1, the legacy-tree signal) — e.g. an unreadable `scaffold.go` —
is a repo-state failure (exit 2) rather than a silent legacy skip, closing
the gate's only fail-open path. Not covered by a matrix case: simulating an
unreadable file portably (chmod 000 is a no-op under root) would add
flakiness without protecting any real workflow path — the release job always
runs against a fresh checkout.

### D3: Explicit bash semver comparison, not `sort -V`

Strip the leading `v`, split the core on `.`, compare major/minor/patch
numerically. Pre-release handling: a tag without a pre-release suffix sorts
above the same core with one (`v2.4.0 > v2.4.0-rc1`); two pre-release suffixes
compare lexically. The tag grammar restricts pre-release suffixes to
non-empty dot-separated identifiers (`-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*`),
mirroring what `x/mod/semver`'s `IsValid` contributes to the Go-side chain —
rejecting malformed identifiers (empty dot-separated segments) that a plain
character class would admit.

*Rationale*: `sort -V` pre-release ordering is implementation-defined and has
burned this kind of gate before. The lexical fallback is correct because
pre-release floors are rejected outright (D4), so at most one operand (the
release tag) can carry a suffix, and the no-suffix-wins rule covers it.
Exotic remaining divergence (leading-zero numeric identifiers, which semver
rejects but the regex admits) is contained downstream by the org-infra
preflight's Python semver validation.

### D4: Legacy trees skip with a notice, floor-bearing trees are held to the contract

If the tree has no `minWorkflowTag` constant, the validator exits 0 with a
`::notice::` — the tree predates floor governance and has no contract to
enforce. If the constant exists, the floor MUST be a canonical release tag:
a malformed value or a pre-release suffix fails as repo-state error (exit 2),
enforcing by construction the invariant D3's comparison relies on. The tree
MUST also contain `.github/workflows/reusable_gaze_ai_report.yml`; a
floor-bearing tree without the workflow file fails unconditionally (exit 2),
before any tag comparison.

*Rationale*: skipping keeps the gate forward-compatible with old tags/hotfix
branches, while the workflow-presence check catches the catastrophic case of
releasing `T == F` from a tree where the workflow was deleted or renamed.

### D5: Above-floor releases must prove the floor tag shipped the workflow

When `T > F`, the validator requires (a) `F` resolves as a local git tag
(`git rev-parse -q --verify "refs/tags/$F"`) and (b)
`git show "$F:.github/workflows/reusable_gaze_ai_report.yml"` succeeds.
Failure of either → exit 1.

*Rationale*: a floor bump to a typo like `v2.30.0`, or a bump merged but whose
floor release was cancelled, would otherwise ship binaries whose `--ci` pins
customers to a nonexistent workflow tag. Verifying tag *contents* (not just
existence) also catches a floor pointing at a pre-workflow tag.

### D6: Offline and unskippable

The release job checks out with `fetch-depth: 0` so all existing tags are
available locally; the validator makes no network or GitHub API calls. No
release skip flag is added for this gate.

*Rationale*: matches `release-major-module-validation` VAL-REQ-002 philosophy
(deterministic, offline, not bypassable by the existing skip inputs). Full
clone is acceptable — this repository is small.

### D7: Repo-invariant test assertions contain the silent-skip path

The CI test script first asserts the repository's own state: the floor
constant exists in `internal/scaffold/scaffold.go`, parses cleanly, and the
reusable workflow file exists in the tree. Only then does it run the
synthetic-repo matrix.

*Rationale*: without these assertions, moving the constant (or deleting the
workflow) would silently convert the repo's own gate into a permanent skip.
The test converts silent skip into loud CI failure, forcing the validator to
be updated in the same change.

### D8: Test matrix via synthetic git repositories

The test script builds throwaway repos in `mktemp -d` (with `trap` cleanup):
writes `internal/scaffold/scaffold.go` with a chosen constant (or none),
optionally writes `.github/workflows/reusable_gaze_ai_report.yml`, commits,
and tags the floor at chosen commits (with or without the workflow file). Each
case invokes the validator and compares exit status, using the
`set +e` / capture / `set -e` pattern from `test-validate-module-major.sh`.

Cases (expected exit): below-floor `v2.2.9` → 1; at-floor `v2.3.0` → 0;
above-floor valid → 0; above-floor, floor tag missing → 1; above-floor, floor
tag predates the workflow → 1; floor pre-release `v2.3.0-rc1` as release tag
→ 1 (mirrors `--ci` rejection of floor pre-releases); later pre-release
`v2.4.0-rc1` → 0; legacy tree without constant → 0 (skip); malformed floor
`"banana"` → 2; duplicate constant → 2; floor constant but no workflow in
tree → 2; malformed release tag `release-2` → 2.

*Post-review revision*: repo-state failures (malformed floor, duplicate
constant, floor-bearing tree missing the workflow, pre-release floor
constant) exit 2, aligning the taxonomy with `validate-module-major.sh`.
The matrix adds two cases for the hardened tag grammar — pre-release floor
constant `v2.4.0-rc.1` → 2 and malformed pre-release tag `v2.4.0-rc..1`
→ 2 — bringing the total to fourteen cases.

## Risks / Trade-offs

- **Floor constant relocates** → validator silently skips (D4). Contained by
  D7 repo-invariant assertions failing CI.
- **Lexical pre-release comparison** could mis-order exotic suffixes (e.g.
  `rc.10` vs `rc.2`). Accepted: floors are canonical releases; release tags
  use simple `-rcN`/none forms here.
- **`fetch-depth: 0`** slightly slower checkout. Accepted for offline
  determinism; repository is small.
- **Gate cannot protect its own introducing release** (v2.3.0 already shipped).
  First exercised on the next release; its above-floor branch then
  retroactively proves v2.3.0 contains the workflow.

## Migration Plan

Purely additive: new script, new test script, one new release job, one test
step, comment/doc updates. No consumer-facing behavior changes; no ordering
constraints with other changes.

## Open Questions

None.
