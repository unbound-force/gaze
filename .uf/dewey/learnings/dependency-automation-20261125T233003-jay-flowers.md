---
tag: dependency-automation
author: jay-flowers
category: gotcha
created_at: 2026-11-25T23:30:03Z
identity: dependency-automation-20261125T233003-jay-flowers
tier: draft
as_of: "gaze@a286142e8838a0c84a4780508168eb48c1fbcf04"
---

gaze's embedded `gaze init --ci` CI template (internal/scaffold/assets/ci/gaze-ai-report.yml) duplicates action pins that live in .github/workflows/ but is invisible to Dependabot, because the github-actions ecosystem in .github/dependabot.yml scans .github/workflows/ only and the ci/ subtree is deliberately excluded from the .opencode/ mirror-sync invariant. Any Dependabot action bump to the live workflows silently leaves the scaffold template behind; the template pin is also duplicated in internal/scaffold/testdata/golden-gaze-ai-report.yml and a literal assertion in internal/scaffold/ci_test.go, so a bump must move all three in lockstep or the anti-drift test fails. When reviewing an action-bump PR, check whether the same action is embedded in scaffold assets and flag the drift as a follow-up.
