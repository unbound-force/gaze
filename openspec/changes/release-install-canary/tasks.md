# Tasks: Release Install Canary

<!--
[P] markers indicate parallelizable tasks:
a) Tasks touch different files.
b) No task depends on another's output.
c) No ordering requirement exists between them.
Tasks touching the same file must NOT carry [P] relative to each other.
-->

## 1. Workflow

- [x] 1.1 Add `.github/workflows/install-canary.yml`: daily `schedule` +
  `workflow_dispatch`; resolve the latest release tag with `gh release view`;
  run `go install github.com/unbound-force/gaze/v2/cmd/gaze@<tag>` and
  `gaze --version`; verify the installed module version via `go version -m`
  (D1–D4).

## 2. Documentation

- [x] 2.1 Update `AGENTS.md`: CI/CD list entry and Recent Changes entry.
  (No website issue: CI/CD pipeline change.)

## 3. Verification and process gates

- [x] 3.1 Constitution alignment per `proposal.md`.
- [x] 3.2 CI parity locally: `actionlint` and `yamllint -c .yamllint.yml` on
  the new workflow; check the `go version -m` parsing against sample output.
- [ ] 3.3 After merge, run the canary via `workflow_dispatch` on `main` and
  confirm it installs the latest release and the version check passes.
