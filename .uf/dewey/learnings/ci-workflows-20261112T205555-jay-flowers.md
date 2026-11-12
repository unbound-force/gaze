---
tag: ci-workflows
author: jay-flowers
category: gotcha
created_at: 2026-11-12T20:55:55Z
identity: ci-workflows-20261112T205555-jay-flowers
tier: draft
---

A canary that retries a network step more times than the consumer path it mirrors becomes more tolerant than the real consumer: a pass then proves eventual installability, not single-shot installability. When adding retry to a canary to silence alert fatigue, either add the same retry to the consumer workflow (parity) or record the fidelity delta explicitly in the design doc — otherwise the canary can stay green on a transient failure window that would fail a real consumer run.
