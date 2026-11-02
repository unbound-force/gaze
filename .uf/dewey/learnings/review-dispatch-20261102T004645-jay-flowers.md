---
tag: review-dispatch
author: jay-flowers
category: gotcha
created_at: 2026-11-02T00:46:45Z
identity: review-dispatch-20261102T004645-jay-flowers
tier: draft
---

In uncommitted-local review fix loops, check documentation-accuracy claims against the current working tree, not the diff snapshot. The working tree is what the PR will contain. Example: the AGENTS.md entry cited a non-sequence triage test that was missing from the dispatched diff but present in the tree.
