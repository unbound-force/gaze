## Context

The `divisor-gaze-testing` change (#303) registered the gaze
test-quality persona in `.uf/review-matrix.yaml` by swapping
`divisor-testing` for `divisor-gaze-testing` in the `code`, `spec`,
and `feedback` advisor lists, deliberately leaving `triage`
untouched (its design D1, coexistence). `gaze init` mirrors that
shape into consumer projects through `syncReviewMatrix` in
`internal/scaffold/ufconfig.go`.

The uf dispatch planner includes every discovered `divisor-*`
agent whose capability scopes match the input, whether or not it
is advisor-listed for the mode. An agent that is discovered but
not advisor-listed becomes a `source: host` / `host-fallback`
plan entry. The uf review-dispatch core never resolves a host
model for host-source entries, so those runs persist advisor-style
model provenance that the finalizer's host-run invariants reject
(uf issue unbound-force/unbound-force#901, still open; uf PR #898
only relaxed the parent-variant requirement). In a bulk triage of
12 gaze issues, every panel that included a host-fallback
`divisor-gaze-testing` run (9 of 12) failed Phase 4.6
finalization; the 3 panels that scope-skipped it finalized.

Constitution alignment:

- **Accuracy**: no analysis behavior changes; the change makes
  review-dispatch artifacts finalizable for the gaze persona.
- **Minimal Assumptions**: `gaze init` still never creates `.uf`
  files and still skips malformed or unexpected matrices.
- **Actionable Output**: unchanged console reporting; skip
  warnings name the missing key.
- **Testability**: the swap stays a pure file transform covered
  by table-style unit tests in `ufconfig_test.go`.

## Goals / Non-Goals

### Goals

- `divisor-gaze-testing` is advisor-listed in all four modes
  (`code`, `spec`, `triage`, `feedback`) in the gaze repository
  and in consumer projects registered by `gaze init`.
- The merge stays idempotent and comment-preserving.

### Non-Goals

- Fixing host-source model resolution in uf (tracked upstream in
  unbound-force/unbound-force#901).
- Removing the generic `divisor-testing` capabilities entry or
  agent file.
- Changing the `always` overrides or profile tiers.

## Decisions

### D1: Replace, not coexist, in triage

The triage list swaps `divisor-testing` for
`divisor-gaze-testing`, matching the other three modes. Keeping
both advisor-listed would avoid host-fallback for either persona,
but it diverges from the established one-test-persona-per-mode
shape. The maintainer chose Replace, accepting that
`divisor-testing` becomes a host-fallback candidate in triage
(as it already is in `code`, `spec`, and `feedback`) until uf
fixes #901.

### D2: Missing `advisor.triage` skips the whole file

`syncReviewMatrix` already skips the entire matrix with a warning
when any expected advisor list is missing or not a sequence,
rather than applying a partial registration. Triage joins that
loop with the same strict behavior. Current uf matrices (version
3) always carry a triage list, so the strictness costs nothing in
practice and keeps the registration all-or-nothing.

### D3: Generic capabilities entry preserved

The `divisor-testing` capabilities entry stays. uf's dispatch
plugin special-cases `divisor-testing` for test-review scope
routing, and removing the entry is outside this change's intent.

## Risks / Trade-offs

- **Host-fallback exposure moves to `divisor-testing`**: panels
  that include the generic persona still fail finalization until
  uf #901 ships. Mitigation: the issue is filed and linked; the
  exposure already existed in three of four modes.
- **Consumer matrices without triage**: registration is skipped
  with a named warning. Mitigation: uf has shipped triage in its
  matrix since version 3; users can add the list and re-run.
