<!--
  [P] marks tasks eligible for parallel execution.
  Add [P] when a task: (a) touches different files from
  other [P] tasks in the same group, (b) has no dependency
  on prior tasks in the group, (c) can safely execute
  without ordering constraints.
  Do NOT add [P] when tasks modify the same file —
  parallel workers will cause merge conflicts.
  Tasks without [P] run sequentially first, then [P]
  tasks run in parallel.
-->

## 1. Lint configuration

- [ ] 1.1 In `.mega-linter.yml`, add the `openspec/schemas/` fragment to `MARKDOWN_MARKDOWNLINT_FILTER_REGEX_EXCLUDE` and rewrite the comment block above the regex as the four-bucket policy (design D1): each fragment labeled with its bucket (user-owned / generated / tool-owned / historical) and a one-line reason hand-fixing is not durable or the content is not author-owned. Do not touch the global `FILTER_REGEX_EXCLUDE`.
- [ ] 1.2 In `.mega-linter.yml`, add `MARKDOWN_MARKDOWNLINT_DISABLE_ERRORS: false` with a comment explaining that MegaLinter v9.3.0 registers markdownlint as a formatter whose errors default to non-blocking warnings, and that this knob makes violations fail the check (design D3). Same file as 1.1 — run after it.
- [ ] 1.3 [P] In `.markdownlint.json`, add `"MD024": {"siblings_only": true}` (strict JSON — no comments; the rationale is recorded in 1.1's comment block and in task 3.1). Keep `MD013: false` and `MD060: false` unchanged; no rule may be newly disabled.

## 2. User-owned documentation debt to zero (71 reported + 35 auto-fixable findings)

- [ ] 2.1 Run the auto-fix pass scoped to user-owned files ONLY — `npx --no-install markdownlint-cli2 --fix README.md AGENTS.md CLAUDE.md $(git ls-files 'docs/*.md')` — clearing the 35 auto-fixable findings (MD032×14, MD037×10, MD022×6, MD012×3, MD038×2). NEVER run repo-wide: `internal/scaffold/assets/` and `internal/aireport/assets/` are byte-identical mirrors of `.opencode/` sources and `TestEmbeddedAssetsMatchSource` must stay green (design D6). Runs before the manual fixes so they apply to the post-fix text.
- [ ] 2.2 [P] In `docs/concepts/quality.md`, fix MD036×6 by converting the bold "Pass 1 — …" through "Pass 5 — …" pseudo-headings to `####` headings, and fix MD040×2 (lines ~130, ~149) by adding fence info strings (`text` for output examples).
- [ ] 2.3 [P] In `docs/reference/glossary.md`, fix MD001×1 (line 7 heading-level jump — correct the level, keeping the heading text and anchor unchanged) and MD040×4 (lines 37, 51, 90, 112) with appropriate info strings.
- [ ] 2.4 [P] Fix the remaining MD040 findings in architecture/concepts/porting docs — `docs/architecture/contributing.md` (×5), `docs/architecture/overview.md` (×2), `docs/concepts/analysis-pipeline.md` (×1), `docs/concepts/scoring.md` (×3), `docs/porting/contracts.md` (×3), `docs/porting/requirements.md` (×1), `docs/porting/taxonomy-reference.md` (×1) — using `text` for output examples and `bash`/`yaml`/`json`/`go` where the fence content is code.
- [ ] 2.5 [P] Fix the remaining MD040 findings in reference docs — `docs/reference/cli/analyze.md` (×2), `docs/reference/cli/crap.md` (×5), `docs/reference/cli/docscan.md` (×1), `docs/reference/cli/quality.md` (×2), `docs/reference/cli/report.md` (×1), `docs/reference/cli/schema.md` (×1), `docs/reference/cli/self-check.md` (×1), `docs/reference/configuration.md` (×1) — same info-string convention.
- [ ] 2.6 Verify the 28 MD024 duplicate-heading findings (`docs/guides/ai-reports.md` ×11, `docs/guides/improving-scores.md` ×9, `docs/architecture/extending.md` ×4, `docs/reference/json-schemas.md` ×4) are resolved by the task 1.3 `siblings_only` configuration; if any true sibling duplicates remain, restructure those specific headings to be unique rather than widening the rule config.
- [ ] 2.7 Full-scope proof: run `npx --no-install markdownlint-cli2 README.md AGENTS.md CLAUDE.md $(git ls-files 'docs/*.md')` with no `--fix` and confirm ZERO findings (design D8.1; local cli2 rule parity with CI markdownlint-cli 0.47.0 verified on the unfixable set).

## 3. Policy documentation

- [ ] 3.1 In `AGENTS.md` (CI/CD section, MegaLinter item), document: the four-bucket markdownlint path policy with the excluded paths summarized; that markdownlint is blocking via `MARKDOWN_MARKDOWNLINT_DISABLE_ERRORS: false` (violations fail the check); the `MD024 siblings_only` rationale; and the convention that any newly added generated or tool-owned Markdown path requires a documented exclusion decision in `.mega-linter.yml` before merge (default-open: unmatched paths are linted).
- [ ] 3.2 Documentation Validation Gate sweep: assess `README.md` (no behavioral change expected — confirm no lint-policy mention needs updating), GoDoc (N/A — no Go changes), and confirm no in-flight spec artifacts outside this change are affected.

## 4. Verification

- [ ] 4.1 CI parity per AGENTS.md Technical Guardrails (commands derived from `.github/workflows/test.yml` and `mega-linter.yml`): `go build ./cmd/gaze`, `go vet ./...`, `go test -race -count=1 -short ./...` (must include a green `TestEmbeddedAssetsMatchSource` proving mirror byte-identity survived the fix passes), and `golangci-lint run`. No Go changes are expected; this proves no collateral damage.
- [ ] 4.2 Validate both modified config files parse: `.mega-linter.yml` (YAML) and `.markdownlint.json` (strict JSON — no comments, no trailing commas).
- [ ] 4.3 Blocking probe (design D8.3): on a throwaway branch, introduce one deliberate MD040 violation (a bare ``` fence) in a user-owned docs file and run MegaLinter — locally via `npx mega-linter-runner@9.3.0 --flavor go` where Docker is available, otherwise via a throwaway draft PR — and confirm the markdownlint check FAILS. Use a non-auto-fixable rule so `APPLY_FIXES` cannot mask the probe. Revert the violation and delete the branch/PR afterwards.
- [ ] 4.4 Constitution alignment verification: confirm each principle against the final state — Accuracy (gate blocking, zero-error baseline, exclusions documented), Minimal Assumptions (exclusions linter-scoped and explicit, no global widening), Actionable Output (every hypothetical error maps to an author-owned file:line and fails red), Testability (local zero-error run and blocking probe both reproducible without external services). Record the result in the implementation summary.
- [ ] 4.5 Post-merge watch: the first push-to-main full-codebase MegaLinter run must report markdownlint with zero errors and no "non blocking error(s)" wording for markdownlint; file a follow-up issue if any error appears.

<!-- scaffolded by uf vdev -->
