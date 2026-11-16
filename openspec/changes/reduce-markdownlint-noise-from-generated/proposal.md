# Reduce markdownlint noise from generated and tool-owned artifacts

## Why

Issue #295 (filed 2026-09-30) reported MegaLinter showing 1,216 non-blocking markdownlint errors across 281 files — the Markdown check displayed a warning while CI still succeeded. Two interim commits already narrowed the blast radius (`8ab1fb0` for #279 excluded the embedded asset mirrors; `4ae9126` for #306 added `.uf/` and `openspec/changes/`), so the verified current state (MegaLinter run 43775471798, 2026-11-15, megalinter v9.3.0 / markdownlint-cli 0.47.0) is 39 linted files with 80 non-blocking errors across 25 files. What remains is the systematic work the issue asks for: a classified path policy, one missing exclusion, and — the root cause of "warning despite CI succeeding" — enforcement.

That root cause is now identified: MegaLinter v9.3.0 registers markdownlint with `is_formatter: true`, and in `Linter.py` a formatter's `disable_errors` defaults to `true` unless `FORMATTERS_DISABLE_ERRORS: false` or a `MARKDOWN_MARKDOWNLINT_DISABLE_ERRORS` override is configured. The markdownlint check has therefore never actually been blocking in this repository — every error it reported was downgraded to a warning. The issue's acceptance criterion "markdownlint remains blocking for newly introduced violations in covered files" is currently false and must be made true, not merely preserved.

The remaining 80 errors split into two populations with different owners:

- 9 errors in 4 vendored template files under `openspec/schemas/unbound-force/templates/` (MD041, MD033, MD024, MD001) — tool-owned content vendored with the `@fission-ai/openspec` schema whose templates intentionally conflict with those rules. The unanchored `specs/` exclusion fragment covers `openspec/specs/` but, being a substring match, does not cover `openspec/schemas/` — that gap is the one remaining exclusion to close.
- 71 errors in 21 user-owned files under `docs/` (36 MD040, 28 MD024, 6 MD036, 1 MD001) — genuine documentation debt, every instance actionable at a file:line. A further 35 auto-fixable formatting errors (MD032, MD037, MD022, MD012, MD038) exist that CI's `--fix` pass repairs ephemerally and never reports; they persist in the tree because fixes are only committed on PRs and only for changed files.

OpenSpec's structured Markdown profile (RFC 2119 keywords, template placeholders, long proposal/spec tables) intentionally conflicts with generic markdownlint rules such as MD041, MD022, and MD032, so linting those artifacts as author-owned prose is a category error. The prior `fix-duplicate-scaffold-marker` change proved the correct lever: linter-scoped path exclusion with a per-path rationale documented as comments in `.mega-linter.yml`, never reformatting generated content and never widening the global `FILTER_REGEX_EXCLUDE` (which also gates yamllint and gitleaks coverage).

This change is epic #337 Phase 3 (v2.4.0). Phases 1-2 shipped the Tier 1 `markdown-autolink-entity-bypass` bypass fix and `baseline-absolute-path-compat`; this Phase 3 step touches only lint routing configuration, rule scoping, documentation formatting, and documentation of the policy, so it is parallel-safe with other in-flight changes.

## What Changes

### Path classification and markdownlint-scoped exclusions

Classify the repository's Markdown surface into four buckets. Exclusions live in `MARKDOWN_MARKDOWNLINT_FILTER_REGEX_EXCLUDE` only (the global `FILTER_REGEX_EXCLUDE` is untouched), and each fragment carries its bucket and rationale as comments in `.mega-linter.yml`:

