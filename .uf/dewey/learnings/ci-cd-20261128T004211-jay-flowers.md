---
tag: ci-cd
author: jay-flowers
category: gotcha
created_at: 2026-11-28T00:42:11Z
identity: ci-cd-20261128T004211-jay-flowers
tier: draft
as_of: "gaze@57756bf1ba9862946b32a9ad9784d4f35e3183e6"
---

Pinning a shared CI-gating reusable workflow (e.g. jflowers/ship's pr-scope.yml) to an unreleased main-tip commit while leaving it inside a Dependabot own-org auto-merge group means a bot can silently change which CI jobs run for every PR. The release pipeline's own pin convention (a released tag such as `# v0.10.0`) gives an auditable artifact; a `# main, after vX.Y.Z` SHA does not, and Dependabot's ability to track a tag-less SHA is uncertain. Prefer pinning the classifier to a released tag, or treat the main-tip pin as transient with a plan to move it. Also verify the push-to-main path separately: PR CI runs the head's workflow files and never exercises the callee's push behavior, which is exactly what the release preflight (requiring Unit + Integration / MegaLinter on main HEAD) depends on.
