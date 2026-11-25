---
tag: release-pipeline
author: jay-flowers
category: gotcha
created_at: 2026-11-07T17:15:25Z
identity: release-pipeline-20261107T171525-jay-flowers
tier: draft
---

Switching a repository's dogfood caller of its own reusable workflow to a build-from-source path silently removes the only first-party CI exercise of the workflow's default release-install path: lines added to that path in the same change (e.g., post-install `gaze --version` echoes) ship unexecuted and first run in downstream consumer pipelines, only after the next tagged release and a consumer pin bump. Design docs may accept this trade-off, but the acceptance leaves the release-pipeline checklist item 'Is there a smoke test or post-release verification step?' unmet for the consumer default path. The economical closure is to fold a canary into already-tracked post-release verification rather than inventing new scope in the dogfood PR: gaze #297's post-release smoke (install the published tag through proxy.golang.org into temporary caches, verify with `go version -m` / `gaze --version`, read-only non-blocking job) or a scheduled workflow that calls the reusable workflow with default inputs either restore proactive coverage. Verified on gaze PR 317 feedback item 5 (thread PRRT_kwDORU7ATs604): design.md Risks accepted the gap; the reviewer's canary suggestion mapped onto the already-tracked #297 post-release smoke scope (cf. learning/release-verification-20261101T185210-jay-flowers, which notes that incidental consumption of a release is not regression protection for its publication).
