# Design: Reduce markdownlint noise from generated and tool-owned artifacts

## Context

Issue #295 (2026-09-30) reported 1,216 non-blocking markdownlint errors across 281 files. Since then, `8ab1fb0` (#279) and `4ae9126` (#306) extended `MARKDOWN_MARKDOWNLINT_FILTER_REGEX_EXCLUDE` to the embedded asset mirrors, `.uf/`, and `openspec/changes/`. The verified current state (MegaLinter run 43775471798, 2026-11-15; megalinter v9.3.0 pinned by `.github/workflows/mega-linter.yml`; markdownlint-cli 0.47.0):

- 973 tracked `*.md` files; 39 in the effective markdownlint scope after the global and linter-scoped exclusions.
- 80 non-blocking errors across 25 files: MD040×36, MD024×30, MD036×6, MD041×4, MD033×3, MD001×1. All MD041/MD033 errors and 2 MD024 + 1 MD001 are in the 4 vendored template files under `openspec/schemas/unbound-force/templates/`; the other 71 are user-facing `docs/` debt.
- 35 additional auto-fixable violations (MD032×14, MD037×10, MD022×6, MD012×3, MD038×2) exist in the tree; CI's `markdownlint --fix` repairs them ephemerally (115 local findings − 35 fixable = 80 reported), but the fixes are discarded on push-to-main runs and never committed on PRs that do not touch those files.
- **Root cause of "warning despite CI succeeding"**: MegaLinter v9.3.0 registers markdownlint with `is_formatter: true`. In `megalinter/Linter.py` (v9.3.0), `disable_errors` defaults to `true` for formatters unless `FORMATTERS_DISABLE_ERRORS: false` or a linter-scoped override (`<MARKDOWN_MARKDOWNLINT>_DISABLE_ERRORS`) is set, and `Linter.name` resolves to `MARKDOWN_MARKDOWNLINT`. No such override exists in `.mega-linter.yml`, so every markdownlint error this repository has ever reported was a non-blocking warning.
- The exclusion regex is unanchored (substring semantics): `openspec/changes/` silently covers `openspec/changes/archive/`; the bare `specs/` fragment covers Speckit `specs/`, `openspec/specs/`, and per-change delta specs — but not `openspec/schemas/`, which is the remaining tool-owned gap.

The proposal's Constitution Alignment (all four principles PASS) motivates the design below: Accuracy (the gate must mean what it claims), Minimal Assumptions (exclusions explicit and linter-scoped), Actionable Output (every remaining error is a fixable file:line; failures are red, not warnings), Testability (locally verifiable without external services).

## Goals / Non-Goals

### Goals

- Classify every Markdown path into one of four ownership buckets and enforce the classification through markdownlint-scoped exclusions with per-fragment rationale.
- Close the one remaining tool-owned exclusion gap (`openspec/schemas/`).
- Make the markdownlint gate actually blocking (`MARKDOWN_MARKDOWNLINT_DISABLE_ERRORS: false`), eliminating the silent warning downgrade.
- Bring the user-owned scope to zero markdownlint errors so the blocking gate starts from a clean, threshold-free baseline.
- Document the policy durably (`.mega-linter.yml` comments, `AGENTS.md`) and prove blocking end-to-end.

### Non-Goals

- No Go source, test, or workflow changes; no changes to `golangci-lint`, `yamllint`, `gitleaks`, or `git_diff` configuration or behavior.
- No changes to the global `FILTER_REGEX_EXCLUDE` (its scope covers other linters).
- No reformatting of generated, vendored, or tool-owned content — excluded paths are excluded, not fixed.
- No wholesale markdownlint rule disables; MD024 is narrowed (`siblings_only`), not turned off.
- No MegaLinter version bump or workflow refactoring (the v9.3.0 pin stays; semantics are documented against it).
- No tracking backlog beyond the fixing commits themselves — the post-change baseline is zero, so there is nothing left to ratchet.

## Decisions

### D1: Four-bucket path classification (the policy table)