- **User-owned (linted, blocking)**: `README.md`, `AGENTS.md`, `CLAUDE.md`, `docs/**` — 35 files at the time of this change. Any path matching no exclusion defaults to this bucket.
- **Generated (excluded)**: `.uf/` (semantic store that regenerates on every retrospective), `openspec/changes/` (machine-authored workflow artifacts, covering both active changes and `archive/` with one fragment), `internal/scaffold/assets/` and `internal/aireport/assets/` (byte-identical embedded mirrors guarded by `TestEmbeddedAssetsMatchSource`), `testdata/` (golden fixtures, excluded globally since the first config).
- **Tool-owned / vendored (excluded)**: `.opencode/` (UF/gaze-init scaffolded agents, commands, and references — globally excluded from all linters since the first config), `.specify/` (Speckit templates), and — the one **new** exclusion in this change — `openspec/schemas/` (templates vendored from `@fission-ai/openspec`).
- **Historical / structured (excluded)**: `specs/` (Speckit point-in-time spec artifacts) and, via the same substring fragment, `openspec/specs/` (synced structured main specs).

### Blocking enforcement

Set `MARKDOWN_MARKDOWNLINT_DISABLE_ERRORS: false` in `.mega-linter.yml`, overriding the formatter default that has silently downgraded every markdownlint error to a warning. With the user-owned scope at zero errors (below), the gate needs no count-based ratchet (`..._DISABLE_ERRORS_IF_LESS_THAN`) — any violation in a covered file fails the check.

### Zero-debt user-owned scope

Fix all markdownlint violations in user-owned files so the blocking gate starts from a clean baseline of zero:

- Apply `markdownlint --fix` scoped to user-owned files only (never repo-wide — mirror trees must stay byte-identical), clearing the 35 auto-fixable formatting errors.
- Add language tags to the 36 MD040 fenced code blocks (`text` for output examples, `bash`/`yaml`/`json`/`go` where the content is code).
- Convert the 6 MD036 emphasis-as-heading lines in `docs/concepts/quality.md` to real headings, and fix the 1 MD001 heading-level jump in `docs/reference/glossary.md`.
- Resolve the 28 MD024 duplicate-heading findings — all of them repeated section names under different parents (per-adapter and per-strategy sections in the guides, per-schema sections in the reference) — by configuring `MD024` with `siblings_only: true` in `.markdownlint.json`, a narrowly scoped adjustment that still catches accidental sibling duplicates. The rationale is documented in `.mega-linter.yml` comments and AGENTS.md because the JSON config cannot carry comments. Wholesale rule disables are not used.

### Policy documentation

Document the policy where it lives: `.mega-linter.yml` carries the fragment-by-fragment rationale (bucket + reason) directly above the regex; AGENTS.md's MegaLinter CI section summarizes the four-bucket policy, the blocking semantics, and the convention that any newly added generated or tool-owned Markdown path requires a documented exclusion decision before merge. This change's `design.md` records the full classification table for the change history.

### No other behavioral change

No Go source, test, agent prompt, or workflow file changes. No other linter changes: `golangci-lint`, `yamllint`, `gitleaks`, and `git_diff` keep identical configuration and blocking behavior, and the global `FILTER_REGEX_EXCLUDE` is untouched so their coverage cannot widen or narrow. Generated mirrors stay byte-identical. `CLAUDE.md` remains linted (one auto-fixable blank-line error); if uf tooling ever regenerates it, the now-blocking gate surfaces that as an actionable finding rather than a silent warning.

## Capabilities

### New Capabilities

- `markdownlint-path-policy`: four-bucket Markdown path classification with documented, markdownlint-scoped exclusions for generated, tool-owned, vendored, and historical paths; default-open scope (unmatched paths are linted); mirror-pair integrity.
- `lint-gating`: the MegaLinter lint gate's enforcement semantics — markdownlint blocking for covered files, zero-error baseline, CI output that distinguishes ignored paths from genuine failures. No main spec exists for this capability today; this change creates it, formalizing previously undocumented gate behavior.

### Modified Capabilities

- None

### Removed Capabilities

- None

## Impact

