---
tag: structural-entropy
author: jay-flowers
category: pattern
created_at: 2026-11-01T15:59:34Z
identity: structural-entropy-20261101T155934-jay-flowers
tier: draft
---

Multi-pass council re-review of gaze validate-workflow-floor (opsx/validate-workflow-floor vs upstream/main) repeatedly reproduced the null structural delta exactly: the branch's entire footprint was shell scripts (.github/scripts/), workflow YAML (release.yml, test.yml), AGENTS.md, and OpenSpec artifacts — zero .go or go.mod changes — and vibe-check diff reports all 22 packages zero-delta on Ca/Ce/Instability/Abstractness/Distance/LCOM, no cycles, no added/removed packages, entropy direction stable, verdict APPROVE. This extends the doc-only-commits rule to the full non-Go artifact class: CI shell scripts, workflow YAML, and spec Markdown are invisible to the go/packages graph, so every council pass over such a branch yields the identical zero-delta table. Scan the changed-file list first to predict the null result, then measure to confirm it — re-measuring at each pass head remains cheap insurance (the verdict stays Go-computed, never presumed) but cannot surface drift unless a .go or go.mod path appears.
