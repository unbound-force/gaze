<!--
  Planning boundary: this change was proposed via /opsx-propose.
  Implementation happens in the apply workflow. Reference design
  decisions as D1-D9; convention-pack rules cite the go pack
  (.opencode/uf/packs/go.md).
-->

## 1. Ownership Flip (scaffold core)

Both tasks modify `internal/scaffold/scaffold.go` — sequential, no [P].

- [x] 1.1 Add `"commands/gaze.md"` to the exact-match `switch` in
  `isToolOwned` (`internal/scaffold/scaffold.go`, design D1) and rewrite
  the function's GoDoc comment: remove the "user-owned files (only
  commands/gaze.md today)" claim and the commands/ mixed-ownership
  rationale — after the flip all nine embedded `.opencode/` assets are
  tool-owned and the CI workflow (classified by its dedicated write
  path) is the only user-owned scaffold output. Keep the
  references/-prefix + exact-match list shape (design D1 rejected the
  inverted default). No changes to any file under `.opencode/` or
  `internal/scaffold/assets/` (design D5).
- [x] 1.2 Add the escape-hatch tip to `printSummary` (design D3): while
  iterating `Result.Updated`, set a flag when the path is
  `commands/gaze.md`; after the per-file lists, when the flag is set,
  print exactly one line:
  `Tip: commands/gaze.md is tool-owned; to keep a customized /gaze command, name it commands/gaze-local.md (gaze never writes that path).`
  Gate on `Updated` only — never `Created`, `Skipped`, or `Overwritten`.
  The tip lives in `printSummary` (not `Run`) so `Run` — at the dogfood
  GazeCRAPload ceiling — gains no branches (design D6). GoDoc comment
  explains the gating rationale (CS-004).

## 2. Scaffold Tests

Tasks 2.1-2.7 all modify `internal/scaffold/scaffold_test.go` —
sequential within the group. Task 2.8 touches a different file and is
parallel-eligible.

- [x] 2.1 In the `isToolOwned` table-driven test, flip the
  `{"commands/gaze.md", false}` case to `true` and add
  `{"commands/gaze-local.md", false}` — pinning the escape hatch at the
  pure-function level alongside the existing
  `{"commands/speckit.analyze.md", false}` edge case (TC-006, TC-009).
- [x] 2.2 Add `TestRun_OverwritesCustomizedGazeCommand`, mirroring
  `TestRun_OverwritesCustomizedGazeReporter` (FR-034 precedent):
  scaffold into `t.TempDir()`, overwrite `.opencode/commands/gaze.md`
  with custom content, re-run `Run` without force, assert
  `result.Updated` contains the file and the on-disk content equals the
  embedded asset rendered with the current marker (TC-003, TC-004,
  TC-009 — assert exact content, not just membership).
