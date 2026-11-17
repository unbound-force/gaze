---
tag: structural-entropy
author: jay-flowers
category: pattern
created_at: 2026-11-15T00:38:08Z
identity: structural-entropy-20261115T003808-jay-flowers
tier: draft
---

Third consecutive code-mode confirmation of the internal/aireport zero-delta pattern (markdown-autolink-entity-bypass, base bff29c8a → PR working tree): adding a replacer pair to two existing strings.NewReplacer values (headerEscaper, proseEscaper) plus a new package-level free helper (escapeAmp) and an intra-package composition change (escapeCell layering escapeAmp under escapeCellMarkup) produced exactly zero delta on all six Martin metrics across all 22 packages (internal/aireport: Ca 1→1, Ce 36→36, Instability 0.9730→0.9730, Abstractness 0.08→0.08, Distance 0.0530→0.0530, LCOM 7→7), vibe-check diff verdict APPROVE, entropy direction stable, no cycles, measurement reliable. Extends learning/structural-entropy-20261030T201229-jay-flowers: helper-extraction refactors are also metric-invisible when they stay intra-package and introduce no new imports and no new types — free-function composition never touches type-level LCOM4 or package coupling. The specs-mode import-subset check ("planned imports ⊆ existing package import set → ΔCe exactly 0") has now predicted the code-mode delta exactly three times running.
