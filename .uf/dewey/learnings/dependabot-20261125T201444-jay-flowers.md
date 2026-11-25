---
tag: dependabot
author: jay-flowers
category: gotcha
created_at: 2026-11-25T20:14:44Z
identity: dependabot-20261125T201444-jay-flowers
tier: draft
as_of: "gaze@296458aaf226406c67d8efca3fe5a85fbe4d425f"
---

Dependabot groups bundle several dependency bumps into one PR, and auto-merge waits on required checks — so a single dependency whose version is pinned by an exact-value guard test (a literal 40-hex SHA compared in a Go/unit test) fails the entire grouped PR. None of the otherwise-fine bumps in that group merge, and the repo gets a weekly perpetually-red PR to close by hand. When adding Dependabot to a repo that already has literal-SHA guard tests, either (a) change the guard tests to assert pin FORMAT (exact workflow path + full 40-hex SHA + a semver version comment) instead of exact values, or (b) exclude/ignore the guarded dependencies touching those pins — otherwise any "tracks releases by itself" claim is false. Note that option (a) modifies a protected governance gate (pinned action SHAs), so it needs explicit human owner authorization; option (b) degrades to the manual lockstep the automation was meant to remove.
