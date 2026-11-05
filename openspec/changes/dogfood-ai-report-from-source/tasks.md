# Tasks: Dogfood AI Report From Source

<!--
[P] markers indicate parallelizable tasks:
a) Tasks touch different files.
b) No task depends on another's output.
c) No ordering requirement exists between them.
Tasks touching the same file must NOT carry [P] relative to each other.
-->

## 1. Workflows

- [x] 1.1 Add the `build-from-source` boolean input to
  `.github/workflows/reusable_gaze_ai_report.yml`, split the install step
  into `Install gaze (release)` / `Install gaze (source)` with the module-path
  guard and `gaze --version` (D2, D3), update the `gaze-version` description
  and the header contract comment.
- [x] 1.2 Pass `build-from-source: true` from the `gaze-ai-report` job in
  `.github/workflows/test.yml` and update its header comment.

## 2. Documentation

- [x] 2.1 [P] Update `docs/guides/ci-integration.md`: inputs table row for
  `build-from-source`, `gaze-version` description, and the note on when the
  reusable workflow gains spec 040 behavior.
- [x] 2.2 [P] Update `AGENTS.md` CI/CD items 1 and 4 and add a Recent
  Changes entry. (No website issue: the input is internal to the gaze
  repository's dogfood job and the consumer-facing default is unchanged.)

## 3. Verification and process gates

- [x] 3.1 Constitution alignment per `proposal.md`.
- [x] 3.2 CI parity locally: `actionlint` and `yamllint` on the modified
  workflows; simulate the source-install script (guard pass on the gaze
  checkout, guard failure on a foreign module).
- [ ] 3.3 After merge, confirm the `Gaze AI Report` step summary on `main`
  shows the spec 040 rendered report and `Install gaze (source)` logged the
  built version. Only then proceed with the follow-up release change.
