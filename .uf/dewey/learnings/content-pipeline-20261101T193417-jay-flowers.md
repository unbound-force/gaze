---
tag: content-pipeline
author: jay-flowers
category: pattern
created_at: 2026-11-01T19:34:17Z
identity: content-pipeline-20261101T193417-jay-flowers
tier: draft
---

In Curator spec-review mode, a missing blog-labeled issue becomes a missing blog-issue TASK (issue creation is prohibited at spec phase): add a Phase 8 task beside the website-docs-issue task to file a separate blog-labeled issue in unbound-force/website at implementation time. Severity MEDIUM because the blog backlog is label-filtered. Detection fingerprint: grep for 'blog'/'tutorial' across spec artifacts returns zero while a docs-issue task is planned.
