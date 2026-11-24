---
tag: release-pipeline
author: jay-flowers
category: gotcha
created_at: 2026-11-23T20:08:22Z
identity: release-pipeline-20261123T200822-jay-flowers
tier: draft
as_of: "gaze@97ed8300c4b544d6828685069c7dd5c3babdf72d"
---

When migrating a release pipeline to an external reusable workflow (e.g., jflowers/ship's go-release.yml), inputs that affect release behavior can silently disappear in the rewrite. In this case `allow_prerelease: true` was dropped when the org-infra preflight call was replaced. Because the caller/callee contract is not validated by the caller's own string-matching test and the pipeline is not exercised until the next real release, behavior regressions (prerelease support, input/secret name mismatches) stay latent. SRE review should diff every input/secret of the old call against the new call and require either a dry-run/canary or an explicit confirmation of the callee's defaults before approving a release-pipeline delegation.