- [x] 2.3 Add a duplicate-marker healing regression (design D4): pre-
  write `.opencode/commands/gaze.md` containing two
  `<!-- scaffolded by gaze ... -->` lines (e.g. `dev` and `v1.8.0`, the
  survey's observed pair) plus stale body content, run `Run` without
  force, assert the file is updated and the result contains exactly one
  marker carrying the current run's version.
- [x] 2.4 Restructure `TestRun_ReScaffoldNonForceSingleMarker`: post-flip
  every embedded asset takes the tool-owned branch, so the user-owned
  else-branch (dev marker preserved, "must not be updated without
  --force") is dead. Assert all nine scaffolded files carry the new
  version's single marker after the non-force re-run; the
  untouched-user-file scenario moves to the escape-hatch test (2.6).
- [x] 2.5 Update `TestRun_SkipsExisting`: an identical non-force re-run
  still reports 9 skipped files, but the `--force` hint no longer prints
  (no user-owned `.opencode/` file can be skipped) — flip the
  `1 file skipped (use --force to overwrite)` assertion to absence and
  correct the miscounted comment (the old "2 user-owned + 7 tool-owned"
  was already wrong; it is now 0 user-owned + 9 tool-owned). Audit
  `TestRun_ForceOverwrites`, `TestRun_CreatesFiles`,
  `TestRun_OverwriteOnDiff_ToolOwned`, and
  `TestRun_OverwriteOnDiff_SkipsIdentical` for gaze.md-specific
  expectations and update assertions only (no production change).
- [x] 2.6 Add `TestRun_GazeLocalEscapeHatch` (design D2): pre-create a
  customized `.opencode/commands/gaze-local.md` in `t.TempDir()`, run
  `Run` without force, then with force; both times the file's content
  must be byte-identical before and after and absent from every
  `Result` list (`Created`, `Skipped`, `Overwritten`, `Updated`,
  `Removed`).
- [x] 2.7 Add `TestRun_GazeCommandUpdateTip` (design D3): the tip line
  (assert the stable substring `gaze-local.md`) prints on stdout when
  `commands/gaze.md` is updated; does not print when the file is
  skipped as identical; does not print on first-run creation; does not
  print when overwritten via `--force`. Assert stdout content exactly
  (TC-009).
- [x] 2.8 [P] Audit `cmd/gaze/main_test.go` init tests (the
  expected-files list around line 2984 and any init summary assertions)
  for gaze.md ownership or `--force` hint expectations; update
  assertions to match the flipped behavior. No `cmd/gaze/main.go`
  production change is expected — the init command delegates to
  `scaffold.Run` (AP-002/AP-003).

## 3. Documentation

All [P] — each task touches a different file. Derive coverage from the
design D7 grep sweep, not just the obvious tables.

- [x] 3.1 [P] Update `docs/guides/opencode-integration.md`: flip the
  Commands ownership table row (`commands/gaze.md` → Tool-owned);
  rewrite the "Tool-Owned vs User-Owned Files" section — all nine
  `.opencode/` assets are tool-owned, the CI workflow is the only
  user-owned scaffold output, and `commands/gaze-local.md` (any
  non-embedded command path) is the documented escape hatch that gaze
  never touches, even under `--force`; update the Upgrading section —
  sample output shows `updated: .opencode/commands/gaze.md (content
  changed)` plus the tip line, and a migration note for customized
  copies (rename to `commands/gaze-local.md` before upgrading, or
  recover the prior version from git and rename it after).
- [x] 3.2 [P] Update `docs/reference/cli/init.md`: the `--force` flag
  row (user-owned now means the CI workflow only), the scaffolded-files
  tree comment (`# /gaze command definition (tool-owned)`), the
  user-owned bullet (rewrite to the escape-hatch convention), the "On
  subsequent runs" paragraph (all `.opencode/` files update on diff),
  and any sample init output showing the gaze.md skip or hint line.
  Leave the CI workflow's user-owned bullet (line ~68) unchanged.
- [x] 3.3 [P] Add the AGENTS.md Recent Changes entry for
  `gaze-command-tool-owned` (cite issue #353, the survey evidence, the
  escape hatch, the website issue number once filed, and this OpenSpec
  change path). Confirm in the same pass that the File Sync Invariants
  table and the nine-file manifest are unchanged (no asset content was
  edited — design D5), and that `docs/guides/ci-integration.md` (CI
  workflow user-owned statement) and `README.md` (no ownership claims)
  need no changes per design D7.
- [x] 3.4 [P] File the website documentation issue in
  `unbound-force/website` per the AGENTS.md Website Documentation Gate
  (user-facing `gaze init` behavior change; `docs:` title prefix;
  precedents #345, #352, #366): what changed (`commands/gaze.md` is now
  tool-owned and auto-updates on `gaze init`), why it matters (stale
  `/gaze` commands across releases; the escape hatch), and which pages
  need updating (gaze init / OpenCode integration docs). Search by
  title keywords for duplicates first and verify candidates with
  `gh issue view`. File before or alongside the review council dispatch
  (design D8) so the gate artifact exists at council time.

## 4. Verification

- [x] 4.1 CI parity (AGENTS.md Technical Guardrails): derive the exact
  commands from `.github/workflows/test.yml` and the MegaLinter
  configuration, then run them locally — `go build ./cmd/gaze`,
  `go test -race -count=1 -short ./...`, `golangci-lint run` — and fix
  any failures. All tests must pass under `-race -count=1`
  (TC-005), including `TestEmbeddedAssetsMatchSource`.
- [x] 4.2 Dogfood gate check (design D6): confirm the gaze-ai-report
  dogfood thresholds (max-crapload 38, max-gaze-crapload 6,
  min-contract-coverage 8) stay green — `scaffold.Run` must remain
  under the GazeCRAPload ceiling with the new `printSummary` branch. If
  a gate trips, decompose the affected scaffold function (extract a
  helper); never modify the gate values.
- [x] 4.3 Constitution alignment verification (proposal "Constitution
  Alignment" section): confirm each principle's evidence in the
  implementation — I (the scaffolded `/gaze` matches the shipped asset;
  markers healed to one current version), II (escape hatch is a
  filename convention with no new state or configuration), III (the
  `updated:` line and escape-hatch tip name the exact file and opt-out
  path), IV (pure-function ownership decision plus isolated `t.TempDir`
  tests, all green under `-race`). Record the outcome in the
  implementation summary.
- [x] 4.4 Run `/uf.review-council` per the AGENTS.md PR prerequisite
  and resolve all REQUEST CHANGES findings until all reviewers return
  APPROVE, with the website issue from 3.4 already filed. Make no code
  changes between APPROVE and PR submission; the PR targets upstream
  `unbound-force/gaze` (design D9 — origin is the jflowers fork).