- **Modified files**: `.mega-linter.yml` (add `openspec/schemas/` exclusion fragment; add `MARKDOWN_MARKDOWNLINT_DISABLE_ERRORS: false`; rewrite the regex comment block as the four-bucket policy), `.markdownlint.json` (add `"MD024": {"siblings_only": true}`), ~21 user-owned documentation files under `docs/` plus `CLAUDE.md` (violation fixes: 36 MD040, 6 MD036, 1 MD001, 28 MD024 resolved via rule scoping, 35 auto-fixable), `AGENTS.md` (MegaLinter CI section: policy summary and convention). Paper trail for gate modifications per AGENTS.md Gatekeeping Value Protection category 4: every lint-configuration change in this change is enumerated here — one exclusion fragment, one blocking knob, one narrowly scoped rule adjustment — each with its rationale; none weakens coverage of user-owned documentation.
- **Risk**: Low-to-moderate, all documentation and configuration. The exclusion is additive and linter-scoped. The blocking knob turns previously invisible failures red — any missed violation would fail CI, mitigated by a local full-scope zero-error verification before merge (the implementing PR itself modifies the doc files, so PR-mode MegaLinter lints them). The MD024 `siblings_only` adjustment trades detection of intentional cross-parent section-name repeats for guide/reference readability; accidental sibling duplicates remain caught. Reverting the exclusion or the knob is a one-line edit each.
- **Test impact**: No Go production code changes; verification is a local markdownlint run over the 35-file user-owned scope producing zero errors, the standard CI-parity Go checks (`go build`, `go vet`, `go test -race -count=1 -short ./...`, `golangci-lint run`) proving no collateral damage (including `TestEmbeddedAssetsMatchSource` mirror integrity), an end-to-end blocking probe (a temporary MD040 violation must fail a MegaLinter run), and a green full-codebase MegaLinter run on main after merge with no "non blocking" wording for markdownlint.
- **Docs / release**: The `docs/` fixes are formatting-only (fence language tags, blank lines, heading levels); no anchors or links change semantics. No website issue required (CI/CD pipeline change and formatting-only docs edits are exempt). Release: epic #337 Phase 3 of v2.4.0.

## Constitution Alignment

Assessed against the Gaze project constitution (`.specify/memory/constitution.md` v1.3.0).

### I. Accuracy

**Assessment**: PASS

Narrowing markdownlint to author-owned paths removes false positives (9 vendored-template violations linted as if they were prose) without masking true positives — the 71 user-facing findings are fixed, not hidden. The change also fixes an accuracy defect in the gate itself: the check claimed to lint Markdown while silently downgrading every error to a warning. Afterwards, "the Markdown check passed" means "author-owned Markdown is clean," backed by a zero-error baseline rather than a tolerated count.

### II. Minimal Assumptions

**Assessment**: PASS

No new host assumptions: no source annotation, no restructuring of supported Markdown, no new dependencies. `openspec/`, `.uf/`, `.specify/`, and `.opencode/` paths are explicit, documented, and enforced at the lint entry point via a linter-scoped regex — never silently ignored and never excluded globally where the exclusion would also affect yamllint or gitleaks coverage. Already-supported environments (`docs/`, `README.md`, `AGENTS.md`, `CLAUDE.md`) keep identical or stronger lint behavior.

### III. Actionable Output

**Assessment**: PASS

After the change, every markdownlint error maps to a concrete file and line in author-owned documentation, and any such error fails the check instead of hiding in a warning banner. CI output remains machine-parseable and comparable across runs: the markdownlint file count equals the user-owned scope, excluded paths are documented in the config adjacent to the regex, and MegaLinter's per-linter summary continues to report counts. The 71 fixed findings are the actionable remediation the issue asked to be "actionable and tracked"; the durable record is the fixing commit plus the policy documentation.

### IV. Testability

**Assessment**: PASS

The gate is verifiable without external services: a local markdownlint run over the enumerated user-owned file set must produce zero errors; the blocking behavior is verifiable end-to-end by introducing a temporary non-auto-fixable violation and observing the MegaLinter check fail (locally via `mega-linter-runner` or on a throwaway draft PR); and mirror integrity is guarded by the existing `TestEmbeddedAssetsMatchSource` sync test in the standard CI test suite. The plan specifies this coverage strategy explicitly and the tasks include the verification steps.

<!-- scaffolded by uf vdev -->
