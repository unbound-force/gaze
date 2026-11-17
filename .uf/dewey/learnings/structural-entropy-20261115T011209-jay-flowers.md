---
tag: structural-entropy
author: jay-flowers
category: pattern
created_at: 2026-11-15T01:12:09Z
identity: structural-entropy-20261115T011209-jay-flowers
tier: draft
---

Code-mode entropy measurement of markdown-autolink-entity-bypass (gaze base bff29c8a → PR 742a422e, internal/aireport only) confirmed the specs-mode import-subset rule for a third consecutive time and extended it to a new change shape: adding a data pair to two existing strings.NewReplacer tables plus one new unexported free function (escapeAmp) and test-only additions produced exact zero deltas across all 22 packages (internal/aireport Ca 1→1, Ce 36→36, Instability 0.9730→0.9730, Abstractness 0.08→0.08, Distance 0.0530→0.0530, LCOM 7→7, main-sequence zone both sides; vibe-check diff verdict APPROVE, entropy direction stable). Mechanism: LCOM4 is computed over types, so package-level free functions and replacer-table data never move it, and test-file additions do not enter the package dependency graph at all; with no new imports ΔCe is exactly 0 and ΔCa/ΔAbstractness follow. The check "planned imports ⊆ existing package import set AND no new types → all deltas exactly 0 → verdict APPROVE without hedging" now holds across three change shapes (new file of free functions in redact-env-values-from-adapter-stderr, non-subset import in divisor-gaze-testing which correctly predicted ΔCe +1, and replacer-data + free function here).
