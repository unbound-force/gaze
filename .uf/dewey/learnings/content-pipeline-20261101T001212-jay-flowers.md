---
tag: content-pipeline
author: jay-flowers
category: pattern
created_at: 2026-11-01T00:12:12Z
identity: content-pipeline-20261101T001212-jay-flowers
tier: draft
---

A go.mod require-block diff that promotes a dependency from indirect to direct is a deterministic trigger for sweeping architecture documentation: the promotion proves a package's dependency set changed, so every doc cell claiming that package's dependencies needs re-verification even when the PR's documentation tasks don't enumerate the file. In the gaze init-ci-scaffold review (2026-10-31), docs/architecture/overview.md's Package Map cell "None (uses embed.FS)" for internal/scaffold was falsified by the golang.org/x/mod indirect→direct promotion plus the new CI-generation responsibility, while tasks 4.x enumerated only README/init.md/ci-integration.md/AGENTS.md. Keying the docs/architecture/ sweep off go.mod churn (plus any new capability assigned to an existing package) is cheaper and more precise than an unconditional grep in every content sweep.
