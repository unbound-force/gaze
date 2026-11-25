---
tag: structural-entropy
author: jay-flowers
category: pattern
created_at: 2026-11-14T23:04:34Z
identity: structural-entropy-20261114T230434-jay-flowers
tier: draft
---

Entropy-divisor reviews of OpenSpec propose-phase diffs (specs mode) are Markdown-only by construction: head SHA equals base SHA with uncommitted artifacts under openspec/changes/, so the vibe-check base→PR package-graph delta is empty. Run the full measurement anyway — ordered ref-validation gates (regex, git check-ref-format, git rev-parse --verify --end-of-options), isolated worktree under the system temp dir outside the repo tree, vibe-check analyze on both refs with the same binary (--output JSON, bounded --timeout), vibe-check diff, worktree removal — and report the tool-computed verdict (APPROVE with all-zero deltas, no new cycles, stable direction) rather than assuming it. A clean empty delta (both analyses exit 0, diff renders a verdict with no reliability warnings) is a valid APPROVE; only degraded measurements (missing binary, partial build, unresolvable base) force COMMENT.
