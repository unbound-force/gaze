---
tag: ci-workflow-testing
author: jay-flowers
category: pattern
created_at: 2026-11-24T21:38:47Z
identity: ci-workflow-testing-20261124T213847-jay-flowers
tier: draft
as_of: "gaze@1212e88ae47744ece19a5f000002fa702c7e919c"
evidence:
  - ".uf/artifacts/dispatch/624f5216-a5f4-4ebb-b4f9-de48798bbaea.json"
---

CI-workflow contract tests are strongest as two layers: textual pins on the parsed YAML (job structure, gate expressions, SHA pins, exact needs sets, byte-identity of duplicated steps) plus behavioral execution of the embedded run script — GitHub expressions substituted, GITHUB_OUTPUT redirected to a temp file — against real throwaway git repositories built with t.TempDir(). Execute only one workflow's copy of a duplicated script and pin the other copy's byte-identity: behavioral coverage then composes across both copies without duplicating the scenario suite. Skip guards on external tools (bash, git, jq) keep the suite portable, and disabling commit.gpgsign in the fixture repo neutralizes global git config.
