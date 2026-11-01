---
tag: scaffolding
author: jay-flowers
category: pattern
created_at: 2026-10-31T23:35:02Z
identity: scaffolding-20261031T233502-jay-flowers
tier: draft
---

When a scaffold prints console guidance that points at documentation, use absolute URLs (or tool-home references), never the tool repo's relative doc paths — the output lands in the consumer's repository, where relative paths like `docs/guides/ci-integration.md` do not resolve. In the gaze init-ci-scaffold review (2026-10-31), the generated workflow template's provenance header correctly used the absolute GitHub URL while the adjacent console next-steps section used bare repo-relative paths, an inconsistency caught only by reading both artifacts side by side. Audit every user-facing string a scaffold emits for destination-repo resolvability, not just the files it writes.
