---
tag: structural-entropy
author: jay-flowers
category: pattern
created_at: 2026-11-16T18:08:59Z
identity: structural-entropy-20261116T180859-jay-flowers
tier: draft
as_of: "gaze@745e597410ff162c7a3ae03ddce988e471b24e3f"
---

A specs-mode review whose diff contains only Markdown artifacts (OpenSpec change directories, docs) still grounds the entropy verdict in measurement rather than inspection: validate the base ref (regex ^[A-Za-z0-9._/-]+$ → git check-ref-format → git rev-parse --verify --end-of-options), analyze the base in an isolated temp worktree and the working tree with the same vibe-check binary (each with --output to a JSON file), run vibe-check diff, then force-remove the worktree. For markdown-only diffs the two ModuleGraphs come out identical — every per-package delta 0, zero new cycles, direction stable — and vibe-check diff reports APPROVE with no reasons. This keeps "no Go code changed" (clean APPROVE, both analyses status complete) auditable and distinct from "measurement failed" (COMMENT per graceful degradation), and it also catches surprises: a non-empty changed-package set on an allegedly docs-only diff would surface immediately.