| Path | Bucket | Exclusion fragment | Reason |
|---|---|---|---|
| `README.md`, `AGENTS.md`, `CLAUDE.md`, `docs/**` | user-owned | — (linted, blocking) | Author-owned prose; the surface the gate exists to protect |
| `.uf/**` | generated | `\.uf/` | Dewey semantic store; regenerates on every retrospective, hand-fixes not durable |
| `openspec/changes/**` (active + `archive/`) | generated | `openspec/changes/` | Machine-authored workflow artifacts embedding template text; one fragment covers both, so archiving never changes lint treatment |
| `internal/scaffold/assets/`, `internal/aireport/assets/` | generated | `internal/scaffold/assets/`, `internal/aireport/assets/` | Byte-identical embedded mirrors of `.opencode/` sources (`TestEmbeddedAssetsMatchSource`); must never be reformatted independently |
| `testdata/**` | generated | `testdata/` (global, pre-existing) | Golden fixtures |
| `.opencode/**` | tool-owned | `\.opencode/` (also global, pre-existing) | UF/gaze-init scaffolded agents, commands, references |
| `.specify/**` | tool-owned | `\.specify/` | Speckit templates |
| `openspec/schemas/**` | tool-owned (vendored) | `openspec/schemas/` **(new)** | Vendored with the `@fission-ai/openspec` schema; templates intentionally conflict with MD041/MD033/MD024 |
| `specs/**`, `openspec/specs/**` | historical/structured | `specs/` | Speckit point-in-time specs and synced OpenSpec main specs; structured artifacts, not prose |
| any other `*.md` | user-owned (default-open) | — | Unmatched paths are linted |

The default-open row is the load-bearing one: exclusions are an allowlist of noise, not a denylist of scope. New top-level Markdown is linted until someone classifies it. This satisfies the issue's "consistent policy for OpenSpec active changes, archives, schemas, and templates": all four live in excluded buckets via three fragments, each with a stated reason.

### D2: Exclusions stay linter-scoped; exactly one new fragment

The only regex change is adding `openspec/schemas/` to `MARKDOWN_MARKDOWNLINT_FILTER_REGEX_EXCLUDE`. Rationale for not using the global `FILTER_REGEX_EXCLUDE`: it governs every enabled linter, so routing a style-lint cleanup through it would silently remove yamllint and gitleaks coverage from the matched paths (a coverage loss on security-relevant checks). The prior learnings (`lint-governance`, `issue-triage`) and the `fix-duplicate-scaffold-marker` precedent all converge on the linter-scoped variable. The comment block above the regex is rewritten as the condensed D1 table so the rationale lives with the config.

### D3: Explicit blocking via `MARKDOWN_MARKDOWNLINT_DISABLE_ERRORS: false`

The v9.3.0 override chain (verified in source): `MARKDOWN_MARKDOWNLINT_DISABLE_ERRORS_IF_LESS_THAN` (if set) implies blocking-with-tolerance; otherwise `MARKDOWN_MARKDOWNLINT_DISABLE_ERRORS` wins; otherwise the descriptor default applies, and markdownlint's `is_formatter: true` makes that default non-blocking. We set `MARKDOWN_MARKDOWNLINT_DISABLE_ERRORS: false` explicitly rather than `FORMATTERS_DISABLE_ERRORS: false` because the per-linter knob states the intent precisely and cannot change behavior for future formatters added to `ENABLE_LINTERS`. A comment beside the knob cites the formatter default it overrides, so a future MegaLinter version bump re-verifies the semantics rather than deleting a seemingly redundant line.

### D4: Fix-to-zero instead of a count-based ratchet

Considered: `MARKDOWN_MARKDOWNLINT_DISABLE_ERRORS_IF_LESS_THAN: 72` (71 post-exclusion user-owned errors + 1) — MegaLinter's native baseline mechanism, and the most literal reading of "establish a baseline". Rejected because of a verified loophole: on pull requests `VALIDATE_ALL_CODEBASE` is false, so markdownlint (v9.3.0, `cli_lint_mode: list_of_files`) lints only changed files and counts only their errors — a PR introducing one new violation reports `total_number_errors = 1 < 72`, returns exit code 0, and passes. The ratchet would only trip on the next push-to-main full run, turning main red after the PR looked green — the worst possible failure mode for a gate. With the inventory fully tractable (D5–D6), fixing to zero and blocking outright is strictly stronger: "blocking for newly introduced violations in covered files" holds on PRs and on main, with no magic number to maintain. The zero-error baseline is recorded in this document and in the fixing commit; the AGENTS.md convention keeps it at zero.

### D5: Per-rule remediation strategy for the 71 user-owned findings

- **MD040 ×36** (bare fenced code blocks across 19 files): add an info string — `text` for command output and examples, `bash`/`yaml`/`json`/`go` where the fence content is that language. Mechanical; no anchor impact.
- **MD024 ×28** (duplicate headings in `docs/guides/ai-reports.md` ×11, `docs/guides/improving-scores.md` ×9, `docs/architecture/extending.md` ×4, `docs/reference/json-schemas.md` ×4): every instance is the same section name repeated under *different* parents (per-adapter "Prerequisites"/"Usage", per-strategy "When It Applies", per-schema "Top-Level Structure"). Configure `MD024: {"siblings_only": true}` in `.markdownlint.json`, which still flags accidental duplicates among siblings — the case MD024 exists for — while allowing the intentional cross-parent reference-doc pattern. Alternative (unique-ify all 28 headings) rejected: anchor churn across linked docs for no reader benefit. The rationale is documented in `.mega-linter.yml` and `AGENTS.md` because strict JSON cannot carry comments.
- **MD036 ×6** (`docs/concepts/quality.md` "Pass 1 — …" bold pseudo-headings): convert to real `####` headings; this adds anchors where none existed, so no existing link can break.
- **MD001 ×1** (`docs/reference/glossary.md:7` h1→h3 jump): correct the level; heading text (and therefore its anchor) is unchanged.
- **35 auto-fixable** (MD032/MD037/MD022/MD012/MD038): one scoped `markdownlint --fix` pass (D6) commits what CI's ephemeral fix pass already computes, so local runs match CI and the debt stops lurking invisibly.

