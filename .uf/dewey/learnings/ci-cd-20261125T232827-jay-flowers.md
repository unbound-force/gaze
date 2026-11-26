---
tag: ci-cd
author: jay-flowers
category: gotcha
created_at: 2026-11-25T23:28:27Z
identity: ci-cd-20261125T232827-jay-flowers
tier: draft
as_of: "gaze@a286142e8838a0c84a4780508168eb48c1fbcf04"
---

Bumping a GitHub Actions pin in a repo's own workflows (e.g., a Dependabot PR) does not update embedded scaffold templates that ship those pins to consumers. In gaze, advancing `.github/workflows/test.yml`'s `actions/upload-artifact` pin requires bumping `internal/scaffold/assets/ci/gaze-ai-report.yml`, its golden fixture `internal/scaffold/testdata/golden-gaze-ai-report.yml`, and the literal SHA assertion in `internal/scaffold/ci_test.go` in the same PR. The invariant is stated in the init-ci-scaffold spec ('Actions in the test job MUST carry SHA pins identical to those used in gaze's own CI') and design D3 ('no drift between gaze's CI and what it scaffolds for consumers'). The drift is silent to CI because `ci_test.go` pins the old literal rather than comparing the template against `test.yml` — so it must be caught in review, or better, by adding a live template-vs-`test.yml` pin-parity assertion.
