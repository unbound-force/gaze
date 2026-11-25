---
tag: structural-entropy
author: jay-flowers
category: pattern
created_at: 2026-11-02T23:26:42Z
identity: structural-entropy-20261102T232642-jay-flowers
tier: draft
---

On the far-unstable side of the main sequence (Instability > 0.9), a negative ΔAbstractness is usually an improvement, not a regression: Distance = |A + I − 1|, so with I ≈ 0.97 the ideal Abstractness is ≈ 0.03 and any A above it inflates Distance. In the 040-deterministic-ai-report-rendering PR, internal/aireport's Abstractness fell 0.1333→0.0800 while Distance improved 0.1030→0.0530 — reading the Abstractness drop in isolation would have produced a false regression finding. Always read ΔAbstractness jointly with ΔInstability and the resulting ΔDistance before classifying it; report the Abstractness movement as neutral-to-positive when the package sits above the main sequence.