### D6: Fix passes are scoped to user-owned files, never repo-wide

`markdownlint --fix` over the whole tree would reformat exactly one side of the `.opencode/` ↔ `internal/*/assets/` mirror pairs and break `TestEmbeddedAssetsMatchSource` — the root cause of the earlier scaffold-marker drift. The fix pass therefore runs with an explicit file list: `README.md AGENTS.md CLAUDE.md` plus `git ls-files 'docs/*.md'`. The 4 vendored template files are excluded by D2 before any fix pass runs, and mirror trees are excluded by the pre-existing fragments.

### D7: Policy documentation homes

The durable policy lives in three places with distinct audiences: `.mega-linter.yml` (fragment-by-fragment bucket + reason, adjacent to the regex it explains — the reviewer's view), `AGENTS.md` MegaLinter CI section (the four-bucket summary, the blocking semantics, and the convention that new generated/tool-owned Markdown paths require a documented exclusion decision before merge — the contributor's view), and this document (the full decision record with the current-state numbers — the historian's view). Per Spec Artifact Immutability, this design.md is a point-in-time record; the config comments and AGENTS.md are the living copies.

### D8: Verification strategy

1. **Local zero-error proof**: run markdownlint (locally available `markdownlint-cli2` v0.24.1; rule parity with CI's markdownlint-cli 0.47.0 verified on the unfixable set — both report identical MD040/MD024/MD036/MD041/MD033/MD001 counts) over the explicit user-owned file list; expect zero findings.
2. **CI parity**: `go build ./cmd/gaze`, `go vet ./...`, `go test -race -count=1 -short ./...` (includes `TestEmbeddedAssetsMatchSource`), `golangci-lint run` — no Go changes are expected; the run proves no collateral damage and validates that lint fixes never touched mirror trees.
3. **Blocking probe**: introduce a temporary non-auto-fixable violation (MD040) in a user-owned docs file and run MegaLinter — locally via `npx mega-linter-runner@9.3.0 --flavor go` where Docker is available, otherwise on a throwaway draft PR — and observe the check fail; then revert. A non-auto-fixable rule is deliberate so `APPLY_FIXES` cannot mask the probe.
4. **Post-merge watch**: the first push-to-main full-codebase run must report markdownlint with zero errors and no "non blocking" wording.

Constitution alignment carried through the design: **Observable Quality** — the gate produces a machine-parseable, unambiguous red/green signal with provenance (the config comments); **Autonomous Collaboration** — the policy is an artifact (the classification table) that humans and agents share rather than tribal knowledge; **Composability First** — the per-linter knob and linter-scoped regex keep the markdownlint policy from coupling to any other check; **Testability** — every requirement is verifiable locally without external services.

## Risks / Trade-offs

- **The blocking knob turns invisible failures red.** If any user-owned violation is missed, the post-merge main run fails. Mitigations: the local zero-error proof over the enumerated file list (D8.1) runs before merge; the implementing PR itself modifies the doc files, so PR-mode MegaLinter lints exactly the files that changed. Residual risk is a rule-behavior difference between local cli2 and CI's markdownlint-cli — bounded by the verified rule-count parity.
- **MD024 `siblings_only` weakens cross-parent duplicate detection.** Accepted: all 28 flagged instances are intentional cross-parent structure; accidental sibling duplicates — the common regression — remain caught. The adjustment is documented and reversible in one line.
- **Auto-fix may alter rendered examples.** The fixable rules only insert blank lines and fix emphasis/spacing; fenced example content is untouched. The PR diff review covers this; golden fixtures under `testdata/` are excluded and untouched.
- **MegaLinter version drift.** The blocking semantics are verified against v9.3.0 source and cited in the config comment; a future pin bump should re-verify the `disable_errors` default chain (the `MARKDOWN_MARKDOWNLINT_DISABLE_ERRORS` variable is stable across versions and is the documented interface).
- **Docs churn in a release-plan phase.** ~22 files change, all formatting-only (fence info strings, blank lines, heading levels, six new `####` anchors that nothing links to). No semantic content changes; no website issue required.

<!-- scaffolded by uf vdev -->
