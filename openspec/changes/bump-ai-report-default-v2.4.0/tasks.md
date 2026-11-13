# Tasks: Bump AI Report Default to v2.4.0

<!--
[P] markers indicate parallelizable tasks:
a) Tasks touch different files.
b) No task depends on another's output.
c) No ordering requirement exists between them.
Tasks touching the same file must NOT carry [P] relative to each other.
-->

## 1. Workflow

- [x] 1.1 Change the `gaze-version` default in
  `.github/workflows/reusable_gaze_ai_report.yml` to `v2.4.0` and add the
  keep-equal-to-release-tag comment (D1, D2).

## 2. Documentation

- [x] 2.1 [P] Update `docs/guides/ci-integration.md`: inputs table default
  and the spec 040 note.
- [x] 2.2 [P] Add an `AGENTS.md` Recent Changes entry. (No website issue
  beyond the v2.4.0 release notes: the default is documented in the guide.)

## 3. Verification and release

- [x] 3.1 Constitution alignment per `proposal.md`.
- [x] 3.2 `actionlint` and `yamllint` on the modified workflow.
- [ ] 3.3 After merge, tag this merge commit as `v2.4.0` via `release.yml`.
- [ ] 3.4 After release, confirm a caller pinned to `@v2.4.0` without
  `gaze-version` installs v2.4.0, then close #313.
