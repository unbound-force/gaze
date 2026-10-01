<!--
  [P] marks tasks eligible for parallel execution.
  Add [P] when a task: (a) touches different files from
  other [P] tasks in the group, (b) has no dependency
  on prior tasks in the group, (c) can safely execute
  without ordering constraints.
  Do NOT add [P] when tasks modify the same file -
  parallel workers will cause merge conflicts.
  Tasks without [P] run sequentially first, then [P]
  tasks run in parallel.
-->

## 0. Pre-Implementation Gate

- [ ] 0.1 Confirm `proposal.md`, `design.md`, `specs/quality-crap-report-html-output/spec.md`, and `tasks.md` are committed and pushed on `opsx/add-html-output-quality-crap-report` before modifying production or test code.
- [x] 0.2 Report the required outcome, expected changed files, planned new template/formatter/test files, explicit non-goals, and any scope expansion awaiting approval; stop for explicit approval before adding any capability, package, service, protocol, abstraction, generalized test infrastructure, persistent runtime artifact, or follow-up work outside the Statement of Intent.

## 1. Native HTML Formatters

- [ ] 1.1 Add the quality HTML formatter and narrowly embedded template under `internal/quality`, using existing quality models and skipped-test limits to render populated, optional, empty, unavailable, skipped-test, and SSA-degraded states with deterministic contextual escaping and wrapped writer errors.
- [ ] 1.2 Add focused quality formatter tests for complete structure, all required available fields, absent optional sections, empty/degraded diagnostics, adversarial escaping, self-containment, deterministic bytes, and writer failure without snapshots or a generalized harness.
- [ ] 1.3 Add the CRAP HTML formatter and narrowly embedded template under `internal/crap`, supporting normal and baseline-comparison reports, canonical quadrant/remediation ordering, nil metric availability, recommended actions, empty results, and wrapped writer errors.
- [ ] 1.4 Add focused CRAP formatter tests for scores and summaries, unavailable GazeCRAP, empty results, baseline pass/fail and categorized deltas, absent baseline sections, adversarial escaping, self-containment, deterministic bytes, and writer failure.
- [ ] 1.5 Add the native combined HTML formatter and narrowly embedded template under `internal/aireport`, rendering typed headline metrics plus deterministic escaped JSON for each successful pipeline step and recorded errors for failed steps.
- [ ] 1.6 Add focused combined formatter tests for all-success, partial-failure, unavailable metrics, malformed successful step payloads, adversarial errors/data, self-containment, deterministic bytes, and writer failure.

## 2. CLI And Runner Integration

- [ ] 2.1 Update quality and CRAP format validation and dispatch in `cmd/gaze/main.go` to opt into `html`, including empty/degraded quality and normal/comparison CRAP paths, while preserving text/JSON dispatch and threshold or baseline gate ordering.
- [ ] 2.2 Update report preflight and `internal/aireport.Run` dispatch so `html` is explicitly valid, bypasses adapter and prompt setup, writes native HTML, and evaluates existing zero-result and threshold gates unchanged.
- [ ] 2.3 Extend existing command and runner tests to prove HTML selection for quality, CRAP, and report; report operation without `--ai`; comparison and degraded paths; external-analyzer-compatible typed output; invalid-format rejection; and selection of the existing text/JSON writers.

## 3. Documentation

- [ ] 3.1 [P] Update `README.md` examples and output-format summary for quality, CRAP, and native combined HTML reports.
- [ ] 3.2 [P] Update `docs/reference/cli/quality.md` with HTML selection, represented states, self-contained constraints, and a file-output example.
- [ ] 3.3 [P] Update `docs/reference/cli/crap.md` with HTML selection, baseline-comparison behavior, self-contained constraints, and a file-output example.
- [ ] 3.4 [P] Update `docs/reference/cli/report.md` with native HTML selection, no-`--ai` behavior, partial-failure representation, self-contained constraints, and a file-output example.
- [ ] 3.5 Assess `AGENTS.md`, architecture documentation, and exported GoDoc impact; update only documentation directly affected by the implemented behavior and all new exported identifiers.
- [ ] 3.6 Provide the user with a complete title and GitHub-flavored Markdown body for the required `unbound-force/website` tracking issue; the user must create it before the implementing PR is merged because workspace policy prohibits agent-created GitHub issues.

## 4. Verification And Governance

- [ ] 4.1 Read the current `.github/workflows/` files and record the exact local CI-equivalent build, race-enabled test, and lint commands rather than relying on a memorized list.
- [ ] 4.2 Format changed Go files with `gofmt` and `goimports`, then run targeted race-enabled formatter, runner, and command tests with `-count=1`.
- [ ] 4.3 Run every CI-equivalent check discovered in task 4.1; treat any failure as blocking and do not weaken protected gate values.
- [ ] 4.4 Verify every scenario in `specs/quality-crap-report-html-output/spec.md`, including semantic states, contextual escaping, no external resources, deterministic output, and unchanged text/JSON behavior.
- [ ] 4.5 Re-check all four Gaze constitution assessments from `proposal.md`: analysis accuracy is unchanged, no new host-project assumptions or dependencies were added, output remains actionable, and formatters remain isolated and testable without external services.
- [ ] 4.6 Classify every review finding as Required, Optional hardening, or Unrelated; implement only Required findings, provide concise follow-up issue text for Optional findings, and stop for approval if a Required finding would expand the confirmed scope.
- [ ] 4.7 Run the required review council on the final implementation and resolve all Required REQUEST CHANGES findings before PR submission; do not repeat an unchanged review cycle or make substantive changes after approval.
- [ ] 4.8 Before finalization, report the required outcome, actual changed and new files, explicit non-goals, deferred optional findings, and any scope expansion awaiting approval.
<!-- scaffolded by uf v0.17.0 -->
<!-- spec-review: passed -->
