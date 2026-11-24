---
tag: structural-entropy
author: jay-flowers
category: pattern
created_at: 2026-11-14T23:02:27Z
identity: structural-entropy-20261114T230227-jay-flowers
tier: draft
---

Third confirmation of the specs-mode null-delta pattern, in uncommitted-local form (markdown-autolink-entity-bypass, gaze): when the input context reports head_sha == base_sha with uncommitted:true, the entropy delta isolates exactly the working-tree files — analyze the base in a /tmp git worktree at the shared SHA and the PR as the dirty working tree with the same vibe-check binary; a Markdown/YAML-only OpenSpec diff again yields all-packages zero-delta, entropy direction stable, verdict APPROVE (unreliable: false). The package-scope dependency-claim audit (learning structural-entropy-20261029T121754) closes positively with one grep of the target file's import block when the spec's planned code needs only already-imported packages: render_markdown.go already imports strings and defines its escapers as strings.NewReplacer vars, so the planned ampersand-pair additions and plain-cell split require no new import — ΔCe 0 anticipated, no design-precision advisory warranted.
