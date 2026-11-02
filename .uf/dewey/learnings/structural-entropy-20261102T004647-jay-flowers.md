---
tag: structural-entropy
author: jay-flowers
category: pattern
created_at: 2026-11-02T00:46:47Z
identity: structural-entropy-20261102T004647-jay-flowers
tier: draft
---

For diverged-base uncommitted-local reviews, validate the base ref (regex, then git check-ref-format, then rev-parse --verify) and compute the merge-base. If the merge-base is a third SHA (criss-cross history), measure only HEAD vs the working tree in a detached worktree. Diffing against the diverged base attributes upstream and pre-HEAD commits to the change. For gaze-init-triage-advisor the delta was zero on all 6 Martin metrics across 22 packages.
