---
tag: github-actions-security
author: jay-flowers
category: gotcha
created_at: 2026-11-07T17:31:16Z
identity: github-actions-security-20261107T173116-jay-flowers
tier: draft
---

A module-path guard in a reusable workflow's build-from-source path (`go list -m` must equal the tool's module, else fail with ::error::) is a fail-fast UX check, not a trust boundary: any repository can declare another module's path in its own go.mod, so the check cannot establish repository identity. It is safe only while every secret the job sees remains caller-scoped (a spoofing caller can then only exfiltrate its own secret). If the workflow ever gains an org-scoped secret or broader permission, the gu
