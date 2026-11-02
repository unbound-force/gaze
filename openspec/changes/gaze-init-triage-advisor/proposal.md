# Swap divisor-gaze-testing into the triage advisor list

## Why

The `divisor-gaze-testing` change (issue #303) registered the
gaze-specific test-quality persona in `.uf/review-matrix.yaml` by
swapping it for the generic `divisor-testing` in the `code`, `spec`,
and `feedback` advisor lists. It deliberately left the `triage` list
untouched (design D1 coexistence), both in the gaze repository's own
`.uf` config and in the merge `gaze init` applies to consumer
projects (`syncReviewMatrix` in `internal/scaffold/ufconfig.go`).

That gap is now observable. The uf dispatch planner includes every
discovered `divisor-*` agent whose capability scope matches the
review, even when the agent is not listed for the mode. An included
agent that is neither an explicit run nor an advisor for the mode is
planned as a **host-fallback** run. A bulk `/uf.triage-issue` run over
12 gaze issues on 2026-11-01 showed that every panel containing a
host-fallback `divisor-gaze-testing` run (9 of 12) failed Phase 4.6
dispatch finalization with the same four host-run provenance
defects, while the 3 panels where the planner skipped the persona
finalized cleanly. The root cause is a uf review-dispatch defect —
the dispatch core never resolves the host model for `source: host`
runs — filed upstream as
[unbound-force/unbound-force#901](https://github.com/unbound-force/unbound-force/issues/901)
(open; uf PR #898 relaxed only the finalizer's variant requirement
and does not fix host-source resolution).

Until #901 ships, the only way to keep the gaze persona finalizable
in triage is to make it an advisor for that mode. The maintainer
chose to **replace** the generic persona in `triage` as well, so the
advisor lists are consistent across all four modes, accepting that
the generic `divisor-testing` becomes a host-fallback run in triage
whenever the planner includes it — exactly as it already is in
`code`, `spec`, and `feedback`.

## What Changes

1. **Gaze repository config** — `.uf/review-matrix.yaml`
   `advisor.triage` replaces `divisor-testing` with
   `divisor-gaze-testing`. The generic `divisor-testing`
   capabilities entry in `.uf/reviewer-capabilities.yaml` is kept.
2. **`gaze init` registration merge** — `syncReviewMatrix` swaps
   the generic persona out of, and the gaze persona into, the
   `triage` advisor list in addition to `code`, `spec`, and
   `feedback`. A consumer matrix with no `advisor.triage` sequence
   is skipped with the existing warning, consistent with the other
   modes (no partial registration).
3. **Tests** — `internal/scaffold/ufconfig_test.go` assertions that
   required `triage` to retain `divisor-testing` are inverted; the
   idempotency case covers the four-mode end state.
4. **Docs** — `docs/guides/opencode-integration.md`,
   `docs/reference/cli/init.md`, and the AGENTS.md Recent Changes
   entry describe the four-mode swap.

## Capabilities

### Modified Capabilities

- `uf-review-registration`: the idempotent `.uf` registration merge
  swaps the advisor in four modes (`code`, `spec`, `triage`,
  `feedback`) instead of three.
- `testing-persona-agent`: the gaze repository's own `.uf`
  registration swaps the advisor in all four modes.

## Impact

- `internal/scaffold/ufconfig.go`, `internal/scaffold/ufconfig_test.go`
- `.uf/review-matrix.yaml`
- `docs/guides/opencode-integration.md`, `docs/reference/cli/init.md`,
  `AGENTS.md`
- Consumer projects: the next `gaze init` rewrites their
  `.uf/review-matrix.yaml` `triage` list once (comment-preserving,
  idempotent thereafter). User-visible behavior change → website
  documentation issue required.
- Supersedes the `triage`-preservation clauses of the
  `divisor-gaze-testing` change (design D1; `testing-persona-agent`
  requirement "Gaze repository .uf registration"; the
  `uf-review-registration` scenario "Existing .uf config is
  merged"). Those artifacts remain unchanged as point-in-time
  records.
- Archive ordering: this change's `## MODIFIED Requirements` deltas
  chain against requirements the still-active `divisor-gaze-testing`
  change introduced via `## ADDED Requirements`, so
  `divisor-gaze-testing` must archive first — archiving this change
  first would apply its deltas against a nonexistent base
  requirement and let the superseded three-mode text sync to the
  main specs afterward.
