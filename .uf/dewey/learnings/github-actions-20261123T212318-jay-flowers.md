---
tag: github-actions
author: jay-flowers
category: gotcha
created_at: 2026-11-23T21:23:18Z
identity: github-actions-20261123T212318-jay-flowers
tier: draft
as_of: "gaze@97ed8300c4b544d6828685069c7dd5c3babdf72d"
---

When migrating a GitHub Actions caller from one reusable workflow to another, audit every explicitly-passed gate input for parity in the new `with:` block (e.g. allow_prerelease, ci_checks). Re-adding a dropped input is not automatically safe: GitHub Actions fails a reusable-workflow call that passes an input the callee does not declare in `on.workflow_call.inputs`, and a third-party callee's schema may be unverifiable from the consuming repo. The safe caller-side remediation for a dropped gate is to record the decision in-repo (a comment at the `with:` block plus a PR-description note stating the intended behavior) and only restore the explicit input after confirming its exact name/schema in the callee's workflow. Prior art: some org release pipelines deliberately exclude prerelease tags from the tag regex (disk-replicator ci-release-preflight design D3), so "prereleases are unsupported" must be verified per repo, not assumed. Observed on gaze PR 362 (release-via-ship): the migration from complytime reusable_release_preflight (allow_prerelease: true) to jflowers/ship go-release.yml silently dropped the input, and ship's default was unverifiable from the consuming repo.
